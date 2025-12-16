package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/plusk0/webserver/internal/database"
)

func (conf *apiConfig) usersHandlerFunc(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatal("Failed to read body")
	}
	defer r.Body.Close()
	var usr User
	err = json.Unmarshal(data, &usr)
	if err != nil {
		log.Fatalf("Failed to parse email: %v", err)
	}

	dbUsr, err := conf.dbQueries.CreateUser(r.Context(), string(usr.Email))
	if err != nil {
		log.Fatal("Failed to create User")
	}
	usr.CreatedAt = dbUsr.CreatedAt
	usr.UpdatedAt = dbUsr.UpdatedAt
	usr.ID = dbUsr.ID
	usr.Email = dbUsr.Email

	respondWithJSON(w, 201, usr)
}

func healthHandlerFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	_, err := w.Write([]byte("OK"))
	if err != nil {
		log.Fatal(err)
	}
	w.WriteHeader(200)
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type chirpReq struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

func (conf *apiConfig) validateHandlerFunc(w http.ResponseWriter, r *http.Request) {
	var req chirpReq

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
	args := database.CreateChirpParams{payload, req.UserID}
	insertedChirp, err := conf.dbQueries.CreateChirp(r.Context(), args)
	if err != nil {
		respondWithError(w, 400, "Failed to insert Chirp")
		return
	}
	var jsonChirp Chirp
	jsonChirp.ID = insertedChirp.ID
	jsonChirp.CreatedAt = insertedChirp.CreatedAt
	jsonChirp.UpdatedAt = insertedChirp.UpdatedAt
	jsonChirp.Body = insertedChirp.Body
	jsonChirp.UserID = insertedChirp.UserID
	fmt.Println(insertedChirp)
	respondWithJSON(w, 201, jsonChirp)
}

func (conf *apiConfig) getChirpsHandlerFunc(w http.ResponseWriter, r *http.Request) {
	chirps, err := conf.dbQueries.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, 400, "Failed to get Chirps")
	}
	var jsonChirps []Chirp
	for _, v := range chirps {
		jsonChirp := dbChirpToJSON(v)
		jsonChirps = append(jsonChirps, jsonChirp)
	}
	respondWithJSON(w, 200, jsonChirps)
}

func dbChirpToJSON(db database.Chirp) Chirp {
	resp := Chirp{db.ID, db.CreatedAt, db.UpdatedAt, db.Body, db.UserID}
	return resp
}

func writeJSONResponse(w http.ResponseWriter, code int, payload any) error {
	w.WriteHeader(code)
	resData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}
	_, err = w.Write(resData)
	return err
}

func respondWithError(w http.ResponseWriter, code int, text string) {
	if err := writeJSONResponse(w, code, map[string]string{"error": text}); err != nil {
		fmt.Printf("Failed at respondWithError: %v", err)
	}
}

func respondWithNamedJSON(w http.ResponseWriter, code int, name string, payload any) {
	if err := writeJSONResponse(w, code, map[string]any{name: payload}); err != nil {
		fmt.Printf("Failed at respondWithNamedJSON: %v", err)
	}
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	if err := writeJSONResponse(w, code, payload); err != nil {
		fmt.Printf("Failed at respondWithJSON: %v", err)
	}
}
