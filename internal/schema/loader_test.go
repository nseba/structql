package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_SingleFile(t *testing.T) {
	schema, err := Load([]string{"../../testdata/schemas/starwars.graphql"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}

	// Verify key types exist
	if schema.Types["Human"] == nil {
		t.Error("expected Human type in schema")
	}
	if schema.Types["Episode"] == nil {
		t.Error("expected Episode enum in schema")
	}
	if schema.Types["ReviewInput"] == nil {
		t.Error("expected ReviewInput input type in schema")
	}
	if schema.Types["SearchResult"] == nil {
		t.Error("expected SearchResult union in schema")
	}
}

func TestLoad_GlobPattern(t *testing.T) {
	schema, err := Load([]string{"../../testdata/schemas/*.graphql"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}
}

func TestLoad_NoMatches(t *testing.T) {
	_, err := Load([]string{"/nonexistent/*.graphql"})
	if err == nil {
		t.Fatal("expected error for no matching files")
	}
}

func TestLoad_InvalidSchema(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.graphql")
	if err := os.WriteFile(f, []byte("this is not valid graphql {{{"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load([]string{f})
	if err == nil {
		t.Fatal("expected error for invalid schema")
	}
}

func TestLoad_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	schema1 := `type Foo { id: ID! }`
	schema2 := `type Bar { name: String! }`

	if err := os.WriteFile(filepath.Join(dir, "a.graphql"), []byte(schema1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.graphql"), []byte(schema2), 0644); err != nil {
		t.Fatal(err)
	}

	sch, err := Load([]string{filepath.Join(dir, "*.graphql")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sch.Types["Foo"] == nil {
		t.Error("expected Foo type")
	}
	if sch.Types["Bar"] == nil {
		t.Error("expected Bar type")
	}
}
