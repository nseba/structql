package query

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nseba/structql/internal/schema"
)

func TestLoad_ValidQueries(t *testing.T) {
	sch, err := schema.Load([]string{"../../testdata/schemas/starwars.graphql"})
	if err != nil {
		t.Fatalf("loading schema: %v", err)
	}

	ops, err := Load("../../testdata/queries", sch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ops) != 5 {
		t.Fatalf("expected 5 operations, got %d", len(ops))
	}

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}

	expected := []string{"HeroQuery", "GetHuman", "CreateReview", "SearchQuery", "ReviewAdded"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected operation %q not found", name)
		}
	}
}

func TestLoad_AnonymousOperation(t *testing.T) {
	sch, err := schema.Load([]string{"../../testdata/schemas/starwars.graphql"})
	if err != nil {
		t.Fatalf("loading schema: %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "anon.graphql"), []byte(`{ hero { name } }`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = Load(dir, sch)
	if err == nil {
		t.Fatal("expected error for anonymous operation")
	}
}

func TestLoad_DuplicateNames(t *testing.T) {
	sch, err := schema.Load([]string{"../../testdata/schemas/starwars.graphql"})
	if err != nil {
		t.Fatalf("loading schema: %v", err)
	}

	dir := t.TempDir()
	q := `query DupQuery($id: ID!) { human(id: $id) { name } }`
	if err := os.WriteFile(filepath.Join(dir, "a.graphql"), []byte(q), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.graphql"), []byte(q), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = Load(dir, sch)
	if err == nil {
		t.Fatal("expected error for duplicate operation names")
	}
}

func TestLoad_EmptyDir(t *testing.T) {
	sch, err := schema.Load([]string{"../../testdata/schemas/starwars.graphql"})
	if err != nil {
		t.Fatalf("loading schema: %v", err)
	}

	dir := t.TempDir()
	ops, err := Load(dir, sch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ops) != 0 {
		t.Errorf("expected 0 operations, got %d", len(ops))
	}
}

func TestLoad_InvalidQuery(t *testing.T) {
	sch, err := schema.Load([]string{"../../testdata/schemas/starwars.graphql"})
	if err != nil {
		t.Fatalf("loading schema: %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.graphql"), []byte(`query Bad { nonexistent { foo } }`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = Load(dir, sch)
	if err == nil {
		t.Fatal("expected error for invalid query")
	}
}

func TestLoad_RecursiveDiscovery(t *testing.T) {
	sch, err := schema.Load([]string{"../../testdata/schemas/starwars.graphql"})
	if err != nil {
		t.Fatalf("loading schema: %v", err)
	}

	// testdata/queries has nested/ subdir
	ops, err := Load("../../testdata/queries", sch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasSearch := false
	for _, op := range ops {
		if op.Name == "SearchQuery" {
			hasSearch = true
			break
		}
	}
	if !hasSearch {
		t.Error("expected SearchQuery from nested directory")
	}
}
