package main

import (
	"log"
	"net/http"
	"os"

	"github.com/csg33k/w2c-generator/internal/adapters/efw2c"
	sqliteadapter "github.com/csg33k/w2c-generator/internal/adapters/sqlite"
	"github.com/csg33k/w2c-generator/internal/handlers"
)

func main() {
	dsn := os.Getenv("DB_PATH")
	if dsn == "" {
		dsn = "w2c.db"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	repo, err := sqliteadapter.New(dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	gen := efw2c.New()
	h := handlers.New(repo, gen)

	log.Printf("W-2c EFW2C Generator running on http://localhost:%s", port)
	log.Printf("Database: %s", dsn)
	if err := http.ListenAndServe(":"+port, h.Routes()); err != nil {
		log.Fatal(err)
	}
}
