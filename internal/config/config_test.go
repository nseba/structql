package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Valid(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "structql.yaml")
	content := `
schema:
  - "schema.graphql"
queries: "queries/"
output: "generated/"
package: "generated"
scalars:
  DateTime:
    type: "time.Time"
    import: "time"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Schema) != 1 {
		t.Fatalf("expected 1 schema entry, got %d", len(cfg.Schema))
	}
	if cfg.Package != "generated" {
		t.Errorf("expected package 'generated', got %q", cfg.Package)
	}
	if cfg.Scalars["DateTime"].Type != "time.Time" {
		t.Errorf("expected DateTime scalar type 'time.Time', got %q", cfg.Scalars["DateTime"].Type)
	}
	if cfg.Scalars["DateTime"].Import != "time" {
		t.Errorf("expected DateTime import 'time', got %q", cfg.Scalars["DateTime"].Import)
	}

	// Paths should be resolved relative to config file
	expected := filepath.Join(dir, "schema.graphql")
	if cfg.Schema[0] != expected {
		t.Errorf("expected schema path %q, got %q", expected, cfg.Schema[0])
	}
	expected = filepath.Join(dir, "queries")
	if cfg.Queries != expected {
		t.Errorf("expected queries path %q, got %q", expected, cfg.Queries)
	}
}

func TestLoad_MissingSchema(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "structql.yaml")
	content := `
queries: "queries/"
output: "generated/"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for missing schema")
	}
}

func TestLoad_InvalidPackageName(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "structql.yaml")
	content := `
schema:
  - "schema.graphql"
queries: "queries/"
output: "generated/"
package: "123-invalid"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid package name")
	}
}

func TestLoad_MissingScalarType(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "structql.yaml")
	content := `
schema:
  - "schema.graphql"
queries: "queries/"
output: "generated/"
scalars:
  DateTime:
    import: "time"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for missing scalar type")
	}
}

func TestLoad_DefaultPackage(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "structql.yaml")
	content := `
schema:
  - "schema.graphql"
queries: "queries/"
output: "mypackage/"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Package != "mypackage" {
		t.Errorf("expected default package 'mypackage', got %q", cfg.Package)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/structql.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Package != "generated" {
		t.Errorf("expected default package 'generated', got %q", cfg.Package)
	}
	if len(cfg.Schema) != 1 {
		t.Errorf("expected 1 default schema entry, got %d", len(cfg.Schema))
	}
}

func TestMarshal(t *testing.T) {
	cfg := DefaultConfig()
	data, err := cfg.Marshal()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty YAML output")
	}
}
