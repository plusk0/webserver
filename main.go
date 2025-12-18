package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/plusk0/webserver/internal/database"
)

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
	apiConf.platform = os.Getenv("PLATFORM")
	apiConf.JWTKey = os.Getenv("JWT")
	apiConf.PolkaKey = os.Getenv("POLKA_KEY")

	port := ":8080"

	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("."))
	mux.Handle("GET /api/healthz", http.HandlerFunc(healthHandlerFunc))
	mux.Handle("POST /api/chirps", http.HandlerFunc(apiConf.validateHandlerFunc))
	mux.Handle("GET /api/chirps", http.HandlerFunc(apiConf.getChirpsHandlerFunc))
	mux.Handle("GET /api/chirps/{chirpID}", http.HandlerFunc(apiConf.getChirpHandlerFunc))
	mux.Handle("DELETE /api/chirps/{chirpID}", http.HandlerFunc(apiConf.deleteChirpHandlerFunc))

	mux.Handle("POST /api/users", http.HandlerFunc(apiConf.usersHandlerFunc))
	mux.Handle("PUT /api/users", http.HandlerFunc(apiConf.userUpdateHandlerFunc))
	mux.Handle("POST /api/login", http.HandlerFunc(apiConf.loginHandlerFunc))
	mux.Handle("POST /api/refresh", http.HandlerFunc(apiConf.refreshHandlerFunc))
	mux.Handle("POST /api/revoke", http.HandlerFunc(apiConf.revokeHandlerFunc))

	mux.Handle("POST /api/polka/webhooks", http.HandlerFunc(apiConf.webhookHandlerFunc))

	mux.Handle("/app/", http.StripPrefix("/app", apiConf.middlewareMetricsInc(fileServer)))

	mux.Handle("GET /admin/metrics", http.HandlerFunc(apiConf.metricsHandler))
	mux.Handle("POST /admin/reset", http.HandlerFunc(apiConf.metricsResetHandler))

	server := http.Server{Handler: mux, Addr: port}

	log.Printf("Serving files from %s on port: %s\n", "/", port)
	log.Fatal(server.ListenAndServe())
}
