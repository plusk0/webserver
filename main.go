package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	port := ":8080"
	var apiConf apiConfig

	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("."))
	mux.Handle("GET /api/healthz", http.HandlerFunc(healthHandlerFunc))
	mux.Handle("POST /api/validate_chirp", http.HandlerFunc(validateHandlerFunc))

	mux.Handle("/app/", http.StripPrefix("/app", apiConf.middlewareMetricsInc(fileServer)))

	mux.Handle("GET /admin/metrics", http.HandlerFunc(apiConf.metricsHandler))
	mux.Handle("POST /admin/reset", http.HandlerFunc(apiConf.metricsResetHandler))

	server := http.Server{Handler: mux, Addr: port}

	log.Printf("Serving files from %s on port: %s\n", "/", port)
	log.Fatal(server.ListenAndServe())
}

func healthHandlerFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	_, err := w.Write([]byte("OK"))
	if err != nil {
		log.Fatal(err)
	}
	w.WriteHeader(200)
}

func validateHandlerFunc(w http.ResponseWriter, r *http.Request) {
	type jsonString struct {
		Body string `json:"body"`
	}

	var req jsonString

	data, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		return
	}
	defer r.Body.Close()
	err = json.Unmarshal(data, &req)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		return
	}
	if len(req.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	dirty := strings.Split(req.Body, " ")
	dirtyWords := []string{"kerfuffle", "sharbert", "fornax"}
	var cleanWords []string
	for _, v := range dirty {
		for _, badWord := range dirtyWords {
			if strings.ToLower(v) == badWord {
				v = "****"
			}
		}
		cleanWords = append(cleanWords, v)
	}
	payload := strings.Join(cleanWords, " ")
	respondWithJSON(w, 200, payload)
}

func respondWithError(w http.ResponseWriter, code int, text string) {
	w.WriteHeader(code)
	resData, err := json.Marshal(map[string]string{"error": text})
	if err != nil {
		fmt.Printf("Failed at respondWithError: %v", err)
		_, _ = w.Write([]byte(resData))
	}
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	w.WriteHeader(code)
	resData, err := json.Marshal(map[string]any{"cleaned_body": payload})
	if err != nil {
		fmt.Printf("Failed at responWithJSON: %v", err)
	}
	_, _ = w.Write([]byte(resData))
}

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
	cfg.fileserverHits.Store(0)
	fmt.Println("resetting")
	w.WriteHeader(200)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
