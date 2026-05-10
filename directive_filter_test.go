package filter_test

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	filter "github.com/PointFiveLabs/graphql-schema-filter/v2"
)

func TestDirectiveFilterMiddleware(t *testing.T) {
	schema := &ast.Schema{
		Directives: map[string]*ast.DirectiveDefinition{
			"expose":     {Name: "expose"},
			"hide":       {Name: "hide"},
			"role":       {Name: "role"},
			"deprecated": {Name: "deprecated"},
			"skip":       {Name: "skip"},
		},
	}

	middleware := filter.NewSchemaFilterWithOptions(schema).GetDirectiveFilterMiddleware(func(name string) bool {
		return name == "deprecated" || name == "skip"
	})

	tests := []struct {
		name           string
		object         string
		fieldName      string
		resolverResult any
		expectedNames  []string
		shouldFilter   bool
	}{
		{
			name:      "filters __schema directives to only allowed ones",
			object:    "__Schema",
			fieldName: "directives",
			resolverResult: []introspection.Directive{
				{Name: "expose"},
				{Name: "hide"},
				{Name: "role"},
				{Name: "deprecated"},
				{Name: "skip"},
			},
			expectedNames: []string{"deprecated", "skip"},
			shouldFilter:  true,
		},
		{
			name:           "passes through non-schema fields",
			object:         "__Type",
			fieldName:      "fields",
			resolverResult: "unchanged",
			shouldFilter:   false,
		},
		{
			name:           "passes through non-directives schema fields",
			object:         "__Schema",
			fieldName:      "types",
			resolverResult: "unchanged",
			shouldFilter:   false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := graphql.WithFieldContext(context.Background(), &graphql.FieldContext{
				Object: testCase.object,
				Field: graphql.CollectedField{
					Field: &ast.Field{Name: testCase.fieldName},
				},
			})

			next := func(ctx context.Context) (any, error) {
				return testCase.resolverResult, nil
			}

			result, err := middleware.InterceptField(ctx, next)
			require.NoError(t, err)

			if testCase.shouldFilter {
				directives, ok := result.([]introspection.Directive)
				require.True(t, ok)
				names := make([]string, len(directives))
				for i, d := range directives {
					names[i] = d.Name
				}
				assert.Equal(t, testCase.expectedNames, names)
			} else {
				assert.Equal(t, testCase.resolverResult, result)
			}
		})
	}
}
