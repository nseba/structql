package codegen

import (
	"testing"

	"github.com/nseba/structql/internal/config"
	"github.com/vektah/gqlparser/v2/ast"
)

func newTestMapper() *TypeMapper {
	return NewTypeMapper(&config.Config{
		Scalars: map[string]config.ScalarConfig{
			"DateTime": {Type: "time.Time", Import: "time"},
			"JSON":     {Type: "json.RawMessage", Import: "encoding/json"},
		},
		TypeMappings: map[string]string{},
	})
}

func TestGoFieldType_Scalars(t *testing.T) {
	m := newTestMapper()
	tests := []struct {
		gqlType *ast.Type
		want    string
	}{
		{ast.NonNullNamedType("String", nil), "string"},
		{ast.NamedType("String", nil), "*string"},
		{ast.NonNullNamedType("Int", nil), "int"},
		{ast.NamedType("Int", nil), "*int"},
		{ast.NonNullNamedType("Float", nil), "float64"},
		{ast.NamedType("Float", nil), "*float64"},
		{ast.NonNullNamedType("Boolean", nil), "bool"},
		{ast.NamedType("Boolean", nil), "*bool"},
		{ast.NonNullNamedType("ID", nil), "graphql.ID"},
		{ast.NamedType("ID", nil), "*graphql.ID"},
	}

	for _, tt := range tests {
		got := m.GoFieldType(tt.gqlType)
		if got != tt.want {
			t.Errorf("GoFieldType(%s) = %q, want %q", tt.gqlType.String(), got, tt.want)
		}
	}
}

func TestGoFieldType_Lists(t *testing.T) {
	m := newTestMapper()
	tests := []struct {
		name    string
		gqlType *ast.Type
		want    string
	}{
		{
			"[String!]!",
			ast.NonNullListType(ast.NonNullNamedType("String", nil), nil),
			"[]string",
		},
		{
			"[String]!",
			ast.NonNullListType(ast.NamedType("String", nil), nil),
			"[]*string",
		},
		{
			"[String!]",
			ast.ListType(ast.NonNullNamedType("String", nil), nil),
			"*[]string",
		},
		{
			"[String]",
			ast.ListType(ast.NamedType("String", nil), nil),
			"*[]*string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.GoFieldType(tt.gqlType)
			if got != tt.want {
				t.Errorf("GoFieldType(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestGoFieldType_CustomScalar(t *testing.T) {
	m := newTestMapper()

	got := m.GoFieldType(ast.NonNullNamedType("DateTime", nil))
	if got != "time.Time" {
		t.Errorf("expected time.Time, got %q", got)
	}

	got = m.GoFieldType(ast.NamedType("DateTime", nil))
	if got != "*time.Time" {
		t.Errorf("expected *time.Time, got %q", got)
	}
}

func TestGoFieldType_Enum(t *testing.T) {
	m := newTestMapper()
	got := m.GoFieldType(ast.NonNullNamedType("Episode", nil))
	if got != "Episode" {
		t.Errorf("expected Episode, got %q", got)
	}
}

func TestGoVariableType_Scalars(t *testing.T) {
	m := newTestMapper()
	tests := []struct {
		gqlType *ast.Type
		want    string
	}{
		{ast.NonNullNamedType("String", nil), "graphql.String"},
		{ast.NonNullNamedType("Int", nil), "graphql.Int"},
		{ast.NonNullNamedType("Float", nil), "graphql.Float"},
		{ast.NonNullNamedType("Boolean", nil), "graphql.Boolean"},
		{ast.NonNullNamedType("ID", nil), "graphql.ID"},
	}

	for _, tt := range tests {
		got := m.GoVariableType(tt.gqlType)
		if got != tt.want {
			t.Errorf("GoVariableType(%s) = %q, want %q", tt.gqlType.String(), got, tt.want)
		}
	}
}

func TestGoName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"name", "Name"},
		{"firstName", "FirstName"},
		{"id", "ID"},
		{"url", "URL"},
		{"primaryFunction", "PrimaryFunction"},
	}

	for _, tt := range tests {
		got := GoName(tt.input)
		if got != tt.want {
			t.Errorf("GoName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestOperationFileName(t *testing.T) {
	tests := []struct {
		name   string
		opType ast.Operation
		want   string
	}{
		{"HeroQuery", ast.Query, "hero_query_query.go"},
		{"CreateReview", ast.Mutation, "create_review_mutation.go"},
		{"ReviewAdded", ast.Subscription, "review_added_subscription.go"},
	}

	for _, tt := range tests {
		got := OperationFileName(tt.name, tt.opType)
		if got != tt.want {
			t.Errorf("OperationFileName(%q, %v) = %q, want %q", tt.name, tt.opType, got, tt.want)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"HeroQuery", "hero_query"},
		{"CreateReview", "create_review"},
		{"GetHTTPResponse", "get_http_response"},
		{"simpleTest", "simple_test"},
		{"ID", "id"},
	}

	for _, tt := range tests {
		got := toSnakeCase(tt.input)
		if got != tt.want {
			t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTypeMapper_WithPrefix(t *testing.T) {
	m := NewTypeMapper(&config.Config{
		Prefix:  "GQL",
		Scalars: map[string]config.ScalarConfig{},
	})

	got := m.GoFieldType(ast.NonNullNamedType("Episode", nil))
	if got != "GQLEpisode" {
		t.Errorf("expected GQLEpisode, got %q", got)
	}
}

func TestTypeMapper_TypeMappingOverride(t *testing.T) {
	m := NewTypeMapper(&config.Config{
		Scalars:      map[string]config.ScalarConfig{},
		TypeMappings: map[string]string{"ID": "string"},
	})

	got := m.GoFieldType(ast.NonNullNamedType("ID", nil))
	if got != "string" {
		t.Errorf("expected string, got %q", got)
	}
}
