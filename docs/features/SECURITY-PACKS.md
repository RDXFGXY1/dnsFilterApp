# Signed Security Packs

A security pack is a JSON file containing a domain list, publisher metadata, and an **Ed25519** signature. The signature proves that the file was signed by the matching private key and that its content has not changed since signing.

A valid signature does not prove that the publisher is trustworthy. Only trust public keys obtained from a source you have independently verified.

## Generate a local signing key

```powershell
go run ./cmd/security-pack keygen -id local -out-dir ./keys
```

This creates a `.private` private key and a `.public` public key. Keep the private key outside the application directory and never share it.

## Sign a domain list

Place one domain on each line of a text file, then run:

```powershell
go run ./cmd/security-pack sign -id my-pack -name "My Pack" -category malware -domains ./domains.txt -private-key ./keys/local.private -out ./my-pack.json
```

Add the publisher's public key from the **Blocklists** view, verify the pack, and then install it.

DNS Filter rejects packs with an invalid signature or structure, expired metadata, excessive size, or an older version than the installed copy. Installed packs are verified again at startup, and expiration is checked while the application is running.
