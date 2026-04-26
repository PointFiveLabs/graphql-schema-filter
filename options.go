package filter

// FilterOptions holds the configuration for schema filtering
type FilterOptions struct {
	exposeDirectives  []string // Directives that include fields (allowlist for Query/Mutation)
	hideDirectives    []string // Directives that hide fields (denylist for all types)
	builtInOperations []string // Built-in GraphQL operations (query, mutation, etc.)
}

// Option is a function that modifies FilterOptions
type Option func(*FilterOptions)

// WithExposeDirective adds a directive name that acts as an allowlist for Query/Mutation fields.
// Fields with this directive are included in the filtered schema and executable.
// Fields with @<name>(listed: false) are included but hidden from introspection.
// The "listed" argument is required — GetFilteredSchema will return an error if any
// field uses @<name> without specifying listed: true or listed: false.
func WithExposeDirective(name string) Option {
	return func(o *FilterOptions) {
		o.exposeDirectives = append(o.exposeDirectives, name)
	}
}

// WithHideDirective adds a directive name that acts as a denylist for type fields.
// Fields with this directive are hidden from the schema entirely.
func WithHideDirective(name string) Option {
	return func(o *FilterOptions) {
		o.hideDirectives = append(o.hideDirectives, name)
	}
}

// WithBuiltInOperations overrides the default built-in operations list.
// Default is ["query", "mutation"].
func WithBuiltInOperations(ops []string) Option {
	return func(o *FilterOptions) {
		o.builtInOperations = ops
	}
}
