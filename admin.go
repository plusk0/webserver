package main

import (
	"fmt"
	"log"
	"net/http"
)

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte(fmt.Sprintf(
		`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %v times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())))
	if err != nil {
		log.Fatalf("Failed to handle Metrics request: %v", err)
	}
	w.Header().Add("Content-Type", "text/html")

	w.WriteHeader(200)
}

func (cfg *apiConfig) metricsResetHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, 403, "Not allowed outside of dev environment")
		return
	}
	cfg.fileserverHits.Store(0)
	_, err := cfg.dbQueries.ResetUsers(r.Context())
	if err != nil {
		log.Fatal("Failed to reset users")
	}
	fmt.Println("resetting")
	w.WriteHeader(200)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
