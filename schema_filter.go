package filter

import (
	"github.com/samber/lo"
	"github.com/vektah/gqlparser/v2/ast"
)

type FilteredSchema struct {
	Schema *ast.Schema

	exposeDirective            string
	hideDirective              string
	supportedBuiltInAttributes []string
}

var (
	builtInOperations = []string{"query", "mutation"}
	builtInTypes      = []string{"__schema", "__field", "__type", "__typekind", "__inputvalue", "__enumvalue", "__directive", "__directivelocation"}
)

func NewSchemaFilter(schema *ast.Schema, exposeDirective, hideDirective string, overrideBuiltInOperations *[]string) *FilteredSchema {
	if overrideBuiltInOperations != nil {
		builtInOperations = lo.FromPtr(overrideBuiltInOperations)
	}
	return &FilteredSchema{
		Schema:                     schema,
		exposeDirective:            exposeDirective,
		hideDirective:              hideDirective,
		supportedBuiltInAttributes: append(builtInOperations, builtInTypes...),
	}
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
