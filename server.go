package main

import (
	"btp-transfer/graph"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	_ "github.com/lib/pq"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Connection configuration
	connStr := "user=user password=password dbname=btp_tokens sslmode=disable host=localhost"

	// If docker:
	if os.Getenv("DB_HOST") != "" {
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	}

	// Open connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Błąd otwierania bazy:", err)
	}
	defer db.Close()

	// Check connection (PING) - check if bd is still alive
	if err := db.Ping(); err != nil {
		log.Fatal("Nie udało się połączyć z bazą (Ping):", err)
	}
	log.Println("Successfully connected to the database!")

	// Send bd to Transfer function
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			DB: db,
		},
	}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
