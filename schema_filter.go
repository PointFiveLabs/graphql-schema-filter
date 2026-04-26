package filter

import (
	"github.com/samber/lo"
	"github.com/vektah/gqlparser/v2/ast"
)

type FilteredSchema struct {
	Schema                     *ast.Schema
	options                    FilterOptions
	supportedBuiltInAttributes []string
}

var builtInTypes = []string{"__schema", "__field", "__type", "__typekind", "__inputvalue", "__enumvalue", "__directive", "__directivelocation"}

// NewSchemaFilterWithOptions creates a new schema filter with flexible options.
//
// Example:
//
//	filter := NewSchemaFilterWithOptions(
//	    schema,
//	    WithPublicDirective("public"),
//	    WithHideDirective("hide"),
//	)
func NewSchemaFilterWithOptions(schema *ast.Schema, opts ...Option) *FilteredSchema {
	options := FilterOptions{
		builtInOperations: []string{"query", "mutation"},
	}

	for _, opt := range opts {
		opt(&options)
	}

	return &FilteredSchema{
		Schema:                     schema,
		options:                    options,
		supportedBuiltInAttributes: append(options.builtInOperations, builtInTypes...),
	}
}

// GetIntrospectionMiddleware returns a gqlgen middleware that hides @public(listed: false)
// fields from introspection while keeping them executable.
func (fs *FilteredSchema) GetIntrospectionMiddleware() *IntrospectionFilterMiddleware {
	return &IntrospectionFilterMiddleware{
		Schema:           fs.Schema,
		PublicDirectives: fs.options.publicDirectives,
	}
}

// NewSchemaFilter creates a new schema filter using the legacy API.
// Deprecated: Use NewSchemaFilterWithOptions instead for more flexibility.
func NewSchemaFilter(schema *ast.Schema, publicDirective, hideDirective string, overrideBuiltInOperations *[]string) *FilteredSchema {
	opts := []Option{}

	if publicDirective != "" {
		opts = append(opts, WithPublicDirective(publicDirective))
	}
	if hideDirective != "" {
		opts = append(opts, WithHideDirective(hideDirective))
	}
	if overrideBuiltInOperations != nil {
		opts = append(opts, WithBuiltInOperations(*overrideBuiltInOperations))
	}

	return NewSchemaFilterWithOptions(schema, opts...)
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

// hasAnyDirective checks if any of the given directive names exist in the directive list
func (fs FilteredSchema) hasAnyDirective(directives ast.DirectiveList, directiveNames []string) bool {
	for _, name := range directiveNames {
		if name != "" && directives.ForName(name) != nil {
			return true
		}
	}
	return false
}

// shouldExposeFieldsByDirectives checks if a field should be included in the filtered schema.
// Returns true if the field does NOT have a hide directive.
func (fs FilteredSchema) shouldExposeFieldsByDirectives(directives ast.DirectiveList) bool {
	return !fs.hasAnyDirective(directives, fs.options.hideDirectives)
}

// mustExposeTypesByDirectives checks if a Query/Mutation field or type must be exposed.
// Returns true if the field/type has a @public directive, and does NOT have @hide.
func (fs FilteredSchema) mustExposeTypesByDirectives(directives ast.DirectiveList) bool {
	if !fs.hasAnyDirective(directives, fs.options.publicDirectives) {
		return false
	}

	return !fs.hasAnyDirective(directives, fs.options.hideDirectives)
}

func (fs FilteredSchema) filterDefinitionArguments(args []*ast.ArgumentDefinition) []*ast.ArgumentDefinition {
	return lo.Filter(args, func(d *ast.ArgumentDefinition, _ int) bool {
		return fs.shouldExposeFieldsByDirectives(d.Directives)
	})
}
