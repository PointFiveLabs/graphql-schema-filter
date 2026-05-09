package filter

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/vektah/gqlparser/v2/ast"
)

// IntrospectionFilterMiddleware is a gqlgen middleware that hides fields with
// listed: false from introspection while keeping them executable.
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

func getParentTypeName(fc *graphql.FieldContext) *string {
	if fc.Parent == nil || fc.Parent.Result == nil {
		return nil
	}
	typeResult, ok := fc.Parent.Result.(*introspection.Type)
	if !ok || typeResult == nil {
		return nil
	}
	return typeResult.Name()
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

	fList := make([]introspection.Field, 0, len(list))
	for _, field := range list {
		astField := astType.Fields.ForName(field.Name)
		if astField == nil {
			continue
		}
		if m.IsUnlisted(astField.Directives) {
			continue
		}
		fList = append(fList, field)
	}
	return fList
}

// IsUnlisted returns true if the field has an expose directive with listed: false.
func (m *IntrospectionFilterMiddleware) IsUnlisted(directives ast.DirectiveList) bool {
	return isUnlisted(directives, m.ExposeDirectives)
}

func isUnlisted(directives ast.DirectiveList, exposeDirectives []string) bool {
	for _, name := range exposeDirectives {
		d := directives.ForName(name)
		if d == nil {
			continue
		}
		arg := d.Arguments.ForName("listed")
		if arg != nil && arg.Value.Raw == "false" {
			return true
		}
	}
	return false
}
