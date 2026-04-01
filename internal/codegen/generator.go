package codegen

import (
	"fmt"
	"sort"

	"github.com/nseba/structql/internal/config"
	"github.com/nseba/structql/internal/query"
	"github.com/vektah/gqlparser/v2/ast"
)

// GeneratedFile represents a file to be written.
type GeneratedFile struct {
	Name    string
	Content []byte
}

// Generator orchestrates code generation from schema and operations.
type Generator struct {
	cfg     *config.Config
	schema  *ast.Schema
	mapper  *TypeMapper
	builder *StructBuilder
}

// New creates a new Generator.
func New(cfg *config.Config, schema *ast.Schema) *Generator {
	mapper := NewTypeMapper(cfg)
	return &Generator{
		cfg:     cfg,
		schema:  schema,
		mapper:  mapper,
		builder: NewStructBuilder(schema, mapper),
	}
}

// Generate produces all output files from the given operations.
func (g *Generator) Generate(ops []*query.Operation) ([]*GeneratedFile, error) {
	var files []*GeneratedFile

	// Track referenced types for shared types file
	referencedEnums := map[string]bool{}
	referencedInputs := map[string]bool{}

	for _, op := range ops {
		// Build the operation AST from the query package operation
		astOp := &ast.OperationDefinition{
			Operation:           op.Type,
			Name:                op.Name,
			VariableDefinitions: op.Variables,
			SelectionSet:        op.Selection,
		}

		data, err := g.builder.BuildOperation(astOp, g.cfg.Package)
		if err != nil {
			return nil, fmt.Errorf("building %s: %w", op.Name, err)
		}

		src, err := RenderOperation(data)
		if err != nil {
			return nil, fmt.Errorf("rendering %s: %w", op.Name, err)
		}

		fileName := OperationFileName(op.Name, op.Type)
		files = append(files, &GeneratedFile{
			Name:    fileName,
			Content: src,
		})

		// Collect referenced enums and input types from variables and fields
		g.collectReferencedTypes(op.Variables, op.Selection, referencedEnums, referencedInputs)
	}

	// Generate shared types file
	typesFile, err := g.generateTypesFile(referencedEnums, referencedInputs)
	if err != nil {
		return nil, fmt.Errorf("generating types: %w", err)
	}
	if typesFile != nil {
		files = append(files, typesFile)
	}

	return files, nil
}

func (g *Generator) collectReferencedTypes(
	vars ast.VariableDefinitionList,
	selSet ast.SelectionSet,
	enums map[string]bool,
	inputs map[string]bool,
) {
	// Collect from variables
	for _, v := range vars {
		g.collectTypeRef(v.Type, enums, inputs)
	}

	// Collect from selection set
	g.collectFromSelectionSet(selSet, enums, inputs)
}

func (g *Generator) collectTypeRef(t *ast.Type, enums map[string]bool, inputs map[string]bool) {
	name := t.Name()
	if name == "" && t.Elem != nil {
		g.collectTypeRef(t.Elem, enums, inputs)
		return
	}

	typeDef := g.schema.Types[name]
	if typeDef == nil {
		return
	}

	switch typeDef.Kind {
	case ast.Enum:
		// Skip built-in enums
		if name != "__DirectiveLocation" && name != "__TypeKind" {
			enums[name] = true
		}
	case ast.InputObject:
		if !inputs[name] {
			inputs[name] = true
			// Recursively collect types referenced by input fields
			for _, f := range typeDef.Fields {
				g.collectTypeRef(f.Type, enums, inputs)
			}
		}
	}
}

func (g *Generator) collectFromSelectionSet(selSet ast.SelectionSet, enums map[string]bool, inputs map[string]bool) {
	for _, sel := range selSet {
		switch s := sel.(type) {
		case *ast.Field:
			if s.Definition != nil {
				g.collectTypeRef(s.Definition.Type, enums, inputs)
			}
			g.collectFromSelectionSet(s.SelectionSet, enums, inputs)
		case *ast.InlineFragment:
			g.collectFromSelectionSet(s.SelectionSet, enums, inputs)
		case *ast.FragmentSpread:
			if s.Definition != nil {
				g.collectFromSelectionSet(s.Definition.SelectionSet, enums, inputs)
			}
		}
	}
}

func (g *Generator) generateTypesFile(enums map[string]bool, inputs map[string]bool) (*GeneratedFile, error) {
	enumNames := sortedKeys(enums)
	inputNames := sortedKeys(inputs)

	if len(enumNames) == 0 && len(inputNames) == 0 {
		return nil, nil
	}

	enumData := g.builder.BuildEnums(enumNames)
	inputData := g.builder.BuildInputTypes(inputNames)

	// Check if any input fields need graphql import
	needsGraphQL := false
	for _, input := range inputData {
		for _, f := range input.Fields {
			if NeedsGraphQLImport(f.GoType) {
				needsGraphQL = true
				break
			}
		}
	}

	data := &TypesData{
		PackageName:  g.cfg.Package,
		Enums:        enumData,
		InputTypes:   inputData,
		NeedsGraphQL: needsGraphQL,
	}

	src, err := RenderTypes(data)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, nil
	}

	return &GeneratedFile{
		Name:    "types.go",
		Content: src,
	}, nil
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
