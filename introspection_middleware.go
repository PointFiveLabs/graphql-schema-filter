package filter

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/vektah/gqlparser/v2/ast"
)

// IntrospectionFilterMiddleware is a gqlgen middleware that hides fields with
// listed: false from introspection while keeping them executable.
// This is a companion to GetFilteredSchema() for the build-time filtering model.
//
// For runtime filtering, use RuntimeFilterMiddleware instead — it handles
// introspection filtering as well as execution blocking.
type IntrospectionFilterMiddleware struct {
	Schema           *ast.Schema
	ExposeDirectives []string
}

func (IntrospectionFilterMiddleware) ExtensionName() string {
	return "SchemaFilterIntrospection"
}

func (IntrospectionFilterMiddleware) Validate(_ graphql.ExecutableSchema) error {
	return nil
}

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

func (m *IntrospectionFilterMiddleware) filterTypeFields(ctx context.Context, list []introspection.Field) []introspection.Field {
	fc := graphql.GetFieldContext(ctx)
	typeName := getParentTypeName(fc)
	if typeName == nil || (*typeName != "Query" && *typeName != "Mutation") {
		return list
	}

	astType := m.Schema.Types[*typeName]
	if astType == nil {
		return list
	}

	return filterFieldList(list, astType, func(astField *ast.FieldDefinition) bool {
		return !isUnlisted(astField.Directives, m.ExposeDirectives)
	})
}

// IsUnlisted returns true if the field has an expose directive with listed: false.
func (m *IntrospectionFilterMiddleware) IsUnlisted(directives ast.DirectiveList) bool {
	return isUnlisted(directives, m.ExposeDirectives)
}
