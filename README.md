# GraphQL Schema Filter for Go

This Go package allows you to filter a GraphQL schema based on custom directives, enabling you to expose or hide specific fields, types, queries, and mutations. It is designed to work with the `gqlparser` library and provides a flexible way to modify your GraphQL schema programmatically.

## Features

- **Flexible Usage**: Integrate the schema filter into your existing GraphQL server implementation.
- **Schema Filtering**: Remove or include specific types, fields, inputs, enums, queries, and mutations based on directives.
- **Custom Directives**: Control the visibility of schema elements using custom directives.
- **Introspection Support**: The introspection query should work correctly with the filtered schema.

## Installation

To install the package, run:

```bash
go get github.com/PointFiveLabs/graphql-schema-filter
```

## Usage
This method should exist alongside a full schema server that is intended for internal usages. This ensures that the internal server has access to the complete schema, while the filtered schema is exposed to external clients.
### Example

```go
package main

import (
    "github.com/vektah/gqlparser/v2/ast"
    "github.com/PointFiveLabs/graphql-schema-filter"
)

func main() {
    // Load your GraphQL schema
    schema := &ast.Schema{...}
	
    // Initialize the schema filter
    schemaFilter := filter.NewSchemaFilter(
        schema,
        "expose", // Directive to expose elements
        "hide",   // Directive to hide elements
        nil,      // Optionally override built-in operations
    )
    
    // Get the filtered schema
    filteredSchema := schemaFilter.GetFilteredSchema()
    
    // Use the filtered schema in your GraphQL server
    // ...
}
```

### Filtering Logic

The filtering logic works as follows:

- **Expose Directive**: Elements with the `expose` directive are included in the filtered schema.
- **Hide Directive**: Elements with the `hide` directive are excluded from the filtered schema.
- **Built-in Operations**: Built-in GraphQL operations such as `Query`, `Mutation`, `Subscription`.

### Customization

You can customize the behavior of the schema filtering by providing an override for the built-in operations:

```go
customBuiltInOperations := []string{"Query"}
schemaFilter := filter.NewSchemaFilter(
    schema,
    "expose",
    "hide",
    &customBuiltInOperations,
)
```

## API

### `NewSchemaFilter`

```go
func NewSchemaFilter(schema *ast.Schema, exposeDirective, hideDirective string, overrideBuiltInOperations *[]string) *FilteredSchema
```

- `schema`: The full GraphQL schema to be filtered.
- `exposeDirective`: The directive used to expose elements in the schema.
- `hideDirective`: The directive used to hide elements in the schema.
- `overrideBuiltInOperations`: An optional list of built-in operations to override the default behavior.

### `GetFilteredSchema`

```go
func (fs FilteredSchema) GetFilteredSchema() *ast.Schema
```

Returns a new filtered GraphQL schema, with only the exposed elements included.

## Live Example

A live example demonstrating how to use this schema filter in a `gqlgen` powered application is available in the [example/](example/) folder. \
This example provides a practical implementation that you can run locally to better understand how the filtering process works. 

## Limitations
* When no Queries or Mutations are exposed in the GraphQL API, even though their types exist in the GraphQL schema. In such cases, additional introspection filtering is needed to remove the mutationType or queryType from the introspection query. \
This ensures the introspection query won’t fail due to the absence of exposed Queries or Mutations, which are expected by default in the API schema. \
To address this limitation, you can use the `gqlgen-introspect-filter` library. This library allows for introspection filtering, ensuring that queryType and mutationType are removed from the introspection query when they are not exposed. \
Additionally, the library provides functionality to hide specific directives from the introspection query while keeping them fully functional. This ensures that sensitive directives remain invisible during introspection without affecting the underlying functionality. \
You can explore more about this tool in the [gqlgen-introspect-filter](https://github.com/ec2-software/gqlgen-introspect-filter) repository.
* `Subscription` filtering is not supported in this package, although it can be easily added by following the same pattern as `Query` and `Mutation` filtering.

## Contributing

Feel free to open issues or submit pull requests to improve the functionality of this package.
