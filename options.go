package filter

// FilterOptions holds the configuration for schema filtering
type FilterOptions struct {
	exposeDirectives   []string // Directives that expose fields (allowlist for Query/Mutation)
	hideDirectives     []string // Directives that hide fields (denylist for all types)
	internalDirectives []string // Directives that expose fields but mark them as internal (hidden from introspection)
	builtInOperations  []string // Built-in GraphQL operations (query, mutation, etc.)
}

// Option is a function that modifies FilterOptions
type Option func(*FilterOptions)

// WithExposeDirective adds a directive name that acts as an allowlist for Query/Mutation fields.
// Fields with this directive are visible in introspection and executable.
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

// WithInternalDirective adds a directive name that marks fields as internal.
// Internal fields are included in the schema (executable) but should be hidden from introspection.
// This is equivalent to having both @expose and @hide - the field exists but is not visible.
func WithInternalDirective(name string) Option {
	return func(o *FilterOptions) {
		o.internalDirectives = append(o.internalDirectives, name)
	}
}

// WithBuiltInOperations overrides the default built-in operations list.
// Default is ["query", "mutation"].
func WithBuiltInOperations(ops []string) Option {
	return func(o *FilterOptions) {
		o.builtInOperations = ops
	}
}
