package filter

import (
	"fmt"

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
//	    WithExposeDirective("expose"),
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

// GetRuntimeFilterMiddleware returns a gqlgen middleware that enforces schema
// filtering at runtime on a per-request basis. This enables a unified server
// where the same schema serves both internal and external clients, with the
// caller controlling when the middleware is active.
//
// The middleware applies the same expose/hide directive rules as GetFilteredSchema,
// but at request time instead of build time. It handles both execution blocking
// (preventing access to non-exposed fields) and introspection filtering (hiding
// fields from schema queries).
//
// See RuntimeFilterMiddleware for the full list of filtering rules.
func (fs *FilteredSchema) GetRuntimeFilterMiddleware() *RuntimeFilterMiddleware {
	return &RuntimeFilterMiddleware{
		Schema:  fs.Schema,
		options: fs.options,
	}
}

// GetIntrospectionMiddleware returns a gqlgen middleware that hides fields with
// listed: false from introspection while keeping them executable.
func (fs *FilteredSchema) GetIntrospectionMiddleware() *IntrospectionFilterMiddleware {
	return &IntrospectionFilterMiddleware{
		Schema:           fs.Schema,
		ExposeDirectives: fs.options.exposeDirectives,
	}
}

// GetFilteredSchema returns a new filtered ast schema out of the full schema,
// filtering out any fields, inputs, enums, types, queries & mutations that are not exposed.
// Returns an error if any expose directive is missing the required "listed" argument.
func (fs FilteredSchema) GetFilteredSchema() (*ast.Schema, error) {
	if err := fs.validateExposeDirectives(); err != nil {
		return nil, err
	}
	return &ast.Schema{
		// Filtering directives completely from the schema will make them unusable.
		// You should filter them from the Introspection Query instead.
		Directives:    fs.Schema.Directives,
		Types:         fs.filterTypes(fs.Schema.Types),
		Query:         fs.filterQueriesAndMutations(fs.Schema.Query),
		Mutation:      fs.filterQueriesAndMutations(fs.Schema.Mutation),
		PossibleTypes: fs.filterImplementsAndPossibleTypes(fs.Schema.PossibleTypes),
		Implements:    fs.filterImplementsAndPossibleTypes(fs.Schema.Implements),
	}, nil
}

// validateExposeDirectives checks that all usages of expose directives include the required
// "listed" argument. Returns an error for the first field that violates this.
func (fs FilteredSchema) validateExposeDirectives() error {
	for _, name := range fs.options.exposeDirectives {
		if name == "" {
			continue
		}
		for _, def := range fs.Schema.Types {
			if err := fs.validateDirectiveHasListed(name, "", def.Name, def.Directives); err != nil {
				return err
			}
			for _, field := range def.Fields {
				if err := fs.validateDirectiveHasListed(name, def.Name, field.Name, field.Directives); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (fs FilteredSchema) validateDirectiveHasListed(directiveName, typeName, fieldName string, directives ast.DirectiveList) error {
	d := directives.ForName(directiveName)
	if d == nil {
		return nil
	}
	if d.Arguments.ForName("listed") == nil {
		location := fieldName
		if typeName != "" {
			location = fmt.Sprintf("%s.%s", typeName, fieldName)
		}
		return fmt.Errorf("@%s directive on %s is missing required argument \"listed\" — use @%s(listed: true) or @%s(listed: false)",
			directiveName, location, directiveName, directiveName)
	}
	return nil
}

// MustGetFilteredSchema is like GetFilteredSchema but panics on validation errors.
// Useful during server initialization where invalid schema should prevent startup.
func (fs FilteredSchema) MustGetFilteredSchema() *ast.Schema {
	schema, err := fs.GetFilteredSchema()
	if err != nil {
		panic(fmt.Sprintf("schema filter validation failed: %v", err))
	}
	return schema
}

func (fs FilteredSchema) hasAnyDirective(directives ast.DirectiveList, directiveNames []string) bool {
	return hasAnyDirective(directives, directiveNames)
}

func hasAnyDirective(directives ast.DirectiveList, directiveNames []string) bool {
	for _, name := range directiveNames {
		if name != "" && directives.ForName(name) != nil {
			return true
		}
	}
	return false
}

func (fs FilteredSchema) shouldExposeFieldsByDirectives(directives ast.DirectiveList) bool {
	return !fs.hasAnyDirective(directives, fs.options.hideDirectives)
}

func (fs FilteredSchema) mustExposeTypesByDirectives(directives ast.DirectiveList) bool {
	if !fs.hasAnyDirective(directives, fs.options.exposeDirectives) {
		return false
	}

	return !fs.hasAnyDirective(directives, fs.options.hideDirectives)
}

func (fs FilteredSchema) filterDefinitionArguments(args []*ast.ArgumentDefinition) []*ast.ArgumentDefinition {
	return lo.Filter(args, func(d *ast.ArgumentDefinition, _ int) bool {
		return fs.shouldExposeFieldsByDirectives(d.Directives)
	})
}
