// Package codegen generates Go source code for hasura/go-graphql-client from GraphQL operations.
package codegen

import (
	"fmt"
	"strings"

	"github.com/nseba/structql/internal/config"
	"github.com/vektah/gqlparser/v2/ast"
)

const graphqlClientPkg = "github.com/hasura/go-graphql-client"

// TypeMapper converts GraphQL types to Go types.
type TypeMapper struct {
	scalars      map[string]config.ScalarConfig
	typeMappings map[string]string
	prefix       string
}

// NewTypeMapper creates a TypeMapper from config.
func NewTypeMapper(cfg *config.Config) *TypeMapper {
	return &TypeMapper{
		scalars:      cfg.Scalars,
		typeMappings: cfg.TypeMappings,
		prefix:       cfg.Prefix,
	}
}

// GoFieldType returns the Go type for a GraphQL type as used in struct fields.
// Handles nullability (pointer types) and lists (slice types).
func (m *TypeMapper) GoFieldType(gqlType *ast.Type) string {
	return m.goType(gqlType, false)
}

// GoVariableType returns the Go type for a GraphQL variable type.
// Variables use wrapper types from the graphql client library.
func (m *TypeMapper) GoVariableType(gqlType *ast.Type) string {
	return m.goType(gqlType, true)
}

func (m *TypeMapper) goType(gqlType *ast.Type, forVariable bool) string {
	if gqlType.Elem != nil {
		// List type
		inner := m.goType(gqlType.Elem, forVariable)
		result := "[]" + inner
		if !gqlType.NonNull {
			result = "*" + result
		}
		return result
	}

	baseType := m.baseGoType(gqlType.NamedType, forVariable)
	if !gqlType.NonNull {
		return "*" + baseType
	}
	return baseType
}

func (m *TypeMapper) baseGoType(name string, forVariable bool) string {
	// Check custom scalar mappings first
	if sc, ok := m.scalars[name]; ok {
		return sc.Type
	}

	// Check type_mappings overrides
	if override, ok := m.typeMappings[name]; ok {
		return override
	}

	// Built-in GraphQL scalars
	if forVariable {
		return m.builtinVariableType(name)
	}
	return m.builtinFieldType(name)
}

func (m *TypeMapper) builtinFieldType(name string) string {
	switch name {
	case "String":
		return "string"
	case "Int":
		return "int"
	case "Float":
		return "float64"
	case "Boolean":
		return "bool"
	case "ID":
		return "graphql.ID"
	default:
		// Enum or input type - use the Go type name
		return m.prefix + name
	}
}

func (m *TypeMapper) builtinVariableType(name string) string {
	switch name {
	case "String":
		return "graphql.String"
	case "Int":
		return "graphql.Int"
	case "Float":
		return "graphql.Float"
	case "Boolean":
		return "graphql.Boolean"
	case "ID":
		return "graphql.ID"
	default:
		// Enum or input type - use the Go type name directly
		return m.prefix + name
	}
}

// Imports returns the set of import paths needed for the generated code, given the types used.
func (m *TypeMapper) Imports(usedTypes []string) []string {
	imports := map[string]bool{}

	for _, t := range usedTypes {
		// Check if it references graphql package
		if strings.HasPrefix(t, "graphql.") || strings.Contains(t, "graphql.") {
			imports[graphqlClientPkg] = true
		}
		// Check custom scalar imports
		for _, sc := range m.scalars {
			if sc.Import != "" && strings.Contains(t, sc.Type) {
				imports[sc.Import] = true
			}
		}
	}

	var result []string
	for imp := range imports {
		result = append(result, imp)
	}
	return result
}

// NeedsGraphQLImport returns true if the given type string references the graphql package.
func NeedsGraphQLImport(goType string) bool {
	return strings.Contains(goType, "graphql.")
}

// GoName converts a GraphQL field name to an exported Go identifier (PascalCase).
func GoName(name string) string {
	if name == "" {
		return ""
	}
	// Handle common acronyms
	upper := strings.ToUpper(name)
	if isCommonAcronym(upper) {
		return upper
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func isCommonAcronym(s string) bool {
	acronyms := map[string]bool{
		"ID": true, "URL": true, "URI": true, "API": true,
		"HTTP": true, "HTTPS": true, "JSON": true, "XML": true,
		"SQL": true, "SSH": true, "TCP": true, "UDP": true,
	}
	return acronyms[s]
}

// OperationFileName returns the output file name for a given operation.
func OperationFileName(name string, opType ast.Operation) string {
	snake := toSnakeCase(name)
	switch opType {
	case ast.Mutation:
		return snake + "_mutation.go"
	case ast.Subscription:
		return snake + "_subscription.go"
	default:
		return snake + "_query.go"
	}
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Don't add underscore if previous char was also uppercase (e.g., "ID")
			if i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z' {
				result.WriteByte('_')
			} else if s[i-1] >= 'a' && s[i-1] <= 'z' {
				result.WriteByte('_')
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// FormatTag builds a graphql struct tag value.
func FormatTag(fieldName string, args []*ast.Argument) string {
	if len(args) == 0 {
		return fieldName
	}
	var argParts []string
	for _, arg := range args {
		argParts = append(argParts, fmt.Sprintf("%s: %s", arg.Name, formatArgValue(arg.Value)))
	}
	return fmt.Sprintf("%s(%s)", fieldName, strings.Join(argParts, ", "))
}

func formatArgValue(val *ast.Value) string {
	if val == nil {
		return "null"
	}
	switch val.Kind {
	case ast.Variable:
		return "$" + val.Raw
	case ast.StringValue:
		return fmt.Sprintf(`\"%s\"`, val.Raw)
	case ast.EnumValue, ast.IntValue, ast.FloatValue, ast.BooleanValue:
		return val.Raw
	default:
		return val.Raw
	}
}
