# Changelog

## v3.0.0

### Breaking changes

- Renamed the required argument on expose directives from `listed` to `introspectable`. The argument now names the mechanism it controls — visibility in schema introspection — rather than the phone-book metaphor.
- Renamed the helper `IsUnlisted` to `IsHiddenFromIntrospection`. Polarity is unchanged: it still returns `true` when the directive carries `introspectable: false`.

### Migration

In every schema, replace the argument name:

```graphql
# before
type Foo @expose(listed: true)
type Bar @expose(listed: false)

# after
type Foo @expose(introspectable: true)
type Bar @expose(introspectable: false)
```

In Go callers, update the helper name (if used directly):

```go
// before
if filter.IsUnlisted(directives, exposeDirectives) { ... }

// after
if filter.IsHiddenFromIntrospection(directives, exposeDirectives) { ... }
```

Update your import path:

```go
filter "github.com/PointFiveLabs/graphql-schema-filter/v3"
```

The validation error message changes from `missing required argument "listed"` to `missing required argument "introspectable"` — update any test fixtures that match on the message.
