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

	switch fc.Object {
	case "__Type":
		return m.interceptTypeIntrospection(ctx, fc, next)
	case "__Schema":
		return m.interceptSchemaIntrospection(ctx, fc, next)
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

func (m *RuntimeFilterMiddleware) interceptTypeIntrospection(ctx context.Context, fc *graphql.FieldContext, next graphql.Resolver) (any, error) {
	res, err := next(ctx)
	if err != nil {
		return nil, err
	}

	switch fc.Field.Name {
	case "fields":
		return m.filterIntrospectionFields(fc, res)
	case "inputFields":
		return m.filterIntrospectionInputFields(fc, res)
	case "enumValues":
		return m.filterIntrospectionEnumValues(fc, res)
	case "interfaces", "possibleTypes":
		return m.filterIntrospectionTypeList(res)
	}

	return res, nil
}

func (m *RuntimeFilterMiddleware) interceptSchemaIntrospection(ctx context.Context, fc *graphql.FieldContext, next graphql.Resolver) (any, error) {
	res, err := next(ctx)
	if err != nil {
		return nil, err
	}

	if fc.Field.Name == "types" {
		return m.filterIntrospectionTypeList(res)
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
				!IsHiddenFromIntrospection(astField.Directives, m.options.exposeDirectives)
		}), nil
	}

	return filterFieldList(fields, astType, func(astField *ast.FieldDefinition) bool {
		return !hasAnyDirective(astField.Directives, m.options.hideDirectives)
	}), nil
}

func (m *RuntimeFilterMiddleware) filterIntrospectionInputFields(fc *graphql.FieldContext, res any) (any, error) {
	typeName := getParentTypeName(fc)
	if typeName == nil {
		return res, nil
	}

	astType := m.schema.Types[*typeName]
	if astType == nil || astType.Kind != ast.InputObject {
		return res, nil
	}

	inputFields, ok := res.([]introspection.InputValue)
	if !ok {
		return res, nil
	}

	filtered := make([]introspection.InputValue, 0, len(inputFields))
	for _, inputField := range inputFields {
		astField := astType.Fields.ForName(inputField.Name)
		if astField == nil {
			continue
		}
		if !hasAnyDirective(astField.Directives, m.options.hideDirectives) {
			filtered = append(filtered, inputField)
		}
	}
	return filtered, nil
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

func (m *RuntimeFilterMiddleware) filterIntrospectionTypeList(res any) (any, error) {
	types, ok := res.([]introspection.Type)
	if !ok {
		return res, nil
	}

	filtered := make([]introspection.Type, 0, len(types))
	for _, t := range types {
		name := t.Name()
		if name == nil {
			continue
		}
		def := m.schema.Types[*name]
		if def == nil {
			continue
		}
		if m.shouldExposeType(def) {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}

func (m *RuntimeFilterMiddleware) shouldExposeType(def *ast.Definition) bool {
	if strings.HasPrefix(def.Name, "__") {
		return true
	}
	if def.Kind == ast.Scalar {
		return true
	}
	if def.Name == "Query" || def.Name == "Mutation" {
		return true
	}
	return hasAnyDirective(def.Directives, m.options.exposeDirectives) &&
		!hasAnyDirective(def.Directives, m.options.hideDirectives)
}
