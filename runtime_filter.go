package filter

import (
	"context"
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/vektah/gqlparser/v2/ast"
)

// RuntimeFilterMiddleware filters fields at request time based on expose/hide directives.
// Blocks execution of non-exposed fields and hides them from introspection.
type RuntimeFilterMiddleware struct {
	schema  *ast.Schema
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
	if strings.HasPrefix(fc.Field.Name, "__") {
		return next(ctx)
	}

	isRootType := fc.Object == "Query" || fc.Object == "Mutation"
	if !isRootType && len(m.options.hideDirectives) == 0 {
		return next(ctx)
	}

	astType := m.schema.Types[fc.Object]
	if astType == nil {
		return next(ctx)
	}

	astField := astType.Fields.ForName(fc.Field.Name)
	if astField == nil {
		return next(ctx)
	}

	if isRootType {
		if !hasAnyDirective(astField.Directives, m.options.exposeDirectives) {
			return nil, fmt.Errorf("field '%s' is not accessible", fc.Field.Name)
		}
	} else {
		if hasAnyDirective(astField.Directives, m.options.hideDirectives) {
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
	typeName := getParentTypeName(fc)
	if typeName == nil {
		return res, nil
	}

	astType := m.schema.Types[*typeName]
	if astType == nil {
		return res, nil
	}

	fields, ok := res.([]introspection.Field)
	if !ok {
		return res, nil
	}

	isRootType := *typeName == "Query" || *typeName == "Mutation"
	if isRootType {
		return filterFieldList(fields, astType, func(astField *ast.FieldDefinition) bool {
			return hasAnyDirective(astField.Directives, m.options.exposeDirectives) &&
				!IsUnlisted(astField.Directives, m.options.exposeDirectives)
		}), nil
	}

	return filterFieldList(fields, astType, func(astField *ast.FieldDefinition) bool {
		return !hasAnyDirective(astField.Directives, m.options.hideDirectives)
	}), nil
}

func (m *RuntimeFilterMiddleware) filterIntrospectionEnumValues(fc *graphql.FieldContext, res any) (any, error) {
	typeName := getParentTypeName(fc)
	if typeName == nil {
		return res, nil
	}

	astType := m.schema.Types[*typeName]
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
		if !hasAnyDirective(astEnumValue.Directives, m.options.hideDirectives) {
			filtered = append(filtered, enumValue)
		}
	}
	return filtered, nil
}
