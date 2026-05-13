package filter_test

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	filter "github.com/PointFiveLabs/graphql-schema-filter/v3"
)

func getRuntimeTestSchema() *ast.Schema {
	return &ast.Schema{
		Types: map[string]*ast.Definition{
			"Query": {
				Name: "Query",
				Kind: ast.Object,
				Fields: ast.FieldList{
					{Name: "publicQuery", Directives: exposeDirective()},
					{Name: "hiddenQuery", Directives: []*ast.Directive{{
						Name:      "expose",
						Arguments: []*ast.Argument{{Name: "introspectable", Value: &ast.Value{Raw: "false", Kind: ast.BooleanValue}}},
					}}},
					{Name: "internalQuery"},
				},
			},
			"Mutation": {
				Name: "Mutation",
				Kind: ast.Object,
				Fields: ast.FieldList{
					{Name: "publicMutation", Directives: exposeDirective()},
					{Name: "internalMutation"},
				},
			},
			"User": {
				Name:       "User",
				Kind:       ast.Object,
				Directives: exposeDirective(),
				Fields: ast.FieldList{
					{Name: "id"},
					{Name: "name"},
					{Name: "internalData", Directives: []*ast.Directive{{Name: "hide"}}},
				},
			},
			"CreateUserInput": {
				Name:       "CreateUserInput",
				Kind:       ast.InputObject,
				Directives: exposeDirective(),
				Fields: ast.FieldList{
					{Name: "name"},
					{Name: "email"},
					{Name: "internalFlag", Directives: []*ast.Directive{{Name: "hide"}}},
				},
			},
			"InternalType": {
				Name:   "InternalType",
				Kind:   ast.Object,
				Fields: ast.FieldList{{Name: "id"}},
			},
			"Status": {
				Name:       "Status",
				Kind:       ast.Enum,
				Directives: exposeDirective(),
				EnumValues: ast.EnumValueList{
					{Name: "ACTIVE"},
					{Name: "INACTIVE"},
					{Name: "INTERNAL_ONLY", Directives: []*ast.Directive{{Name: "hide"}}},
				},
			},
		},
	}
}

func newRuntimeFilter(schema *ast.Schema) *filter.RuntimeFilterMiddleware {
	return filter.NewSchemaFilterWithOptions(
		schema,
		filter.WithExposeDirective("expose"),
		filter.WithHideDirective("hide"),
	).GetRuntimeFilterMiddleware()
}

func withFieldCtx(ctx context.Context, object, fieldName string) context.Context {
	return graphql.WithFieldContext(ctx, &graphql.FieldContext{
		Object: object,
		Field: graphql.CollectedField{
			Field: &ast.Field{Name: fieldName},
		},
	})
}

func TestRuntimeFilter_ExecutionBlocking(t *testing.T) {
	schema := getRuntimeTestSchema()
	middleware := newRuntimeFilter(schema)

	next := func(ctx context.Context) (any, error) {
		return "ok", nil
	}

	testCases := []struct {
		name        string
		object      string
		fieldName   string
		expectError bool
	}{
		{
			name:        "exposed query is allowed",
			object:      "Query",
			fieldName:   "publicQuery",
			expectError: false,
		},
		{
			name:        "non-introspectable query is allowed (executable but hidden from introspection)",
			object:      "Query",
			fieldName:   "hiddenQuery",
			expectError: false,
		},
		{
			name:        "internal query is blocked",
			object:      "Query",
			fieldName:   "internalQuery",
			expectError: true,
		},
		{
			name:        "exposed mutation is allowed",
			object:      "Mutation",
			fieldName:   "publicMutation",
			expectError: false,
		},
		{
			name:        "internal mutation is blocked",
			object:      "Mutation",
			fieldName:   "internalMutation",
			expectError: true,
		},
		{
			name:        "regular field on nested type is allowed",
			object:      "User",
			fieldName:   "name",
			expectError: false,
		},
		{
			name:        "hidden field on nested type is blocked",
			object:      "User",
			fieldName:   "internalData",
			expectError: true,
		},
		{
			name:        "introspection field __typename is allowed",
			object:      "Query",
			fieldName:   "__typename",
			expectError: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := withFieldCtx(context.Background(), testCase.object, testCase.fieldName)

			result, err := middleware.InterceptField(ctx, next)
			if testCase.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "is not accessible")
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "ok", result)
			}
		})
	}
}

func TestRuntimeFilter_IntrospectionFieldFiltering(t *testing.T) {
	schema := getRuntimeTestSchema()
	middleware := newRuntimeFilter(schema)

	queryName := "Query"
	userName := "User"

	testCases := []struct {
		name           string
		typeName       *string
		inputFields    []introspection.Field
		expectedFields []string
	}{
		{
			name:     "root type hides non-exposed and non-introspectable fields",
			typeName: &queryName,
			inputFields: []introspection.Field{
				{Name: "publicQuery"},
				{Name: "hiddenQuery"},
				{Name: "internalQuery"},
			},
			expectedFields: []string{"publicQuery"},
		},
		{
			name:     "nested type hides @hide fields",
			typeName: &userName,
			inputFields: []introspection.Field{
				{Name: "id"},
				{Name: "name"},
				{Name: "internalData"},
			},
			expectedFields: []string{"id", "name"},
		},
	}

	next := func(fields []introspection.Field, typeName *string) graphql.Resolver {
		return func(ctx context.Context) (any, error) {
			return fields, nil
		}
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parentFC := &graphql.FieldContext{}
			if testCase.typeName != nil {
				def := schema.Types[*testCase.typeName]
				parentFC.Result = introspection.WrapTypeFromDef(schema, def)
			}

			// WithFieldContext auto-sets Parent from previous context,
			// so we nest: parent first, then child
			ctx := graphql.WithFieldContext(context.Background(), parentFC)
			childFC := &graphql.FieldContext{
				Object: "__Type",
				Field: graphql.CollectedField{
					Field: &ast.Field{Name: "fields"},
				},
			}
			ctx = graphql.WithFieldContext(ctx, childFC)

			result, err := middleware.InterceptField(ctx, next(testCase.inputFields, testCase.typeName))
			require.NoError(t, err)

			fields, ok := result.([]introspection.Field)
			require.True(t, ok)

			fieldNames := make([]string, len(fields))
			for i, f := range fields {
				fieldNames[i] = f.Name
			}
			assert.Equal(t, testCase.expectedFields, fieldNames)
		})
	}
}

func TestRuntimeFilter_IntrospectionEnumFiltering(t *testing.T) {
	schema := getRuntimeTestSchema()
	middleware := newRuntimeFilter(schema)

	statusName := "Status"

	parentFC := &graphql.FieldContext{
		Result: introspection.WrapTypeFromDef(schema, schema.Types[statusName]),
	}
	ctx := graphql.WithFieldContext(context.Background(), parentFC)
	childFC := &graphql.FieldContext{
		Object: "__Type",
		Field: graphql.CollectedField{
			Field: &ast.Field{Name: "enumValues"},
		},
	}
	ctx = graphql.WithFieldContext(ctx, childFC)

	next := func(ctx context.Context) (any, error) {
		return []introspection.EnumValue{
			{Name: "ACTIVE"},
			{Name: "INACTIVE"},
			{Name: "INTERNAL_ONLY"},
		}, nil
	}

	result, err := middleware.InterceptField(ctx, next)
	require.NoError(t, err)

	enumValues, ok := result.([]introspection.EnumValue)
	require.True(t, ok)
	assert.Len(t, enumValues, 2)
	assert.Equal(t, "ACTIVE", enumValues[0].Name)
	assert.Equal(t, "INACTIVE", enumValues[1].Name)
}

func TestRuntimeFilter_IntrospectionInputFieldFiltering(t *testing.T) {
	schema := getRuntimeTestSchema()
	middleware := newRuntimeFilter(schema)

	inputTypeName := "CreateUserInput"

	parentFC := &graphql.FieldContext{
		Result: introspection.WrapTypeFromDef(schema, schema.Types[inputTypeName]),
	}
	ctx := graphql.WithFieldContext(context.Background(), parentFC)
	childFC := &graphql.FieldContext{
		Object: "__Type",
		Field: graphql.CollectedField{
			Field: &ast.Field{Name: "inputFields"},
		},
	}
	ctx = graphql.WithFieldContext(ctx, childFC)

	next := func(ctx context.Context) (any, error) {
		return []introspection.InputValue{
			{Name: "name"},
			{Name: "email"},
			{Name: "internalFlag"},
		}, nil
	}

	result, err := middleware.InterceptField(ctx, next)
	require.NoError(t, err)

	inputFields, ok := result.([]introspection.InputValue)
	require.True(t, ok)
	assert.Len(t, inputFields, 2)
	assert.Equal(t, "name", inputFields[0].Name)
	assert.Equal(t, "email", inputFields[1].Name)
}

func TestRuntimeFilter_IntrospectionSchemaTypes(t *testing.T) {
	schema := getRuntimeTestSchema()
	middleware := newRuntimeFilter(schema)

	ctx := graphql.WithFieldContext(context.Background(), &graphql.FieldContext{
		Object: "__Schema",
		Field: graphql.CollectedField{
			Field: &ast.Field{Name: "types"},
		},
	})

	next := func(ctx context.Context) (any, error) {
		var types []introspection.Type
		for name := range schema.Types {
			def := schema.Types[name]
			types = append(types, *introspection.WrapTypeFromDef(schema, def))
		}
		return types, nil
	}

	result, err := middleware.InterceptField(ctx, next)
	require.NoError(t, err)

	types, ok := result.([]introspection.Type)
	require.True(t, ok)

	typeNames := make(map[string]bool)
	for _, t := range types {
		if t.Name() != nil {
			typeNames[*t.Name()] = true
		}
	}

	assert.True(t, typeNames["Query"], "root type Query should be visible")
	assert.True(t, typeNames["Mutation"], "root type Mutation should be visible")
	assert.True(t, typeNames["User"], "exposed type User should be visible")
	assert.True(t, typeNames["CreateUserInput"], "exposed type CreateUserInput should be visible")
	assert.True(t, typeNames["Status"], "exposed type Status should be visible")
	assert.False(t, typeNames["InternalType"], "unexposed type InternalType should be hidden")
}

func TestRuntimeFilter_IntrospectionTypeListFiltering(t *testing.T) {
	schema := getRuntimeTestSchema()
	middleware := newRuntimeFilter(schema)

	userDef := schema.Types["User"]
	parentFC := &graphql.FieldContext{
		Result: introspection.WrapTypeFromDef(schema, userDef),
	}
	ctx := graphql.WithFieldContext(context.Background(), parentFC)
	childFC := &graphql.FieldContext{
		Object: "__Type",
		Field: graphql.CollectedField{
			Field: &ast.Field{Name: "interfaces"},
		},
	}
	ctx = graphql.WithFieldContext(ctx, childFC)

	next := func(ctx context.Context) (any, error) {
		return []introspection.Type{
			*introspection.WrapTypeFromDef(schema, schema.Types["User"]),
			*introspection.WrapTypeFromDef(schema, schema.Types["InternalType"]),
		}, nil
	}

	result, err := middleware.InterceptField(ctx, next)
	require.NoError(t, err)

	types, ok := result.([]introspection.Type)
	require.True(t, ok)
	assert.Len(t, types, 1)
	assert.Equal(t, "User", *types[0].Name())
}

func TestRuntimeFilter_NilFieldContext(t *testing.T) {
	schema := getRuntimeTestSchema()
	middleware := newRuntimeFilter(schema)

	next := func(ctx context.Context) (any, error) {
		return "ok", nil
	}

	result, err := middleware.InterceptField(context.Background(), next)
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}

func TestRuntimeFilter_UnknownTypePassesThrough(t *testing.T) {
	schema := getRuntimeTestSchema()
	middleware := newRuntimeFilter(schema)

	next := func(ctx context.Context) (any, error) {
		return "ok", nil
	}

	ctx := withFieldCtx(context.Background(), "UnknownType", "someField")
	result, err := middleware.InterceptField(ctx, next)
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}

func TestRuntimeFilter_NoHideDirectives(t *testing.T) {
	schema := &ast.Schema{
		Types: map[string]*ast.Definition{
			"Query": {
				Name: "Query",
				Kind: ast.Object,
				Fields: ast.FieldList{
					{Name: "publicQuery", Directives: exposeDirective()},
					{Name: "internalQuery"},
				},
			},
			"User": {
				Name: "User",
				Kind: ast.Object,
				Fields: ast.FieldList{
					{Name: "id"},
					{Name: "name"},
					{Name: "internalData", Directives: []*ast.Directive{{Name: "hide"}}},
				},
			},
		},
	}

	middleware := filter.NewSchemaFilterWithOptions(
		schema,
		filter.WithExposeDirective("expose"),
	).GetRuntimeFilterMiddleware()

	next := func(ctx context.Context) (any, error) {
		return "ok", nil
	}

	testCases := []struct {
		name        string
		object      string
		fieldName   string
		expectError bool
	}{
		{
			name:        "exposed root field is allowed",
			object:      "Query",
			fieldName:   "publicQuery",
			expectError: false,
		},
		{
			name:        "non-exposed root field is blocked",
			object:      "Query",
			fieldName:   "internalQuery",
			expectError: true,
		},
		{
			name:        "nested type field passes through without hide directives",
			object:      "User",
			fieldName:   "name",
			expectError: false,
		},
		{
			name:        "nested type @hide field passes through when no hide directives configured",
			object:      "User",
			fieldName:   "internalData",
			expectError: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := withFieldCtx(context.Background(), testCase.object, testCase.fieldName)
			result, err := middleware.InterceptField(ctx, next)
			if testCase.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "is not accessible")
			} else {
				require.NoError(t, err)
				assert.Equal(t, "ok", result)
			}
		})
	}
}
