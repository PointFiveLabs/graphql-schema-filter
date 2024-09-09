package filter

import (
	"strings"

	"github.com/samber/lo"
	"github.com/vektah/gqlparser/v2/ast"
)

func (fs FilteredSchema) filterTypes(types map[string]*ast.Definition) map[string]*ast.Definition {
	collectedTypes := make(map[string]*ast.Definition)
	for t, def := range types {
		// Skips the type if it's neither built-in, exposed via directives, nor a scalar
		if !lo.Contains(fs.supportedBuiltInAttributes, strings.ToLower(def.Name)) && !fs.mustExposeTypesByDirectives(def.Directives) && def.Kind != ast.Scalar {
			continue
		}
		switch def.Kind {
		case ast.Object, ast.Interface, ast.Union, ast.InputObject, ast.Scalar:
			// Filter fields of operations (Query and Mutation) differently from other types
			def.Fields = lo.Filter(def.Fields, func(fd *ast.FieldDefinition, _ int) bool {
				if lo.Contains(builtInOperations, strings.ToLower(def.Name)) {
					fd.Arguments = fs.filterDefinitionArguments(fd.Arguments)
					return lo.Contains(fs.supportedBuiltInAttributes, strings.ToLower(fd.Name)) || fs.mustExposeTypesByDirectives(fd.Directives)
				}
				return fs.shouldExposeFieldsByDirectives(fd.Directives)
			})
		case ast.Enum:
			def.EnumValues = lo.Filter(def.EnumValues, func(ev *ast.EnumValueDefinition, _ int) bool {
				return fs.shouldExposeFieldsByDirectives(ev.Directives)
			})
		}
		collectedTypes[t] = def
	}
	return collectedTypes
}

func (fs FilteredSchema) filterImplementsAndPossibleTypes(possibleTypes map[string][]*ast.Definition) map[string][]*ast.Definition {
	collectedTypes := make(map[string]*ast.Definition)
	for _, types := range possibleTypes {
		for _, t := range types {
			collectedTypes[t.Name] = t
		}
	}
	filteredTypes := fs.filterTypes(collectedTypes)
	for i, types := range possibleTypes {
		newTypes := make([]*ast.Definition, 0)
		for _, t := range types {
			filteredType, ok := filteredTypes[t.Name]
			if ok {
				newTypes = append(newTypes, filteredType)
			}
		}
		if len(newTypes) == 0 {
			delete(possibleTypes, i)
		} else {
			possibleTypes[i] = newTypes
		}
	}
	return possibleTypes
}
