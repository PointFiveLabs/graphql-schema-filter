# GraphQL Schema Filter for Go

This Go package allows you to filter a GraphQL schema based on custom directives, enabling you to expose or hide specific fields, types, queries, and mutations. It is designed to work with the `gqlparser` library and provides a flexible way to modify your GraphQL schema programmatically.

## Features

- **Flexible Usage**: Integrate the schema filter into your existing GraphQL server implementation.
- **Schema Filtering**: Remove or include specific types, fields, inputs, enums, queries, and mutations based on directives.
- **Custom Directives**: Control the visibility of schema elements using custom directives.
- **Introspection Support**: The introspection query should work correctly with the filtered schema.
- **Build-time and Runtime Filtering**: Choose between modifying the schema at startup (two-server model) or filtering per-request at runtime (unified server model).

## Installation

To install the package, run:

```bash
go get github.com/PointFiveLabs/graphql-schema-filter/v2
```

## Usage

### Build-time Filtering (Two-Server Model)

This approach creates a separate filtered schema at startup. Use this when you have dedicated internal and public API servers.

```go
package main

import (
    "log"

    "github.com/vektah/gqlparser/v2/ast"
    filter "github.com/PointFiveLabs/graphql-schema-filter/v2"
)

func main() {
    // Load your GraphQL schema
    schema := &ast.Schema{...}

    // Initialize the schema filter with options
    schemaFilter := filter.NewSchemaFilterWithOptions(
        schema,
        filter.WithExposeDirective("expose"),  // Fields with @expose are included
        filter.WithHideDirective("hide"),      // Fields with @hide are removed entirely
    )

    // Get the filtered schema (validates that @expose always has "listed" argument)
    filteredSchema, err := schemaFilter.GetFilteredSchema()
    if err != nil {
        log.Fatal(err)
    }

    // Use the filtered schema in your GraphQL server
    // ...
}
```

### Runtime Filtering (Unified Server Model)

This approach keeps the full schema and applies filtering rules per-request via middleware. Use this when a single server serves both internal and external clients, and the filtering decision is made at request time (e.g., based on authentication method).

```go
schemaFilter := filter.NewSchemaFilterWithOptions(
    schema,
    filter.WithExposeDirective("expose"),
    filter.WithHideDirective("hide"),
)

executableSchema := generated.NewExecutableSchema(c)
server := handler.NewDefaultServer(executableSchema)

// Get the runtime filter middleware
runtimeFilter := schemaFilter.GetRuntimeFilterMiddleware()

// Wrap it so it only activates for certain requests (e.g., API key auth)
server.Use(myAuthAwareWrapper(runtimeFilter))
```

The runtime filter applies the same directive rules as build-time filtering but without modifying the schema. The caller controls when the middleware is active — for example, activating it only for API key requests while letting JWT-authenticated users access the full schema.

### Filtering Logic

The filtering logic works as follows:

- **@expose(listed: true)**: Fields with this directive are included in the filtered schema and visible in introspection.
- **@expose(listed: false)**: Fields with this directive are included in the filtered schema (executable) but hidden from introspection via the `GetIntrospectionMiddleware()` or `GetRuntimeFilterMiddleware()`.
- **@expose without listed**: Rejected with a validation error. The `listed` argument is always required.
- **@hide**: Fields with this directive are completely excluded from the filtered schema.
- **Built-in Operations**: Built-in GraphQL operations such as `Query`, `Mutation` are supported by default.

### Directive Behavior Examples

```graphql
type Query {
  # Visible in introspection, executable
  publicQuery: String @expose(listed: true)

  # Hidden from introspection, but executable (like an unlisted phone number)
  unlistedQuery: String @expose(listed: false)

  # Not in schema at all
  privateQuery: String
}

type User @expose(listed: true) {
  id: ID!
  name: String!
  # This field is completely removed from the schema
  internalData: String @hide
}
```

## API

### `NewSchemaFilterWithOptions`

```go
func NewSchemaFilterWithOptions(schema *ast.Schema, opts ...Option) *FilteredSchema
```

Creates a new schema filter with flexible configuration options.

**Options:**

- `WithExposeDirective(name string)`: Add a directive that marks fields as exposed (included in schema). Fields with `listed: false` argument are automatically hidden from introspection.
- `WithHideDirective(name string)`: Add a directive that marks fields as hidden (removed from schema)
- `WithBuiltInOperations(ops []string)`: Override the default built-in operations (default: ["query", "mutation"])

### `GetFilteredSchema`

```go
func (fs FilteredSchema) GetFilteredSchema() (*ast.Schema, error)
```

Returns a new filtered GraphQL schema based on the configured directives.
Returns an error if any `@expose` directive is missing the required `listed` argument.

### `MustGetFilteredSchema`

```go
func (fs FilteredSchema) MustGetFilteredSchema() *ast.Schema
```

Like `GetFilteredSchema` but panics on validation errors. Useful during server initialization.

### `GetRuntimeFilterMiddleware`

```go
func (fs FilteredSchema) GetRuntimeFilterMiddleware() *RuntimeFilterMiddleware
```

Returns a gqlgen middleware that enforces schema filtering at runtime on a per-request basis.

**Execution blocking:**

- Query/Mutation fields without `@expose` are rejected with an error
- Fields with `@hide` on any type are rejected

**Introspection filtering:**

- `__Type.fields` — non-exposed Query/Mutation fields hidden; `@expose(listed: false)` fields hidden; `@hide` fields on nested types hidden
- `__Type.inputFields` — `@hide` input object fields hidden
- `__Type.enumValues` — `@hide` enum values hidden
- `__Type.interfaces` / `__Type.possibleTypes` — non-exposed types hidden
- `__Schema.types` — non-exposed and `@hide` types hidden from the type list

**Usage with gqlgen:**

```go
schemaFilter := filter.NewSchemaFilterWithOptions(
    schema,
    filter.WithExposeDirective("public"),
    filter.WithHideDirective("hide"),
)

executableSchema := generated.NewExecutableSchema(c)
server := handler.New(executableSchema)
server.Use(schemaFilter.GetRuntimeFilterMiddleware())
```

**When to use this vs `GetFilteredSchema`:**

| | `GetFilteredSchema` | `GetRuntimeFilterMiddleware` |
|---|---|---|
| **When filtering happens** | At startup (build-time) | Per-request (runtime) |
| **Schema modification** | Creates a new, smaller schema | Full schema unchanged |
| **Use case** | Dedicated public API server | Unified server with mixed auth |
| **Filtering decision** | Fixed at server start | Dynamic, per-request |

### `GetIntrospectionMiddleware`

```go
func (fs FilteredSchema) GetIntrospectionMiddleware() *RuntimeFilterMiddleware
```

Returns a gqlgen middleware that hides `@expose(listed: false)` fields from GraphQL introspection queries. This is a companion to `GetFilteredSchema()` — use it when you need build-time schema filtering with runtime introspection hiding for unlisted fields.

> **Note:** If you are using `GetRuntimeFilterMiddleware()`, you do not need this middleware — the runtime filter already handles `@expose(listed: false)` introspection filtering.

**Usage with gqlgen:**

```go
schemaFilter := filter.NewSchemaFilterWithOptions(
    schema,
    filter.WithExposeDirective("expose"),
    filter.WithHideDirective("hide"),
)

c.Schema = schemaFilter.MustGetFilteredSchema()
executableSchema := generated.NewExecutableSchema(c)

server := handler.NewDefaultServer(executableSchema)
server.Use(schemaFilter.GetIntrospectionMiddleware())
```

**What it does:**

- Intercepts `__Type.fields` introspection queries
- For Query and Mutation types only, filters out fields with `listed: false`
- Regular types are not filtered (their `@hide` fields are already removed by the schema filter)

## Live Example

A live example demonstrating how to use this schema filter in a `gqlgen` powered application is available in the [example/](example/) folder. \
This example provides a practical implementation that you can run locally to better understand how the filtering process works.

## Limitations

- When no Queries or Mutations are exposed in the GraphQL API, even though their types exist in the GraphQL schema. In such cases, additional introspection filtering is needed to remove the mutationType or queryType from the introspection query. \
  This ensures the introspection query won't fail due to the absence of exposed Queries or Mutations, which are expected by default in the API schema. \
  To address this limitation, you can use the `gqlgen-introspect-filter` library. This library allows for introspection filtering, ensuring that queryType and mutationType are removed from the introspection query when they are not exposed. \
  Additionally, the library provides functionality to hide specific directives from the introspection query while keeping them fully functional. This ensures that sensitive directives remain invisible during introspection without affecting the underlying functionality. \
  You can explore more about this tool in the [gqlgen-introspect-filter](https://github.com/ec2-software/gqlgen-introspect-filter) repository.
- `Subscription` filtering is not supported in this package, although it can be easily added by following the same pattern as `Query` and `Mutation` filtering.

## Contributing

Feel free to open issues or submit pull requests to improve the functionality of this package.
