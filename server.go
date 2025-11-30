package main

import (
	"btp-transfer/graph"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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
		log.Fatal("Database opening failure:", err)
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
