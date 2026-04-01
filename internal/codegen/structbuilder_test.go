package codegen

import (
	"testing"

	"github.com/nseba/structql/internal/config"
	"github.com/nseba/structql/internal/schema"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func loadTestSchemaAndQuery(t *testing.T, queryStr string) (*ast.Schema, *ast.OperationDefinition) {
	t.Helper()
	sch, err := schema.Load([]string{"../../testdata/schemas/starwars.graphql"})
	if err != nil {
		t.Fatalf("loading schema: %v", err)
	}

	doc, gqlErr := gqlparser.LoadQueryWithRules(sch, queryStr, nil)
	if gqlErr != nil {
		t.Fatalf("parsing query: %v", gqlErr)
	}
	if len(doc.Operations) == 0 {
		t.Fatal("no operations found")
	}

	return sch, doc.Operations[0]
}

func TestBuildOperation_SimpleQuery(t *testing.T) {
	sch, op := loadTestSchemaAndQuery(t, `
		query GetHuman($id: ID!) {
			human(id: $id) {
				id
				name
				height
			}
		}
	`)

	mapper := NewTypeMapper(&config.Config{Scalars: map[string]config.ScalarConfig{}})
	builder := NewStructBuilder(sch, mapper)

	data, err := builder.BuildOperation(op, "testpkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.OperationName != "GetHuman" {
		t.Errorf("expected operation name GetHuman, got %q", data.OperationName)
	}
	if data.OperationType != "query" {
		t.Errorf("expected operation type query, got %q", data.OperationType)
	}
	if len(data.Fields) != 1 {
		t.Fatalf("expected 1 top-level field, got %d", len(data.Fields))
	}

	humanField := data.Fields[0]
	if humanField.Name != "Human" {
		t.Errorf("expected field name Human, got %q", humanField.Name)
	}
	if len(humanField.Children) != 3 {
		t.Errorf("expected 3 children, got %d", len(humanField.Children))
	}

	// Check variables
	if len(data.Variables) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(data.Variables))
	}
	if data.Variables[0].GraphQLName != "id" {
		t.Errorf("expected variable name 'id', got %q", data.Variables[0].GraphQLName)
	}
}

func TestBuildOperation_InlineFragments(t *testing.T) {
	sch, op := loadTestSchemaAndQuery(t, `
		query HeroQuery($episode: Episode!) {
			hero(episode: $episode) {
				name
				... on Droid {
					primaryFunction
				}
				... on Human {
					homePlanet
				}
			}
		}
	`)

	mapper := NewTypeMapper(&config.Config{Scalars: map[string]config.ScalarConfig{}})
	builder := NewStructBuilder(sch, mapper)

	data, err := builder.BuildOperation(op, "testpkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	heroField := data.Fields[0]
	// Should have: name, Droid fragment, Human fragment
	if len(heroField.Children) != 3 {
		t.Fatalf("expected 3 children (name + 2 fragments), got %d", len(heroField.Children))
	}

	// Check inline fragment tags
	droidFound := false
	humanFound := false
	for _, child := range heroField.Children {
		if child.Tag == "... on Droid" {
			droidFound = true
			if len(child.Children) != 1 {
				t.Errorf("expected 1 Droid field, got %d", len(child.Children))
			}
		}
		if child.Tag == "... on Human" {
			humanFound = true
		}
	}
	if !droidFound {
		t.Error("missing Droid inline fragment")
	}
	if !humanFound {
		t.Error("missing Human inline fragment")
	}
}

func TestBuildOperation_Mutation(t *testing.T) {
	sch, op := loadTestSchemaAndQuery(t, `
		mutation CreateReview($episode: Episode!, $review: ReviewInput!) {
			createReview(episode: $episode, review: $review) {
				stars
				commentary
			}
		}
	`)

	mapper := NewTypeMapper(&config.Config{Scalars: map[string]config.ScalarConfig{}})
	builder := NewStructBuilder(sch, mapper)

	data, err := builder.BuildOperation(op, "testpkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.OperationType != "mutation" {
		t.Errorf("expected mutation, got %q", data.OperationType)
	}
	if len(data.Variables) != 2 {
		t.Errorf("expected 2 variables, got %d", len(data.Variables))
	}
}

func TestBuildEnums(t *testing.T) {
	sch, err := schema.Load([]string{"../../testdata/schemas/starwars.graphql"})
	if err != nil {
		t.Fatalf("loading schema: %v", err)
	}

	mapper := NewTypeMapper(&config.Config{Scalars: map[string]config.ScalarConfig{}})
	builder := NewStructBuilder(sch, mapper)

	enums := builder.BuildEnums([]string{"Episode"})
	if len(enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(enums))
	}
	if enums[0].Name != "Episode" {
		t.Errorf("expected enum name Episode, got %q", enums[0].Name)
	}
	if len(enums[0].Values) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(enums[0].Values))
	}
}

func TestBuildInputTypes(t *testing.T) {
	sch, err := schema.Load([]string{"../../testdata/schemas/starwars.graphql"})
	if err != nil {
		t.Fatalf("loading schema: %v", err)
	}

	mapper := NewTypeMapper(&config.Config{Scalars: map[string]config.ScalarConfig{}})
	builder := NewStructBuilder(sch, mapper)

	inputs := builder.BuildInputTypes([]string{"ReviewInput"})
	if len(inputs) != 1 {
		t.Fatalf("expected 1 input type, got %d", len(inputs))
	}
	if inputs[0].Name != "ReviewInput" {
		t.Errorf("expected input name ReviewInput, got %q", inputs[0].Name)
	}
	if len(inputs[0].Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(inputs[0].Fields))
	}
}
