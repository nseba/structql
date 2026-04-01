// Package query handles loading, parsing, and validating GraphQL query files.
package query

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

// Operation represents a parsed and validated GraphQL operation.
type Operation struct {
	Name      string
	Type      ast.Operation // Query, Mutation, Subscription
	Variables ast.VariableDefinitionList
	Selection ast.SelectionSet
	Source    string // source file path
}

// Load discovers .graphql files recursively under dir, parses and validates them against the schema.
func Load(dir string, schema *ast.Schema) ([]*Operation, error) {
	files, err := discoverFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("discovering query files: %w", err)
	}

	if len(files) == 0 {
		return nil, nil
	}

	var ops []*Operation
	for _, f := range files {
		fileOps, err := parseFile(f, schema)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", f, err)
		}
		ops = append(ops, fileOps...)
	}

	// Check for duplicate operation names
	seen := make(map[string]string)
	for _, op := range ops {
		if prev, ok := seen[op.Name]; ok {
			return nil, fmt.Errorf("duplicate operation name %q in %s and %s", op.Name, prev, op.Source)
		}
		seen[op.Name] = op.Source
	}

	return ops, nil
}

func discoverFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".graphql") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func parseFile(path string, schema *ast.Schema) ([]*Operation, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	source := &ast.Source{
		Name:  path,
		Input: string(content),
	}

	doc, gqlErr := gqlparser.LoadQueryWithRules(schema, source.Input, nil)
	if gqlErr != nil {
		return nil, fmt.Errorf("parsing/validation: %s", gqlErr.Error())
	}

	var ops []*Operation
	for _, op := range doc.Operations {
		if op.Name == "" {
			return nil, fmt.Errorf("anonymous operations are not supported; all operations must be named")
		}
		ops = append(ops, &Operation{
			Name:      op.Name,
			Type:      op.Operation,
			Variables: op.VariableDefinitions,
			Selection: op.SelectionSet,
			Source:    path,
		})
	}

	return ops, nil
}
