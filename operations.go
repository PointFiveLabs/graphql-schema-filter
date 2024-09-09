package filter

import (
	"strings"

	"github.com/samber/lo"
	"github.com/vektah/gqlparser/v2/ast"
)

func (fs FilteredSchema) filterQueriesAndMutations(defs *ast.Definition) *ast.Definition {
	collectedFields := make([]*ast.FieldDefinition, 0, len(defs.Fields))
	for _, fd := range defs.Fields {
		if fs.mustExposeTypesByDirectives(fd.Directives) || lo.Contains(builtInTypes, strings.ToLower(fd.Name)) {
			fd.Arguments = fs.filterDefinitionArguments(fd.Arguments)
			collectedFields = append(collectedFields, fd)
		}
	}
	defs.Fields = collectedFields
	if len(defs.Fields) == 0 {
		defs.Fields = nil
	}
	return defs
}
