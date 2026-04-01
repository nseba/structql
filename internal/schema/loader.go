// Package schema handles loading and parsing GraphQL schema files.
package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

// Load parses GraphQL schema files from the given glob patterns and returns a merged schema.
func Load(patterns []string) (*ast.Schema, error) {
	var files []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		files = append(files, matches...)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no schema files found matching patterns: %v", patterns)
	}

	// Deduplicate and sort for deterministic behavior
	files = dedupSort(files)

	var sources []*ast.Source
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading schema file %q: %w", f, err)
		}
		sources = append(sources, &ast.Source{
			Name:  f,
			Input: string(content),
		})
	}

	schema, gqlErr := gqlparser.LoadSchema(sources...)
	if gqlErr != nil {
		return nil, fmt.Errorf("parsing schema: %s", gqlErr.Error())
	}

	return schema, nil
}

func dedupSort(ss []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range ss {
		abs, _ := filepath.Abs(s)
		if !seen[abs] {
			seen[abs] = true
			result = append(result, s)
		}
	}
	sort.Strings(result)
	return result
}
