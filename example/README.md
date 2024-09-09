# GraphQL Project Example

This example project demonstrates a simple GraphQL server implementation using `gqlgen` and `graphql-schema-filter`. The project includes a basic schema for managing `Todo` items and `User` entities.

## Overview

The project defines a GraphQL schema with the following types:

- **Todo**: Represents a task with fields for `id`, `text`, `done`, `user`, and `isGlobal`.
- **User**: Represents a user with fields for `id` and `name`.
- **NewTodo**: Input type for creating a new `Todo` item with fields for `text`, `userId`, and `isGlobal`.

## Hidden Operations

- **InternalQuery**: Example query for internal use.
- **InternalMutation**: Example mutation for internal use.

## Hidden Field: `isGlobal`

The `isGlobal` field in the `Todo` type indicates whether a `Todo` item is visible to all users or only to the specific user who created it. If `isGlobal` is set to `true`, the `Todo` item is visible to everyone; otherwise, it is only visible to the user who created it.

## Schema Filter Library

The project uses the `graphql-schema-filter` library to manage the visibility of schema elements. This library allows you to use custom directives to expose or hide parts of your schema. In this project, the following directives are used:

- **@expose**: Marks a field or type to be included in the filtered schema.
- **@hide**: Marks a field or type to be excluded from the filtered schema.

The `NewSchemaFilter` function is used to create a filtered schema based on these directives, ensuring that only the intended parts of the schema are exposed to the clients.

## Running the Server

To run the server, use the following command:

```sh
go run example/server.go
