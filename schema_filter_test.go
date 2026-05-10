package filter_test

import (
	"testing"

	filter "github.com/PointFiveLabs/graphql-schema-filter/v2"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestSchemaFiltering(t *testing.T) {
	fullSchema := getTestSchema()

	schemaFilter := filter.NewSchemaFilterWithOptions(fullSchema,
		filter.WithExposeDirective("expose"),
		filter.WithHideDirective("hide"),
	)
	filteredSchema, err := schemaFilter.GetFilteredSchema()
	assert.NoError(t, err)

	tests := []struct {
		name            string
		typeName        string
		shouldTypeExist bool
		fields          map[string]bool
	}{
		{
			name:            "Todo Type Exists, isGlobal is Hidden",
			typeName:        "Todo",
			shouldTypeExist: true,
			fields: map[string]bool{
				"id":       true,
				"text":     true,
				"done":     true,
				"user":     true,
				"isGlobal": false,
			},
		},
		{
			name:            "User Type Exists, All Fields Exposed",
			typeName:        "User",
			shouldTypeExist: true,
			fields: map[string]bool{
				"id":   true,
				"name": true,
			},
		},
		{
			name:            "NewTodo Input Exists, isGlobal is Hidden",
			typeName:        "NewTodo",
			shouldTypeExist: true,
			fields: map[string]bool{
				"text":     true,
				"userId":   true,
				"isGlobal": false,
			},
		},
		{
			name:            "Query Type Exists, Internal Query is Hidden",
			typeName:        "Query",
			shouldTypeExist: true,
			fields: map[string]bool{
				"todos":         true,
				"internalQuery": false,
			},
		},
		{
			name:            "Mutation Type Exists, Internal Mutation is Hidden",
			typeName:        "Mutation",
			shouldTypeExist: true,
			fields: map[string]bool{
				"createTodo":       true,
				"internalMutation": false,
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			checkTypeAndFields(t, filteredSchema, testCase.typeName, testCase.shouldTypeExist, testCase.fields)
		})
	}
}

func checkTypeAndFields(t *testing.T, filteredSchema *ast.Schema, typeName string, shouldTypeExist bool, fields map[string]bool) {
	typ, typeExists := filteredSchema.Types[typeName]
	assert.Equal(t, shouldTypeExist, typeExists, "Expected type existence to be %v for type %s", shouldTypeExist, typeName)

	if typeExists {
		for fieldName, shouldFieldExist := range fields {
			fieldExists := false
			for _, field := range typ.Fields {
				if field.Name == fieldName {
					fieldExists = true
					break
				}
			}
			assert.Equal(t, shouldFieldExist, fieldExists, "Expected field %s in type %s to exist: %v", fieldName, typeName, shouldFieldExist)
		}
	}
}

func getTestSchema() *ast.Schema {
	return &ast.Schema{
		Query:    createQuery(),
		Mutation: createMutation(),
		Types:    createTypes(),
	}
}

func createQuery() *ast.Definition {
	return &ast.Definition{
		Name: "Query",
		Fields: []*ast.FieldDefinition{
			{Name: "todos", Directives: exposeDirective()},
			{Name: "internalQuery"},
		},
	}
}

func createMutation() *ast.Definition {
	return &ast.Definition{
		Name: "Mutation",
		Fields: []*ast.FieldDefinition{
			{Name: "createTodo", Directives: exposeDirective(), Arguments: []*ast.ArgumentDefinition{
				{Name: "input", Type: ast.NonNullNamedType("NewTodo", nil)},
			}},
			{Name: "internalMutation"},
		},
	}
}

func TestIsUnlisted(t *testing.T) {
	schema := &ast.Schema{
		Types: map[string]*ast.Definition{
			"Query": {
				Name: "Query",
				Kind: ast.Object,
				Fields: []*ast.FieldDefinition{
					{
						Name: "exposedListed",
						Directives: []*ast.Directive{{
							Name: "expose",
							Arguments: []*ast.Argument{{
								Name:  "listed",
								Value: &ast.Value{Raw: "true", Kind: ast.BooleanValue},
							}},
						}},
					},
					{
						Name: "exposedUnlisted",
						Directives: []*ast.Directive{{
							Name: "expose",
							Arguments: []*ast.Argument{{
								Name:  "listed",
								Value: &ast.Value{Raw: "false", Kind: ast.BooleanValue},
							}},
						}},
					},
					{
						Name:       "noDirective",
						Directives: nil,
					},
				},
			},
		},
	}

	exposeDirectives := []string{"expose"}

	tests := []struct {
		name       string
		fieldName  string
		shouldHide bool
	}{
		{
			name:       "listed field should not be hidden",
			fieldName:  "exposedListed",
			shouldHide: false,
		},
		{
			name:       "unlisted field should be hidden",
			fieldName:  "exposedUnlisted",
			shouldHide: true,
		},
		{
			name:       "field without directive should not be hidden",
			fieldName:  "noDirective",
			shouldHide: false,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			astField := schema.Types["Query"].Fields.ForName(testCase.fieldName)
			assert.NotNil(t, astField)
			assert.Equal(t, testCase.shouldHide, filter.IsUnlisted(astField.Directives, exposeDirectives))
		})
	}
}

func TestGetFilteredSchema_RejectsExposeDirectiveWithoutListed(t *testing.T) {
	tests := []struct {
		name        string
		schema      *ast.Schema
		expectedErr string
	}{
		{
			name: "bare @expose on Query field",
			schema: &ast.Schema{
				Query: &ast.Definition{
					Name: "Query",
					Fields: []*ast.FieldDefinition{
						{Name: "validQuery", Directives: []*ast.Directive{{
							Name:      "expose",
							Arguments: []*ast.Argument{{Name: "listed", Value: &ast.Value{Raw: "true", Kind: ast.BooleanValue}}},
						}}},
						{Name: "invalidQuery", Directives: []*ast.Directive{{Name: "expose"}}},
					},
				},
				Types: map[string]*ast.Definition{
					"Query": {
						Name: "Query",
						Kind: ast.Object,
						Fields: []*ast.FieldDefinition{
							{Name: "validQuery", Directives: []*ast.Directive{{
								Name:      "expose",
								Arguments: []*ast.Argument{{Name: "listed", Value: &ast.Value{Raw: "true", Kind: ast.BooleanValue}}},
							}}},
							{Name: "invalidQuery", Directives: []*ast.Directive{{Name: "expose"}}},
						},
					},
				},
			},
			expectedErr: `@expose directive on Query.invalidQuery is missing required argument "listed" — use @expose(listed: true) or @expose(listed: false)`,
		},
		{
			name: "bare @expose on Mutation field",
			schema: &ast.Schema{
				Query:    &ast.Definition{Name: "Query"},
				Mutation: &ast.Definition{Name: "Mutation", Fields: []*ast.FieldDefinition{{Name: "doThing", Directives: []*ast.Directive{{Name: "expose"}}}}},
				Types: map[string]*ast.Definition{
					"Mutation": {Name: "Mutation", Kind: ast.Object, Fields: []*ast.FieldDefinition{{Name: "doThing", Directives: []*ast.Directive{{Name: "expose"}}}}},
				},
			},
			expectedErr: `@expose directive on Mutation.doThing is missing required argument "listed" — use @expose(listed: true) or @expose(listed: false)`,
		},
		{
			name: "bare @expose on type",
			schema: &ast.Schema{
				Query: &ast.Definition{Name: "Query"},
				Types: map[string]*ast.Definition{
					"User": {Name: "User", Kind: ast.Object, Directives: []*ast.Directive{{Name: "expose"}}},
				},
			},
			expectedErr: `@expose directive on User is missing required argument "listed" — use @expose(listed: true) or @expose(listed: false)`,
		},
		{
			name: "valid @expose(listed: true) passes",
			schema: &ast.Schema{
				Query: &ast.Definition{
					Name: "Query",
					Fields: []*ast.FieldDefinition{
						{Name: "myQuery", Directives: []*ast.Directive{{
							Name:      "expose",
							Arguments: []*ast.Argument{{Name: "listed", Value: &ast.Value{Raw: "true", Kind: ast.BooleanValue}}},
						}}},
					},
				},
				Types: map[string]*ast.Definition{
					"Query": {
						Name: "Query",
						Kind: ast.Object,
						Fields: []*ast.FieldDefinition{
							{Name: "myQuery", Directives: []*ast.Directive{{
								Name:      "expose",
								Arguments: []*ast.Argument{{Name: "listed", Value: &ast.Value{Raw: "true", Kind: ast.BooleanValue}}},
							}}},
						},
					},
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			schemaFilter := filter.NewSchemaFilterWithOptions(
				testCase.schema,
				filter.WithExposeDirective("expose"),
			)
			_, err := schemaFilter.GetFilteredSchema()
			if testCase.expectedErr != "" {
				assert.EqualError(t, err, testCase.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func createTypes() map[string]*ast.Definition {
	return map[string]*ast.Definition{
		"Query": {
			Name: "Query",
			Kind: ast.Object,
			Fields: []*ast.FieldDefinition{
				{Name: "todos", Directives: exposeDirective()},
				{Name: "internalQuery"},
			},
		},
		"Mutation": {
			Name: "Mutation",
			Kind: ast.Object,
			Fields: []*ast.FieldDefinition{
				{Name: "createTodo",
					Directives: exposeDirective(),
					Arguments:  []*ast.ArgumentDefinition{{Name: "input"}}},
				{Name: "internalMutation"},
			},
		},
		"Todo": {
			Name: "Todo",
			Kind: ast.Object,
			Fields: []*ast.FieldDefinition{
				{Name: "id"},
				{Name: "text"},
				{Name: "done"},
				{Name: "user"},
				{Name: "isGlobal", Directives: []*ast.Directive{{Name: "hide"}}},
			},
			Directives: exposeDirective(),
		},
		"User": {
			Name: "User",
			Kind: ast.Object,
			Fields: []*ast.FieldDefinition{
				{Name: "id"},
				{Name: "name"},
			},
			Directives: exposeDirective(),
		},
		"NewTodo": {
			Name: "NewTodo",
			Kind: ast.InputObject,
			Fields: []*ast.FieldDefinition{
				{Name: "text"},
				{Name: "userId"},
				{Name: "isGlobal", Directives: []*ast.Directive{{Name: "hide"}}},
			},
			Directives: exposeDirective(),
		},
	}
}
