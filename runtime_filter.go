package filter

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/vektah/gqlparser/v2/ast"
)

// RuntimeFilterMiddleware is a gqlgen middleware that enforces schema filtering
// at runtime on a per-request basis. Unlike GetFilteredSchema() which creates a
// separate filtered schema at build time, this middleware operates on the full
// schema and applies filtering rules during request execution.
//
// This enables a unified server architecture where the same server can serve
// both internal clients (full schema) and external clients (filtered schema),
// with the filtering decision made per-request by the caller.
//
// The middleware handles both execution blocking and introspection filtering:
//   - Query/Mutation fields without an expose directive are blocked and hidden
//   - Fields with @expose(listed: false) are executable but hidden from introspection
//   - Fields with a hide directive on any type are blocked and hidden
//   - Enum values with a hide directive are hidden from introspection
type RuntimeFilterMiddleware struct {
	Schema  *ast.Schema
	options FilterOptions
}

func (RuntimeFilterMiddleware) ExtensionName() string {
	return "RuntimeFilter"
}

func (RuntimeFilterMiddleware) Validate(_ graphql.ExecutableSchema) error {
	return nil
}

func (m *RuntimeFilterMiddleware) InterceptField(ctx context.Context, next graphql.Resolver) (res any, err error) {
	fc := graphql.GetFieldContext(ctx)
	if fc == nil {
		return next(ctx)
	}

	if fc.Object == "__Type" {
		return m.interceptIntrospection(ctx, fc, next)
	}

	return m.interceptExecution(ctx, fc, next)
}

func (m *RuntimeFilterMiddleware) interceptExecution(ctx context.Context, fc *graphql.FieldContext, next graphql.Resolver) (any, error) {
	if len(fc.Field.Name) > 0 && fc.Field.Name[0] == '_' {
		return next(ctx)
	}

	isRootType := fc.Object == "Query" || fc.Object == "Mutation"
	if !isRootType && len(m.options.hideDirectives) == 0 {
		return next(ctx)
	}

	astType := m.Schema.Types[fc.Object]
	if astType == nil {
		return next(ctx)
	}

	astField := astType.Fields.ForName(fc.Field.Name)
	if astField == nil {
		return next(ctx)
	}

	if isRootType {
		if !m.hasAnyDirective(astField.Directives, m.options.exposeDirectives) {
			return nil, fmt.Errorf("field '%s' is not accessible", fc.Field.Name)
		}
	} else {
		if m.hasAnyDirective(astField.Directives, m.options.hideDirectives) {
			return nil, fmt.Errorf("field '%s' is not accessible", fc.Field.Name)
		}
	}

	return next(ctx)
}

func (m *RuntimeFilterMiddleware) interceptIntrospection(ctx context.Context, fc *graphql.FieldContext, next graphql.Resolver) (any, error) {
	res, err := next(ctx)
	if err != nil {
		return nil, err
	}

	switch fc.Field.Name {
	case "fields":
		return m.filterIntrospectionFields(fc, res)
	case "enumValues":
		return m.filterIntrospectionEnumValues(fc, res)
	}

	return res, nil
}

func (m *RuntimeFilterMiddleware) filterIntrospectionFields(fc *graphql.FieldContext, res any) (any, error) {
	typeName := m.getParentTypeName(fc)
	if typeName == nil {
		return res, nil
	}

	astType := m.Schema.Types[*typeName]
	if astType == nil {
		return res, nil
	}

	fields, ok := res.([]introspection.Field)
	if !ok {
		return res, nil
	}

	isRootType := *typeName == "Query" || *typeName == "Mutation"
	filtered := make([]introspection.Field, 0, len(fields))
	for _, field := range fields {
		astField := astType.Fields.ForName(field.Name)
		if astField == nil {
			continue
		}
		if isRootType {
			if !m.hasAnyDirective(astField.Directives, m.options.exposeDirectives) {
				continue
			}
			if m.isUnlisted(astField.Directives) {
				continue
			}
			filtered = append(filtered, field)
		} else {
			if !m.hasAnyDirective(astField.Directives, m.options.hideDirectives) {
				filtered = append(filtered, field)
			}
		}
	}
	return filtered, nil
}

func (m *RuntimeFilterMiddleware) filterIntrospectionEnumValues(fc *graphql.FieldContext, res any) (any, error) {
	typeName := m.getParentTypeName(fc)
	if typeName == nil {
		return res, nil
	}

	astType := m.Schema.Types[*typeName]
	if astType == nil || astType.Kind != ast.Enum {
		return res, nil
	}

	enumValues, ok := res.([]introspection.EnumValue)
	if !ok {
		return res, nil
	}

	filtered := make([]introspection.EnumValue, 0, len(enumValues))
	for _, enumValue := range enumValues {
		astEnumValue := astType.EnumValues.ForName(enumValue.Name)
		if astEnumValue == nil {
			continue
		}
		if !m.hasAnyDirective(astEnumValue.Directives, m.options.hideDirectives) {
			filtered = append(filtered, enumValue)
		}
	}
	return filtered, nil
}

func (m *RuntimeFilterMiddleware) getParentTypeName(fc *graphql.FieldContext) *string {
	if fc.Parent == nil || fc.Parent.Result == nil {
		return nil
	}
	typeResult, ok := fc.Parent.Result.(*introspection.Type)
	if !ok || typeResult == nil {
		return nil
	}
	return typeResult.Name()
}

func (m *RuntimeFilterMiddleware) hasAnyDirective(directives ast.DirectiveList, directiveNames []string) bool {
	for _, name := range directiveNames {
		if name != "" && directives.ForName(name) != nil {
			return true
		}
	}
	return false
}

func (m *RuntimeFilterMiddleware) isUnlisted(directives ast.DirectiveList) bool {
	for _, name := range m.options.exposeDirectives {
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
