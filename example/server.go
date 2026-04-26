//go:generate echo "[Generate] Running gqlgen generate..."
//go:generate go run github.com/99designs/gqlgen@v0.17.66 generate --config gqlgen.yml

package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/PointFiveLabs/graphql-schema-filter/v2"
	"github.com/PointFiveLabs/graphql-schema-filter/v2/example/graph"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	c := graph.Config{Resolvers: &graph.Resolver{}}

	fullSchema := graph.NewExecutableSchema(c)
	schemaFilter := filter.NewSchemaFilterWithOptions(
		fullSchema.Schema(),
		filter.WithPublicDirective("public"),
		filter.WithHideDirective("hide"),
	)
	c.Schema = schemaFilter.MustGetFilteredSchema()
	schema := graph.NewExecutableSchema(c)
	srv := handler.NewDefaultServer(schema)
	srv.Use(schemaFilter.GetIntrospectionMiddleware())

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
