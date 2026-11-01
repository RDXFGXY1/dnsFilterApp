package sideapps

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"errors"
	"golang.org/x/net/html"
	"mime"
	"path"
	"strings"
)

//go:embed sdk.js
var sdk string

// AppCSP applies even if someone opens a runner directly rather than in its frame.
const AppCSP = "sandbox allow-scripts; default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data:; font-src data:; connect-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'self'"

func attribute(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func removeAttribute(n *html.Node, key string) {
	attrs := n.Attr[:0]
	for _, a := range n.Attr {
		if a.Key != key {
			attrs = append(attrs, a)
		}
	}
	n.Attr = attrs
}
func setAttribute(n *html.Node, key, value string) {
	removeAttribute(n, key)
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: value})
}
func (p *Package) resource(ref string) ([]byte, error) {
	if strings.ContainsAny(ref, "?#:\\") || strings.HasPrefix(ref, "/") {
		return nil, errors.New("app assets must use relative package paths")
	}
	name := path.Join(path.Dir(p.Manifest.Entry), ref)
	if !safePath(name) {
		return nil, errors.New("unsafe asset path")
	}
	b, ok := p.Files[name]
	if !ok {
		return nil, errors.New("referenced app asset is missing: " + name)
	}
	return b, nil
}
func (p *Package) Render() ([]byte, error) {
	doc, e := html.Parse(bytes.NewReader(p.Files[p.Manifest.Entry]))
	if e != nil {
		return nil, e
	}
	var visit func(*html.Node) error
	visit = func(n *html.Node) error {
		if n.Type == html.ElementNode {
			if n.Data == "head" {
				bridge := &html.Node{Type: html.ElementNode, Data: "script"}
				bridge.AppendChild(&html.Node{Type: html.TextNode, Data: sdk})
				n.InsertBefore(bridge, n.FirstChild)
			}
			if n.Data == "script" && attribute(n, "src") != "" {
				ref := attribute(n, "src")
				if path.Ext(ref) != ".js" {
					return errors.New("scripts must reference packaged .js files")
				}
				b, e := p.resource(ref)
				if e != nil {
					return e
				}
				removeAttribute(n, "src")
				removeAttribute(n, "integrity")
				for n.FirstChild != nil {
					n.RemoveChild(n.FirstChild)
				}
				n.AppendChild(&html.Node{Type: html.TextNode, Data: string(b)})
			}
			if n.Data == "link" && strings.EqualFold(attribute(n, "rel"), "stylesheet") {
				ref := attribute(n, "href")
				if path.Ext(ref) != ".css" {
					return errors.New("stylesheets must reference packaged .css files")
				}
				b, e := p.resource(ref)
				if e != nil {
					return e
				}
				n.Data = "style"
				n.Attr = nil
				n.AppendChild(&html.Node{Type: html.TextNode, Data: string(b)})
			}
			if n.Data == "img" && attribute(n, "src") != "" && !strings.HasPrefix(attribute(n, "src"), "data:") {
				ref := attribute(n, "src")
				b, e := p.resource(ref)
				if e != nil {
					return e
				}
				t := mime.TypeByExtension(path.Ext(ref))
				if !strings.HasPrefix(t, "image/") {
					return errors.New("invalid image asset")
				}
				setAttribute(n, "src", "data:"+t+";base64,"+base64.StdEncoding.EncodeToString(b))
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if e := visit(c); e != nil {
				return e
			}
		}
		return nil
	}
	if e = visit(doc); e != nil {
		return nil, e
	}
	var out bytes.Buffer
	if e = html.Render(&out, doc); e != nil {
		return nil, e
	}
	return out.Bytes(), nil
}
