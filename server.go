package main

import (
	"btp-transfer/config"
	"btp-transfer/graph"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	_ "github.com/lib/pq"
)

const defaultPort = "8080"

func main() {
	// Open connection and check whether configuration is correct
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Printf("Configuration loaded. Port: %s", cfg.Port)

	// Use cfg.DatabaseURL to connect with database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}
	defer db.Close()

	// Wait for connection with DB
	fmt.Println("Trying to connect with database...")

	// Check connection (PING) - check if bd is still alive
	// Try 10 times every 2 seconds
	for i := 0; i < 10; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		fmt.Printf("... database is not ready (%d/10). Waiting 2s...\n", i+1)
		time.Sleep(2 * time.Second)
	}

	// If there is no connection after 20s:
	if err != nil {
		log.Fatal("Couldn't connect several times.", err)
	}

	log.Println("Successfully connected to the database!")

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to connect to database (Ping):", err)
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

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
