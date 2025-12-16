package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func usersHandlerFunc(w http.ResponseWriter, r *http.Request) {
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
