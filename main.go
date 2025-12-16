package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/plusk0/webserver/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load env variables in main")
	}
	var apiConf apiConfig

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to open DB")
	}
	apiConf.dbQueries = database.New(db)

	port := ":8080"

	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("."))
	mux.Handle("GET /api/healthz", http.HandlerFunc(healthHandlerFunc))
	mux.Handle("POST /api/validate_chirp", http.HandlerFunc(validateHandlerFunc))
	mux.Handle("POST /api/users", http.HandlerFunc(usersHandlerFunc))

	mux.Handle("/app/", http.StripPrefix("/app", apiConf.middlewareMetricsInc(fileServer)))

	mux.Handle("GET /admin/metrics", http.HandlerFunc(apiConf.metricsHandler))
	mux.Handle("POST /admin/reset", http.HandlerFunc(apiConf.metricsResetHandler))

	server := http.Server{Handler: mux, Addr: port}

	log.Printf("Serving files from %s on port: %s\n", "/", port)
	log.Fatal(server.ListenAndServe())
}
