package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
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

func (conf *apiConfig) userUpdateHandlerFunc(w http.ResponseWriter, r *http.Request) {
	tk, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Invalid Header")
	}
	validUser, err := auth.ValidateJWT(tk, conf.JWTKey)
	if err != nil {
		fmt.Println(err)
	}

	usr, err := getUsrReq(r)
	if err != nil {
		respondWithError(w, 401, "Failed")
		return
	}
	hash, err := auth.HashPassword(usr.Password)
	if err != nil {
		respondWithError(w, 401, "Failed")
		return
	}
	params := database.UpdateUserParams{validUser, usr.Email, hash}
	user, err := conf.dbQueries.UpdateUser(r.Context(), params)
	if err != nil {
		respondWithError(w, 401, "Failed")
		return
	}
	respondWithJSON(w, 200, dbUserToUserJSON(user))
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

func (conf *apiConfig) revokeHandlerFunc(w http.ResponseWriter, r *http.Request) {
	tk, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Fatal("Failed to get header Token")
	}
	_, err = conf.dbQueries.RevokeToken(r.Context(), tk)
	if err != nil {
		respondWithError(w, 400, "Failed to revoke Token")
	}

	respondWithJSON(w, 204, "")
}

func dbUserToSafeJSON(db database.User, tk string, rTK string) User {
	return User{
		db.ID,
		db.CreatedAt,
		db.UpdatedAt,
		db.Email,
		"",
		tk,
		rTK,
		db.IsChirpyRed,
	}
}

func dbUserToUserJSON(db database.UpdateUserRow) User {
	return User{
		db.ID,
		db.CreatedAt,
		db.UpdatedAt,
		db.Email,
		"",
		"",
		"",
		db.IsChirpyRed,
	}
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
		respondWithError(w, 401, "Unauthorized")
		return
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
		respondWithError(w, 401, "Unauthorized")
		return
	}
	respondWithJSON(w, 201, dbChirpToJSON(insertedChirp))
}

func (conf *apiConfig) getChirpsHandlerFunc(w http.ResponseWriter, r *http.Request) {
	f := r.URL.Query().Get("author_id")
	var id uuid.UUID
	var filtering bool
	var err error
	if f != "" {
		filtering = true
		id, err = uuid.Parse(f)
		if err != nil {
			filtering = false
		}
	}

	s := r.URL.Query().Get("sort")
	sortAscending := s != "desc"

	chirps, err := conf.dbQueries.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, 400, "Failed to get Chirps")
	}
	var jsonChirps []Chirp
	for _, v := range chirps {
		if filtering {
			if v.UserID != id {
				continue
			}
		}
		jsonChirp := dbChirpToJSON(v)
		jsonChirps = append(jsonChirps, jsonChirp)

		sort.Slice(jsonChirps, func(i, j int) bool {
			if sortAscending {
				return jsonChirps[i].CreatedAt.Before(jsonChirps[j].CreatedAt)
			}
			return jsonChirps[i].CreatedAt.After(jsonChirps[j].CreatedAt)
		})

	}
	respondWithJSON(w, 200, jsonChirps)
}

func (conf *apiConfig) getChirpHandlerFunc(w http.ResponseWriter, r *http.Request) {
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 404, "Failed to parse ChirpID")
		return
	}
	chirp, err := conf.dbQueries.GetChirp(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, 404, "ChirpNotFound")
		return
	}
	respondWithJSON(w, 200, dbChirpToJSON(chirp))
}

func (conf *apiConfig) deleteChirpHandlerFunc(w http.ResponseWriter, r *http.Request) {
	tk, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Token not found")
		return
	}
	validUser, err := auth.ValidateJWT(tk, conf.JWTKey)
	if err != nil {
		respondWithError(w, 403, "User not Authorized")
		return
	}
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 404, "Failed to parse ChirpID")
		return
	}
	chirp, err := conf.dbQueries.GetChirp(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, 404, "ChirpNotFound")
		return
	}
	if chirp.UserID != validUser {
		respondWithError(w, 403, "User not Authorized")
		return
	}
	_, err = conf.dbQueries.DeleteChirp(r.Context(), chirp.ID)
	if err != nil {
		respondWithError(w, 404, "Chirp not found")
		return
	}
	if err == nil {
		w.WriteHeader(204)
	}
}

func dbChirpToJSON(db database.Chirp) Chirp {
	return Chirp{db.ID, db.CreatedAt, db.UpdatedAt, db.Body, db.UserID}
}

func (conf *apiConfig) webhookHandlerFunc(w http.ResponseWriter, r *http.Request) {
	tk, err := auth.GetAPIKey(r.Header)
	if err != nil || tk != conf.PolkaKey {
		respondWithError(w, 401, "Token not found")
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(204)
		return
	}
	defer r.Body.Close()

	var req WebHook
	err = json.Unmarshal(data, &req)
	if err != nil || req.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}
	uid, err := uuid.Parse(req.Data.Data)
	if err != nil {
		w.WriteHeader(204)
		return
	}
	_, err = conf.dbQueries.UpgradeUser(r.Context(), uid)
	if err != nil {
		w.WriteHeader(404)
		fmt.Println("Failed to upgrade User")
	}
	w.WriteHeader(204)
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

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	if err := writeJSONResponse(w, code, payload); err != nil {
		fmt.Printf("Failed at respondWithJSON: %v", err)
	}
}
