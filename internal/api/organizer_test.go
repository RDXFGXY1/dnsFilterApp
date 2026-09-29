package api

import "testing"

func TestDecodeTodoYAML(t *testing.T) {
	doc, err := decodeTodoYAML(`
version: 1
list: Launch My Website
todos:
  - title: Plan the website
    done: false
    priority: high
    subtodos:
      - title: Define the purpose
        done: true
  - title: Build and test
    priority: medium
`)
	if err != nil {
		t.Fatalf("decodeTodoYAML returned error: %v", err)
	}
	if doc.List != "Launch My Website" || len(doc.Todos) != 2 || len(doc.Todos[0].Subtodos) != 1 {
		t.Fatalf("unexpected document: %#v", doc)
	}
	if doc.Todos[0].Priority != "high" || doc.Todos[0].Subtodos[0].Priority != "medium" {
		t.Fatalf("priorities were not normalized: %#v", doc.Todos[0])
	}
}

func TestDecodeTodoYAMLHelpfulValidation(t *testing.T) {
	_, err := decodeTodoYAML("version: 1\nlist: Test\ntodos:\n  - done: false\n")
	if err == nil || err.Error() != "todo 1 is missing a title" {
		t.Fatalf("expected missing-title error, got %v", err)
	}

	_, err = decodeTodoYAML("version: 1\nlist: Test\ntodos:\n  - title: Bad priority\n    priority: extreme\n")
	if err == nil {
		t.Fatal("expected invalid-priority error")
	}
}
