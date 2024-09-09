package filter_test

import (
	"testing"

	filter "github.com/PointFiveLabs/graphql-schema-filter"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestSchemaFiltering(t *testing.T) {
	fullSchema := getTestSchema()

	schemaFilter := filter.NewSchemaFilter(fullSchema, "expose", "hide", nil)
	filteredSchema := schemaFilter.GetFilteredSchema()

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkTypeAndFields(t, filteredSchema, tt.typeName, tt.shouldTypeExist, tt.fields)
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
			{Name: "todos", Directives: []*ast.Directive{{Name: "expose"}}},
			{Name: "internalQuery"},
		},
	}
}

func createMutation() *ast.Definition {
	return &ast.Definition{
		Name: "Mutation",
		Fields: []*ast.FieldDefinition{
			{Name: "createTodo", Directives: []*ast.Directive{{Name: "expose"}}, Arguments: []*ast.ArgumentDefinition{
				{Name: "input", Type: ast.NonNullNamedType("NewTodo", nil)},
			}},
			{Name: "internalMutation"},
		},
	}
}

func createTypes() map[string]*ast.Definition {
	return map[string]*ast.Definition{
		"Query": {
			Name: "Query",
			Kind: ast.Object,
			Fields: []*ast.FieldDefinition{
				{Name: "todos", Directives: []*ast.Directive{{Name: "expose"}}},
				{Name: "internalQuery"},
			},
		},
		"Mutation": {
			Name: "Mutation",
			Kind: ast.Object,
			Fields: []*ast.FieldDefinition{
				{Name: "createTodo",
					Directives: []*ast.Directive{{Name: "expose"}},
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
			Directives: []*ast.Directive{{Name: "expose"}},
		},
		"User": {
			Name: "User",
			Kind: ast.Object,
			Fields: []*ast.FieldDefinition{
				{Name: "id"},
				{Name: "name"},
			},
			Directives: []*ast.Directive{{Name: "expose"}},
		},
		"NewTodo": {
			Name: "NewTodo",
			Kind: ast.InputObject,
			Fields: []*ast.FieldDefinition{
				{Name: "text"},
				{Name: "userId"},
				{Name: "isGlobal", Directives: []*ast.Directive{{Name: "hide"}}},
			},
			Directives: []*ast.Directive{{Name: "expose"}},
		},
	}
}
