package codegen

import (
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// StructField represents a Go struct field for code generation.
type StructField struct {
	Name     string
	GoType   string
	Tag      string // graphql struct tag value
	Comment  string
	Children []*StructField // for nested structs (anonymous or inline fragment)
	IsScalar bool           // if true, render as scalar tag
	IsList   bool           // if true, this nested struct is a slice
	IsPtr    bool           // if true, this nested struct is a pointer (nullable)
}

// OperationData holds all data needed to render an operation file.
type OperationData struct {
	PackageName   string
	OperationName string
	OperationType string // "query", "mutation", "subscription"
	Fields        []*StructField
	Variables     []*VariableField
	Imports       []string
	NeedsGraphQL  bool
}

// VariableField represents a variable in the operation.
type VariableField struct {
	GraphQLName string
	GoName      string
	GoType      string
	NonNull     bool
}

// TypesData holds all data needed to render the shared types file.
type TypesData struct {
	PackageName  string
	Enums        []*EnumData
	InputTypes   []*InputTypeData
	Imports      []string
	NeedsGraphQL bool
}

// EnumData represents a GraphQL enum type.
type EnumData struct {
	Name    string
	Comment string
	Values  []*EnumValue
}

// EnumValue represents a single enum value.
type EnumValue struct {
	GoName       string
	Value        string
	EnumTypeName string // The Go type name of the parent enum
}

// InputTypeData represents a GraphQL input type.
type InputTypeData struct {
	Name    string
	Comment string
	Fields  []*InputField
}

// InputField represents a field in an input type.
type InputField struct {
	GoName  string
	GoType  string
	JSONTag string
}

// StructBuilder builds Go struct representations from GraphQL selection sets.
type StructBuilder struct {
	schema *ast.Schema
	mapper *TypeMapper
}

// NewStructBuilder creates a StructBuilder.
func NewStructBuilder(schema *ast.Schema, mapper *TypeMapper) *StructBuilder {
	return &StructBuilder{
		schema: schema,
		mapper: mapper,
	}
}

// BuildOperation builds the OperationData for a given operation.
func (b *StructBuilder) BuildOperation(op *ast.OperationDefinition, pkgName string) (*OperationData, error) {
	fields, err := b.buildSelectionSet(op.SelectionSet, nil)
	if err != nil {
		return nil, fmt.Errorf("building selection set for %s: %w", op.Name, err)
	}

	vars := b.buildVariables(op.VariableDefinitions)

	// Collect all types to determine imports
	allTypes := b.collectTypes(fields, vars)
	needsGraphQL := false
	for _, t := range allTypes {
		if NeedsGraphQLImport(t) {
			needsGraphQL = true
			break
		}
	}

	imports := b.mapper.Imports(allTypes)

	opType := "query"
	switch op.Operation {
	case ast.Mutation:
		opType = "mutation"
	case ast.Subscription:
		opType = "subscription"
	}

	return &OperationData{
		PackageName:   pkgName,
		OperationName: op.Name,
		OperationType: opType,
		Fields:        fields,
		Variables:     vars,
		Imports:       imports,
		NeedsGraphQL:  needsGraphQL,
	}, nil
}

func (b *StructBuilder) buildSelectionSet(selSet ast.SelectionSet, parentDef *ast.Definition) ([]*StructField, error) {
	var fields []*StructField

	for _, sel := range selSet {
		switch s := sel.(type) {
		case *ast.Field:
			field, err := b.buildField(s)
			if err != nil {
				return nil, err
			}
			fields = append(fields, field)

		case *ast.InlineFragment:
			field, err := b.buildInlineFragment(s)
			if err != nil {
				return nil, err
			}
			fields = append(fields, field)

		case *ast.FragmentSpread:
			// Inline the fragment's selections
			frag := s.Definition
			if frag == nil {
				return nil, fmt.Errorf("undefined fragment: %s", s.Name)
			}
			fragFields, err := b.buildSelectionSet(frag.SelectionSet, frag.Definition)
			if err != nil {
				return nil, err
			}
			fields = append(fields, fragFields...)
		}
	}

	return fields, nil
}

func (b *StructBuilder) buildField(f *ast.Field) (*StructField, error) {
	fieldDef := f.Definition
	if fieldDef == nil {
		return nil, fmt.Errorf("no definition for field %q", f.Name)
	}

	goName := GoName(f.Alias)
	if f.Alias == f.Name {
		goName = GoName(f.Name)
	}

	// Build the graphql tag
	tag := b.buildFieldTag(f)

	typeDef := b.schema.Types[fieldDef.Type.Name()]
	isLeaf := typeDef == nil || typeDef.Kind == ast.Scalar || typeDef.Kind == ast.Enum

	if isLeaf || len(f.SelectionSet) == 0 {
		goType := b.mapper.GoFieldType(fieldDef.Type)

		// Check if this is a custom scalar that needs the scalar tag
		isCustomScalar := false
		if typeDef != nil && typeDef.Kind == ast.Scalar {
			_, isCustom := b.mapper.scalars[typeDef.Name]
			// Also check builtins
			switch typeDef.Name {
			case "String", "Int", "Float", "Boolean", "ID":
			default:
				if !isCustom {
					isCustomScalar = true
				}
			}
		}

		return &StructField{
			Name:     goName,
			GoType:   goType,
			Tag:      tag,
			IsScalar: isCustomScalar,
		}, nil
	}

	// Object type - recurse into children
	children, err := b.buildSelectionSet(f.SelectionSet, typeDef)
	if err != nil {
		return nil, err
	}

	// Determine if it's a list or nullable
	isList, isPtr := b.objectTypeFlags(fieldDef.Type)

	return &StructField{
		Name:     goName,
		Tag:      tag,
		Children: children,
		IsList:   isList,
		IsPtr:    isPtr,
	}, nil
}

func (b *StructBuilder) buildFieldTag(f *ast.Field) string {
	// Determine the field name for the tag
	tagName := f.Name

	// If field has an alias that differs, use alias:name format
	if f.Alias != "" && f.Alias != f.Name {
		tagName = f.Alias + ":" + f.Name
	}

	// Build arguments
	if len(f.Arguments) > 0 {
		return FormatTag(tagName, f.Arguments)
	}

	// Only add tag if Go name differs from what the library would generate
	goName := GoName(f.Name)
	if f.Alias != "" && f.Alias != f.Name {
		return tagName
	}
	if strings.ToLower(goName[:1])+goName[1:] == f.Name {
		return "" // library auto-converts, no tag needed
	}
	return tagName
}

func (b *StructBuilder) buildInlineFragment(f *ast.InlineFragment) (*StructField, error) {
	typeName := f.TypeCondition
	children, err := b.buildSelectionSet(f.SelectionSet, b.schema.Types[typeName])
	if err != nil {
		return nil, err
	}

	return &StructField{
		Name:     typeName,
		Tag:      fmt.Sprintf("... on %s", typeName),
		Children: children,
	}, nil
}

func (b *StructBuilder) buildVariables(vars ast.VariableDefinitionList) []*VariableField {
	var result []*VariableField
	for _, v := range vars {
		goType := b.mapper.GoVariableType(v.Type)
		result = append(result, &VariableField{
			GraphQLName: v.Variable,
			GoName:      GoName(v.Variable),
			GoType:      goType,
			NonNull:     v.Type.NonNull,
		})
	}
	return result
}

// objectTypeFlags returns whether an object type is a list and/or nullable.
func (b *StructBuilder) objectTypeFlags(gqlType *ast.Type) (isList bool, isPtr bool) {
	if gqlType.Elem != nil {
		return true, !gqlType.NonNull
	}
	return false, !gqlType.NonNull
}

func (b *StructBuilder) collectTypes(fields []*StructField, vars []*VariableField) []string {
	var types []string
	for _, f := range fields {
		if f.GoType != "" && f.GoType != "list" && f.GoType != "pointer" && f.GoType != "struct" {
			types = append(types, f.GoType)
		}
		if len(f.Children) > 0 {
			types = append(types, b.collectTypes(f.Children, nil)...)
		}
	}
	for _, v := range vars {
		types = append(types, v.GoType)
	}
	return types
}

// BuildEnums builds enum data for all enums referenced by the operations.
func (b *StructBuilder) BuildEnums(names []string) []*EnumData {
	var enums []*EnumData
	for _, name := range names {
		typeDef := b.schema.Types[name]
		if typeDef == nil || typeDef.Kind != ast.Enum {
			continue
		}

		enum := &EnumData{
			Name:    b.mapper.prefix + name,
			Comment: fmt.Sprintf("%s represents the GraphQL enum %s.", b.mapper.prefix+name, name),
		}
		for _, v := range typeDef.EnumValues {
			enum.Values = append(enum.Values, &EnumValue{
				GoName:       b.mapper.prefix + name + GoName(strings.ToLower(v.Name)),
				Value:        v.Name,
				EnumTypeName: b.mapper.prefix + name,
			})
		}
		enums = append(enums, enum)
	}
	return enums
}

// BuildInputTypes builds input type data for all input types referenced by the operations.
func (b *StructBuilder) BuildInputTypes(names []string) []*InputTypeData {
	var inputs []*InputTypeData
	for _, name := range names {
		typeDef := b.schema.Types[name]
		if typeDef == nil || typeDef.Kind != ast.InputObject {
			continue
		}

		input := &InputTypeData{
			Name:    b.mapper.prefix + name,
			Comment: fmt.Sprintf("%s represents the GraphQL input type %s.", b.mapper.prefix+name, name),
		}
		for _, f := range typeDef.Fields {
			goType := b.mapper.GoFieldType(f.Type)
			jsonTag := f.Name
			if !f.Type.NonNull {
				jsonTag += ",omitempty"
			}
			input.Fields = append(input.Fields, &InputField{
				GoName:  GoName(f.Name),
				GoType:  goType,
				JSONTag: jsonTag,
			})
		}
		inputs = append(inputs, input)
	}
	return inputs
}
