package filter

import "github.com/vektah/gqlparser/v2/ast"

// IntrospectionHidePredicate is a function that inspects a field's directive list
// and returns true if the field should be hidden from introspection.
type IntrospectionHidePredicate func(ast.DirectiveList) bool

// FilterOptions holds the configuration for schema filtering
type FilterOptions struct {
	exposeDirectives            []string                    // Directives that expose fields (allowlist for Query/Mutation)
	hideDirectives              []string                    // Directives that hide fields (denylist for all types)
	internalDirectives          []string                    // Directives that expose fields but mark them as internal (hidden from introspection)
	builtInOperations           []string                    // Built-in GraphQL operations (query, mutation, etc.)
	introspectionHidePredicate  IntrospectionHidePredicate  // Custom predicate for hiding fields from introspection
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

// WithIntrospectionHidePredicate sets a custom predicate for determining whether a field
// should be hidden from introspection. The predicate receives the field's directive list
// and returns true if the field should be hidden.
//
// This is useful when hiding logic depends on directive arguments rather than just
// directive names. For example, hiding fields with @public(listed: false):
//
//	filter.WithIntrospectionHidePredicate(func(directives ast.DirectiveList) bool {
//	    d := directives.ForName("public")
//	    if d == nil { return false }
//	    arg := d.Arguments.ForName("listed")
//	    return arg != nil && arg.Value.Raw == "false"
//	})
//
// When set, this takes precedence over WithInternalDirective for introspection filtering.
func WithIntrospectionHidePredicate(predicate IntrospectionHidePredicate) Option {
	return func(o *FilterOptions) {
		o.introspectionHidePredicate = predicate
	}
}
