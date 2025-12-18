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
	"github.com/plusk0/webserver/internal/auth"
	"github.com/plusk0/webserver/internal/database"
)

func (conf *apiConfig) usersHandlerFunc(w http.ResponseWriter, r *http.Request) {
	usr, err := getUsrReq(r)
	if err != nil {
		respondWithError(w, 404, "Failed")
	}

	hash, err := auth.HashPassword(usr.Password)
	if err != nil {
		respondWithError(w, 404, "Failed")
	}

	params := database.CreateUserParams{usr.Email, hash}
	dbUsr, err := conf.dbQueries.CreateUser(r.Context(), params)
	if err != nil {
		log.Fatal("Failed to create User")
	}

	tk, err := auth.MakeJWT(dbUsr.ID, conf.JWTKey)
	if err != nil {
		log.Fatal("Failed to make JWT")
	}
	rTK, err := auth.MakeRefreshToken()
	if err != nil {
		log.Fatal("Failed to make rTK")
	}
	user := dbUserToSafeJSON(dbUsr, tk, rTK)
	respondWithJSON(w, 201, user)
}

func (conf *apiConfig) loginHandlerFunc(w http.ResponseWriter, r *http.Request) {
	usr, err := getUsrReq(r)
	if err != nil {
		respondWithError(w, 401, "invalid Username or Password")
		return
	}

	user, err := conf.dbQueries.GetUser(r.Context(), usr.Email)
	if err != nil {
		respondWithError(w, 401, "invalid Username or Password")
		return
	}

	valid, err := auth.CheckPasswordHash(usr.Password, user.Password)
	if !valid {
		respondWithError(w, 401, "invalid Username or Password")
	}
	if err != nil {
		log.Fatalf("Failed to check Login credentials: %v", err)
	}

	tk, err := auth.MakeJWT(user.ID, conf.JWTKey)
	if err != nil {
		log.Fatal("Failed to make JWT")
	}
	rTK, err := auth.MakeRefreshToken()
	if err != nil {
		log.Fatal("Failed to make rTK")
	}
	params := database.CreateTokenParams{rTK, user.ID, time.Now().Add(time.Hour)}
	conf.dbQueries.CreateToken(r.Context(), params)
	respondWithJSON(w, 200, dbUserToSafeJSON(user, tk, rTK))
}

func (conf *apiConfig) refreshHandlerFunc(w http.ResponseWriter, r *http.Request) {
	tk, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Fatal("Failed to get header Token")
	}
	dbToken, err := conf.dbQueries.GetToken(r.Context(), tk)
	if err != nil || dbToken.RevokedAt.Valid || dbToken.Token == "" {
		respondWithError(w, 401, "Invalid token")
	}
	tkNew, err := auth.MakeJWT(dbToken.UserID, conf.JWTKey)
	if err != nil {
		log.Fatal("Failed to make JWT")
	}
	tokenMap := map[string]string{
		"token": tkNew,
	}
	respondWithJSON(w, 200, tokenMap)
}

func dbUserToSafeJSON(db database.User, tk string, rTK string) User {
	return User{db.ID, db.CreatedAt, db.UpdatedAt, db.Email, "", tk, rTK}
}

func getUsrReq(r *http.Request) (usrReq, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return usrReq{}, fmt.Errorf("failed to read body: %v", err)
	}
	defer r.Body.Close()

	var usr usrReq
	err = json.Unmarshal(data, &usr)
	if err != nil {
		return usrReq{}, fmt.Errorf("failed to parse email: %v", err)
	}
	return usr, nil
}

func healthHandlerFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	_, err := w.Write([]byte("OK"))
	if err != nil {
		log.Fatal(err)
	}
	w.WriteHeader(200)
}

func (conf *apiConfig) validateHandlerFunc(w http.ResponseWriter, r *http.Request) {
	var req chirpReq
	tk, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Fatal("Failed to get header Token")
	}
	validUser, err := auth.ValidateJWT(tk, conf.JWTKey)
	if err != nil {
		fmt.Println(err)
	}
	req.UserID = validUser

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
		respondWithError(w, 400, "Unauthorized")
		return
	}
	respondWithJSON(w, 201, dbChirpToJSON(insertedChirp))
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

func (conf *apiConfig) getChirpHandlerFunc(w http.ResponseWriter, r *http.Request) {
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 404, "Failed to parse ChirpID")
	}
	chirp, err := conf.dbQueries.GetChirp(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, 404, "ChirpNotFound")
	}
	respondWithJSON(w, 200, dbChirpToJSON(chirp))
}

func dbChirpToJSON(db database.Chirp) Chirp {
	return Chirp{db.ID, db.CreatedAt, db.UpdatedAt, db.Body, db.UserID}
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
