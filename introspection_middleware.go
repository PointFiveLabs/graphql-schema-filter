package filter

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/vektah/gqlparser/v2/ast"
)

// IntrospectionFilterMiddleware is a gqlgen middleware that filters fields from introspection
// based on directives. This works in conjunction with the SchemaFilter to provide runtime
// introspection control for fields that are included in the schema but should be hidden.
type IntrospectionFilterMiddleware struct {
	Schema             *ast.Schema
	InternalDirectives []string
}

// NewIntrospectionFilterMiddleware creates a middleware that hides fields with internal directives
// from GraphQL introspection queries.
func NewIntrospectionFilterMiddleware(schema *ast.Schema, internalDirectives []string) *IntrospectionFilterMiddleware {
	return &IntrospectionFilterMiddleware{
		Schema:             schema,
		InternalDirectives: internalDirectives,
	}
}

func (IntrospectionFilterMiddleware) ExtensionName() string {
	return "SchemaFilterIntrospection"
}

func (IntrospectionFilterMiddleware) Validate(_ graphql.ExecutableSchema) error {
	return nil
}

// InterceptField intercepts introspection queries to hide fields marked with internal directives
func (m *IntrospectionFilterMiddleware) InterceptField(ctx context.Context, next graphql.Resolver) (res any, err error) {
	res, err = next(ctx)
	if err != nil {
		return nil, err
	}

	fc := graphql.GetFieldContext(ctx)
	if fc.Object == "__Type" && fc.Field.Name == "fields" {
		if res == nil {
			return nil, nil
		}
		return m.filterTypeFields(ctx, res.([]introspection.Field)), nil
	}

	return res, err
}

// filterTypeFields filters Query/Mutation fields to hide those with internal directives
func (m *IntrospectionFilterMiddleware) filterTypeFields(ctx context.Context, list []introspection.Field) []introspection.Field {
	fc := graphql.GetFieldContext(ctx)
	if fc == nil || fc.Parent == nil || fc.Parent.Result == nil {
		return list
	}

	typeResult, ok := fc.Parent.Result.(*introspection.Type)
	if !ok || typeResult == nil {
		return list
	}

	typeName := typeResult.Name()
	if typeName == nil {
		return list
	}

	// Only filter Query/Mutation fields
	if *typeName != "Query" && *typeName != "Mutation" {
		return list
	}

	astType := m.Schema.Types[*typeName]
	if astType == nil {
		return list
	}

	fList := make([]introspection.Field, 0, len(list))
	for _, field := range list {
		astField := astType.Fields.ForName(field.Name)
		if astField == nil {
			continue
		}

		// Check if field has any internal directive
		hasInternal := false
		for _, internalDir := range m.InternalDirectives {
			if astField.Directives.ForName(internalDir) != nil {
				hasInternal = true
				break
			}
		}

		// Hide fields with internal directive from introspection
		if !hasInternal {
			fList = append(fList, field)
		}
	}
	return fList
}
