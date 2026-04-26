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

### Example (Recommended - Options API)

```go
package main

import (
    "github.com/vektah/gqlparser/v2/ast"
    filter "github.com/PointFiveLabs/graphql-schema-filter"
)

func main() {
    // Load your GraphQL schema
    schema := &ast.Schema{...}

    // Initialize the schema filter with flexible options
    schemaFilter := filter.NewSchemaFilterWithOptions(
        schema,
        filter.WithExposeDirective("expose"),     // Fields visible and executable
        filter.WithInternalDirective("internal"), // Fields executable but hidden from introspection
        filter.WithHideDirective("hide"),         // Fields completely removed from schema
    )

    // Get the filtered schema
    filteredSchema := schemaFilter.GetFilteredSchema()

    // Use the filtered schema in your GraphQL server
    // ...
}
```

### Example (Legacy API - Deprecated)

```go
package main

import (
    "github.com/vektah/gqlparser/v2/ast"
    "github.com/PointFiveLabs/graphql-schema-filter"
)

func main() {
    // Load your GraphQL schema
    schema := &ast.Schema{...}

    // Initialize the schema filter (legacy API)
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

- **@expose**: Fields with this directive are included in the filtered schema and visible in introspection. Primarily used for Query/Mutation fields.
- **@internal**: Fields with this directive are included in the filtered schema (executable) but should be hidden from introspection by middleware. This is equivalent to having both @expose and @hide - the field exists but is not publicly visible.
- **@hide**: Fields with this directive are completely excluded from the filtered schema.
- **Built-in Operations**: Built-in GraphQL operations such as `Query`, `Mutation` are supported by default.

### Customization

You can customize the behavior of the schema filtering using options:

```go
schemaFilter := filter.NewSchemaFilterWithOptions(
    schema,
    filter.WithExposeDirective("expose"),
    filter.WithExposeDirective("public"), // Multiple expose directives
    filter.WithInternalDirective("internal"),
    filter.WithHideDirective("hide"),
    filter.WithBuiltInOperations([]string{"query"}), // Custom operations
)
```

### Directive Behavior Examples

````graphql
type Query {
  # Visible in introspection, executable
  publicQuery: String @expose

  # Hidden from introspection, but executable (for internal tools)
  internalQuery: String @internal

  # Not in schema at all
  privateQuery: String
}

type User @expose {
  id: ID!
  name: String!
  # This field is completely removed from the schema
  internalData: String @hide
}

## API

### `NewSchemaFilterWithOptions` (Recommended)

```go
func NewSchemaFilterWithOptions(schema *ast.Schema, opts ...Option) *FilteredSchema
````

Creates a new schema filter with flexible configuration options.

**Options:**

- `WithExposeDirective(name string)`: Add a directive that marks fields as exposed (visible and executable)
- `WithInternalDirective(name string)`: Add a directive that marks fields as internal (executable but hidden from introspection)
- `WithHideDirective(name string)`: Add a directive that marks fields as hidden (removed from schema)
- `WithIntrospectionHidePredicate(predicate)`: Set a custom predicate for hiding fields from introspection based on directive arguments (e.g., `@public(listed: false)`)
- `WithBuiltInOperations(ops []string)`: Override the default built-in operations (default: ["query", "mutation"])

### `NewSchemaFilter` (Deprecated)

```go
func NewSchemaFilter(schema *ast.Schema, exposeDirective, hideDirective string, overrideBuiltInOperations *[]string) *FilteredSchema
```

Legacy API for backward compatibility. Use `NewSchemaFilterWithOptions` for new code.

- `schema`: The full GraphQL schema to be filtered.
- `exposeDirective`: The directive used to expose elements in the schema.
- `hideDirective`: The directive used to hide elements in the schema.
- `overrideBuiltInOperations`: An optional list of built-in operations to override the default behavior.

### `GetFilteredSchema`

```go
func (fs FilteredSchema) GetFilteredSchema() *ast.Schema
```

Returns a new filtered GraphQL schema based on the configured directives.

### `GetIntrospectionMiddleware`

```go
func (fs *FilteredSchema) GetIntrospectionMiddleware() *IntrospectionFilterMiddleware
```

Returns a gqlgen middleware that hides fields marked with `@internal` directives from GraphQL introspection queries.

**Why is this needed?**

The schema filter operates at **build-time** by modifying the AST. It can either include a field in the schema (making it executable) or remove it entirely. However, `@internal` fields need to be:

- ✅ **Included in the schema** (so they can be executed)
- ❌ **Hidden from introspection** (so they don't appear in schema queries)

This requires **runtime** filtering of introspection responses, which is what this middleware provides.

**Usage with gqlgen:**

```go
// Setup schema filter
schemaFilter := filter.NewSchemaFilterWithOptions(
    schema,
    filter.WithExposeDirective("expose"),
    filter.WithInternalDirective("internal"),
    filter.WithHideDirective("hide"),
)

// Apply filtered schema
c.Schema = schemaFilter.GetFilteredSchema()
executableSchema := generated.NewExecutableSchema(c)

// Register the introspection middleware (if using @internal)
server := handler.NewDefaultServer(executableSchema)
server.Use(schemaFilter.GetIntrospectionMiddleware())
```

**What it does:**

- Intercepts `__Type.fields` introspection queries
- For Query and Mutation types only, filters out fields with `@internal` directives
- Regular types are not filtered (their `@hide` fields are already removed by the schema filter)

## Live Example

A live example demonstrating how to use this schema filter in a `gqlgen` powered application is available in the [example/](example/) folder. \
This example provides a practical implementation that you can run locally to better understand how the filtering process works.

## Limitations

- When no Queries or Mutations are exposed in the GraphQL API, even though their types exist in the GraphQL schema. In such cases, additional introspection filtering is needed to remove the mutationType or queryType from the introspection query. \
  This ensures the introspection query won’t fail due to the absence of exposed Queries or Mutations, which are expected by default in the API schema. \
  To address this limitation, you can use the `gqlgen-introspect-filter` library. This library allows for introspection filtering, ensuring that queryType and mutationType are removed from the introspection query when they are not exposed. \
  Additionally, the library provides functionality to hide specific directives from the introspection query while keeping them fully functional. This ensures that sensitive directives remain invisible during introspection without affecting the underlying functionality. \
  You can explore more about this tool in the [gqlgen-introspect-filter](https://github.com/ec2-software/gqlgen-introspect-filter) repository.
- `Subscription` filtering is not supported in this package, although it can be easily added by following the same pattern as `Query` and `Mutation` filtering.

## Contributing

Feel free to open issues or submit pull requests to improve the functionality of this package.
