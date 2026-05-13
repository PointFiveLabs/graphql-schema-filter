package filter_test

import (
	"github.com/vektah/gqlparser/v2/ast"
)

func exposeDirective() []*ast.Directive {
	return []*ast.Directive{{
		Name:      "expose",
		Arguments: []*ast.Argument{{Name: "introspectable", Value: &ast.Value{Raw: "true", Kind: ast.BooleanValue}}},
	}}
}
