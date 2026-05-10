package filter

import (
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/vektah/gqlparser/v2/ast"
)

func hasAnyDirective(directives ast.DirectiveList, directiveNames []string) bool {
	for _, name := range directiveNames {
		if name != "" && directives.ForName(name) != nil {
			return true
		}
	}
	return false
}

// IsUnlisted returns true if the directives include an expose directive with listed: false.
func IsUnlisted(directives ast.DirectiveList, exposeDirectives []string) bool {
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

func getParentTypeName(fc *graphql.FieldContext) *string {
	if fc == nil || fc.Parent == nil || fc.Parent.Result == nil {
		return nil
	}
	typeResult, ok := fc.Parent.Result.(*introspection.Type)
	if !ok || typeResult == nil {
		return nil
	}
	return typeResult.Name()
}

func filterFieldList(fields []introspection.Field, astType *ast.Definition, shouldInclude func(*ast.FieldDefinition) bool) []introspection.Field {
	filtered := make([]introspection.Field, 0, len(fields))
	for _, field := range fields {
		astField := astType.Fields.ForName(field.Name)
		if astField == nil {
			continue
		}
		if shouldInclude(astField) {
			filtered = append(filtered, field)
		}
	}
	return filtered
}
