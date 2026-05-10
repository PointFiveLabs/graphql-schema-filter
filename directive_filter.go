package filter

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/vektah/gqlparser/v2/ast"
)

// DirectiveFilterMiddleware filters which directives appear in __schema { directives }
// introspection responses. The filter function returns true for directives to include.
type DirectiveFilterMiddleware struct {
	schema          *ast.Schema
	directiveFilter func(name string) bool
}

func (DirectiveFilterMiddleware) ExtensionName() string {
	return "DirectiveFilter"
}

func (DirectiveFilterMiddleware) Validate(_ graphql.ExecutableSchema) error {
	return nil
}

func (m *DirectiveFilterMiddleware) InterceptField(ctx context.Context, next graphql.Resolver) (res any, err error) {
	res, err = next(ctx)
	if err != nil {
		return nil, err
	}

	fc := graphql.GetFieldContext(ctx)
	if fc.Object == "__Schema" && fc.Field.Name == "directives" {
		if res == nil {
			return nil, nil
		}
		return m.filterDirectives(res.([]introspection.Directive)), nil
	}

	return res, err
}

func (m *DirectiveFilterMiddleware) filterDirectives(list []introspection.Directive) []introspection.Directive {
	if m.directiveFilter == nil {
		return list
	}
	filtered := make([]introspection.Directive, 0, len(list))
	for _, directive := range list {
		if m.directiveFilter(directive.Name) {
			filtered = append(filtered, directive)
		}
	}
	return filtered
}
