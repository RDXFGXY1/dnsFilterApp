package main

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/securitypacks"
	mdns "github.com/miekg/dns"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = keygen(os.Args[2:])
	case "sign":
		err = sign(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  security-pack keygen -id publisher -out-dir ./keys")
	fmt.Fprintln(os.Stderr, "  security-pack sign -key-id publisher -private ./keys/publisher.private -id threats -name 'Threat Pack' -category malware -domains domains.txt -out threats.json")
	os.Exit(2)
}

func keygen(args []string) error {
	flags := flag.NewFlagSet("keygen", flag.ContinueOnError)
	id := flags.String("id", "", "publisher key id")
	outDir := flags.String("out-dir", ".", "output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !validID(*id) {
		return fmt.Errorf("invalid key id %q", *id)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*outDir, 0700); err != nil {
		return err
	}
	privatePath := filepath.Join(*outDir, *id+".private")
	publicPath := filepath.Join(*outDir, *id+".public")
	if _, err := os.Stat(privatePath); err == nil {
		return fmt.Errorf("refusing to overwrite existing private key %s", privatePath)
	}
	if _, err := os.Stat(publicPath); err == nil {
		return fmt.Errorf("refusing to overwrite existing public key %s", publicPath)
	}
	if err := os.WriteFile(privatePath, []byte(base64.StdEncoding.EncodeToString(privateKey)+"\n"), 0600); err != nil {
		return err
	}
	if err := os.Chmod(privatePath, 0600); err != nil {
		return err
	}
	if err := os.WriteFile(publicPath, []byte(base64.StdEncoding.EncodeToString(publicKey)+"\n"), 0644); err != nil {
		return err
	}
	fmt.Println("Private key:", privatePath)
	fmt.Println("Public key: ", publicPath)
	fmt.Println("Keep the private key secret. Add only the public key to DNS Filter.")
	return nil
}

func sign(args []string) error {
	flags := flag.NewFlagSet("sign", flag.ContinueOnError)
	keyID := flags.String("key-id", "", "trusted publisher key id")
	privatePath := flags.String("private", "", "base64 Ed25519 private key file")
	packID := flags.String("id", "", "security pack id")
	name := flags.String("name", "", "display name")
	category := flags.String("category", "threat", "domain category")
	domainPath := flags.String("domains", "", "newline-separated domain file")
	outPath := flags.String("out", "", "output JSON pack")
	version := flags.Int("version", 1, "positive pack version")
	validFor := flags.Duration("valid-for", 720*time.Hour, "pack lifetime")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !validID(*keyID) || !validID(*packID) || strings.TrimSpace(*name) == "" || *privatePath == "" || *domainPath == "" || *outPath == "" || *version < 1 || *validFor <= 0 {
		return fmt.Errorf("key-id, id, name, private, domains, out, positive version, and valid-for are required")
	}
	privateEncoded, err := os.ReadFile(*privatePath)
	if err != nil {
		return err
	}
	privateKey, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(privateEncoded)))
	if err != nil || len(privateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("private key file is not a valid Ed25519 private key")
	}
	domains, err := readDomains(*domainPath)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	manifest := securitypacks.Manifest{
		FormatVersion: securitypacks.FormatVersion, ID: *packID, Name: strings.TrimSpace(*name), Version: *version,
		Category: strings.TrimSpace(*category), KeyID: *keyID, CreatedAt: now, ExpiresAt: now.Add(*validFor), Domains: domains,
	}
	payload, _ := json.Marshal(manifest)
	pack := securitypacks.Pack{Manifest: manifest, Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(ed25519.PrivateKey(privateKey), payload))}
	encoded, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(*outPath, encoded, 0644); err != nil {
		return err
	}
	fmt.Printf("Signed %s with %d domains; expires %s\n", *outPath, len(domains), manifest.ExpiresAt.Format(time.RFC3339))
	return nil
}

func readDomains(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	seen := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		domain := strings.Trim(strings.ToLower(strings.TrimSpace(scanner.Text())), ".")
		if domain == "" || strings.HasPrefix(domain, "#") {
			continue
		}
		if _, valid := mdns.IsDomainName(domain); !valid {
			return nil, fmt.Errorf("invalid domain %q", domain)
		}
		seen[domain] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	domains := make([]string, 0, len(seen))
	for domain := range seen {
		domains = append(domains, domain)
	}
	sort.Strings(domains)
	if len(domains) == 0 {
		return nil, fmt.Errorf("domain file is empty")
	}
	return domains, nil
}

func validID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for i, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (i > 0 && (r == '.' || r == '_' || r == '-')) {
			continue
		}
		return false
	}
	return true
}
