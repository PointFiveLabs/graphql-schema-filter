package filter

import (
	"strings"

	"github.com/samber/lo"
	"github.com/vektah/gqlparser/v2/ast"
)

type FilteredSchema struct {
	Schema *ast.Schema

	exposeDirective            string
	hideDirective              string
	supportedBuiltInAttributes []string
	referencedTypes            map[string]bool // Types referenced by exposed operations
}

var (
	builtInOperations = []string{"query", "mutation"}
	builtInTypes      = []string{"__schema", "__field", "__type", "__typekind", "__inputvalue", "__enumvalue", "__directive", "__directivelocation"}
)

func NewSchemaFilter(schema *ast.Schema, exposeDirective, hideDirective string, overrideBuiltInOperations *[]string) *FilteredSchema {
	if overrideBuiltInOperations != nil {
		builtInOperations = lo.FromPtr(overrideBuiltInOperations)
	}
	fs := &FilteredSchema{
		Schema:                     schema,
		exposeDirective:            exposeDirective,
		hideDirective:              hideDirective,
		supportedBuiltInAttributes: append(builtInOperations, builtInTypes...),
		referencedTypes:            make(map[string]bool),
	}
	// Collect all types referenced by exposed operations
	fs.collectReferencedTypes()
	return fs
}

// GetFilteredSchema returns a new filtered ast schema out of the full schema,
// filtering out any fields, inputs, enums, types, queries & mutations that are not exposed.
func (fs FilteredSchema) GetFilteredSchema() *ast.Schema {
	return &ast.Schema{
		// Filtering directives completely from the schema will make them unusable.
		// You should filter them from the Introspection Query instead.
		Directives:    fs.Schema.Directives,
		Types:         fs.filterTypes(fs.Schema.Types),
		Query:         fs.filterQueriesAndMutations(fs.Schema.Query),
		Mutation:      fs.filterQueriesAndMutations(fs.Schema.Mutation),
		PossibleTypes: fs.filterImplementsAndPossibleTypes(fs.Schema.PossibleTypes),
		Implements:    fs.filterImplementsAndPossibleTypes(fs.Schema.Implements),
	}
}

func (fs FilteredSchema) shouldExposeFieldsByDirectives(directives ast.DirectiveList) bool {
	// Returns true if "hide" is not present
	return directives.ForName(fs.hideDirective) == nil
}

func (fs FilteredSchema) mustExposeTypesByDirectives(directives ast.DirectiveList) bool {
	// Returns true if "expose" directive exists and "hide" is not present
	return directives.ForName(fs.exposeDirective) != nil && directives.ForName(fs.hideDirective) == nil
}

func (fs FilteredSchema) filterDefinitionArguments(args []*ast.ArgumentDefinition) []*ast.ArgumentDefinition {
	return lo.Filter(args, func(d *ast.ArgumentDefinition, _ int) bool {
		return fs.shouldExposeFieldsByDirectives(d.Directives)
	})
}

// collectReferencedTypes traverses exposed queries and mutations and collects all referenced types
func (fs *FilteredSchema) collectReferencedTypes() {
	// Collect from Query operations
	if fs.Schema.Query != nil {
		for _, field := range fs.Schema.Query.Fields {
			if fs.mustExposeTypesByDirectives(field.Directives) {
				fs.collectTypesFromField(field)
			}
		}
	}

	// Collect from Mutation operations
	if fs.Schema.Mutation != nil {
		for _, field := range fs.Schema.Mutation.Fields {
			if fs.mustExposeTypesByDirectives(field.Directives) {
				fs.collectTypesFromField(field)
			}
		}
	}
}

// collectTypesFromField recursively collects all types referenced by a field
func (fs *FilteredSchema) collectTypesFromField(field *ast.FieldDefinition) {
	// Collect return type
	fs.collectTypesFromType(field.Type)

	// Collect argument types
	for _, arg := range field.Arguments {
		fs.collectTypesFromType(arg.Type)
	}
}

// collectTypesFromType recursively collects types, handling lists and non-null wrappers
func (fs *FilteredSchema) collectTypesFromType(t *ast.Type) {
	if t == nil {
		return
	}

	// Unwrap lists and non-null types to get to the actual type name
	typeName := t.Name()
	if typeName == "" {
		return
	}

	// Skip if already collected or if it's a built-in scalar
	if fs.referencedTypes[typeName] {
		return
	}

	typeDef := fs.Schema.Types[typeName]
	if typeDef == nil {
		return
	}

	// Skip built-in scalars and introspection types
	if typeDef.Kind == ast.Scalar || lo.Contains(builtInTypes, strings.ToLower(typeName)) {
		return
	}

	// Mark this type as referenced
	fs.referencedTypes[typeName] = true

	// Recursively collect types from this type's fields
	switch typeDef.Kind {
	case ast.Object, ast.Interface:
		for _, field := range typeDef.Fields {
			fs.collectTypesFromType(field.Type)
			// Also collect argument types
			for _, arg := range field.Arguments {
				fs.collectTypesFromType(arg.Type)
			}
		}
	case ast.InputObject:
		for _, field := range typeDef.Fields {
			fs.collectTypesFromType(field.Type)
		}
	case ast.Union:
		for _, unionType := range typeDef.Types {
			fs.referencedTypes[unionType] = true
			// Recursively collect from union member types
			if memberDef := fs.Schema.Types[unionType]; memberDef != nil {
				for _, field := range memberDef.Fields {
					fs.collectTypesFromType(field.Type)
				}
			}
		}
	}
}
