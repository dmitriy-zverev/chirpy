package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dmitriy-zverev/chirpy/internal/auth"
	"github.com/dmitriy-zverev/chirpy/internal/database"
	"github.com/google/uuid"
)

type ApiConfig struct {
	FileserverHits atomic.Int32
	DbQueries      *database.Queries
	Platform       string
	JWTSecret      []byte
}

func (cfg *ApiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.FileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *ApiConfig) MetricsHandler() http.Handler {
	handler := func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		numberHits := cfg.FileserverHits.Load()
		numberHitsStr := strconv.Itoa(int(numberHits))
		w.Write(
			[]byte(fmt.Sprintf(
				`<html>
<body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited %s times!</p>
</body>
</html>`,
				numberHitsStr,
			)),
		)
	}
	return http.HandlerFunc(handler)
}

func (cfg *ApiConfig) ResetHandler() http.Handler {
	handler := func(w http.ResponseWriter, req *http.Request) {
		if cfg.Platform != "dev" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if err := cfg.DbQueries.ResetUsers(context.Background()); err != nil {
			log.Printf("%v\n", err)
		}
		w.WriteHeader(http.StatusOK)
	}
	return http.HandlerFunc(handler)
}

func (cfg *ApiConfig) UsersHandler(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	params := parameters{}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&params); err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	createUserParams := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	}
	user, err := cfg.DbQueries.CreateUser(context.Background(), createUserParams)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(
		struct {
			Id         string `json:"id"`
			Created_at string `json:"created_at"`
			Updated_at string `json:"updated_at"`
			Email      string `json:"email"`
		}{
			Id:         user.ID.String(),
			Created_at: user.CreatedAt.String(),
			Updated_at: user.UpdatedAt.String(),
			Email:      user.Email,
		},
	)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(resp)
}

func (cfg *ApiConfig) ChirpsPostHandler(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	params := parameters{}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	userToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userId, err := auth.ValidateJWT(userToken, string(cfg.JWTSecret))
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if len(params.Body) > 140 {
		type returnVals struct {
			Error string `json:"error"`
		}

		respBody := returnVals{
			Error: "Chirp is too long",
		}
		dat, err := json.Marshal(respBody)
		if err != nil {
			log.Printf("%v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(dat)
		return
	}

	splittedBody := strings.Split(params.Body, " ")
	profoundWords := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}

	for i, word := range splittedBody {
		if slices.Contains(profoundWords, strings.ToLower(word)) {
			splittedBody[i] = "****"
		}
	}

	createChirpParams := database.CreateChirpParams{
		Body:   strings.Join(splittedBody, " "),
		UserID: userId,
	}
	chirp, err := cfg.DbQueries.CreateChirp(context.Background(), createChirpParams)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dat, err := json.Marshal(
		struct {
			Id        string    `json:"id"`
			CreatedAt string    `json:"created_at"`
			UpdatedAt string    `json:"updated_at"`
			Body      string    `json:"body"`
			UserID    uuid.UUID `json:"user_id"`
		}{
			Id:        chirp.ID.String(),
			CreatedAt: chirp.CreatedAt.String(),
			UpdatedAt: chirp.UpdatedAt.String(),
			Body:      chirp.Body,
			UserID:    userId,
		},
	)
	if err != nil {
		log.Printf("Error marshalling JSON: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(dat)
}

func (cfg *ApiConfig) ChirpsGetHandler(w http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.DbQueries.GetChirps(context.Background())
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type chirpJson struct {
		Id        string `json:"id"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Body      string `json:"body"`
		UserID    string `json:"user_id"`
	}

	chirpsJsons := []chirpJson{}

	for _, chirp := range chirps {
		newChirpStruct := chirpJson{
			Id:        chirp.ID.String(),
			CreatedAt: chirp.CreatedAt.String(),
			UpdatedAt: chirp.UpdatedAt.String(),
			Body:      chirp.Body,
			UserID:    chirp.UserID.String(),
		}
		chirpsJsons = append(chirpsJsons, newChirpStruct)
	}

	dat, err := json.Marshal(chirpsJsons)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(dat)
}

func (cfg *ApiConfig) ChirpGetHandler(w http.ResponseWriter, req *http.Request) {
	chirpId, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	chirp, err := cfg.DbQueries.GetChirp(context.Background(), chirpId)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	type chirpJson struct {
		Id        string `json:"id"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Body      string `json:"body"`
		UserID    string `json:"user_id"`
	}

	dat, err := json.Marshal(
		chirpJson{
			Id:        chirp.ID.String(),
			CreatedAt: chirp.CreatedAt.String(),
			UpdatedAt: chirp.UpdatedAt.String(),
			Body:      chirp.Body,
			UserID:    chirp.UserID.String(),
		},
	)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(dat)
}

func (cfg *ApiConfig) LoginHandler(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	params := parameters{}

	if err := json.NewDecoder(req.Body).Decode(&params); err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := cfg.DbQueries.LoginUser(context.Background(), params.Email)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := auth.CheckPasswordHash(params.Password, user.HashedPassword); err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	parsedDuration, err := time.ParseDuration(os.Getenv("JWT_EXPIRATION_TIME"))
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	jwtToken, err := auth.MakeJWT(user.ID, string(cfg.JWTSecret), parsedDuration)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	createRefreshToken := database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(time.Hour * 86400),
	}
	if _, err := cfg.DbQueries.CreateRefreshToken(
		context.Background(),
		createRefreshToken,
	); err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(
		struct {
			Id           string `json:"id"`
			Created_at   string `json:"created_at"`
			Updated_at   string `json:"updated_at"`
			Email        string `json:"email"`
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
		}{
			Id:           user.ID.String(),
			Created_at:   user.CreatedAt.String(),
			Updated_at:   user.UpdatedAt.String(),
			Email:        user.Email,
			Token:        jwtToken,
			RefreshToken: refreshToken,
		},
	)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (cfg *ApiConfig) RefreshHandler(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	refreshTokenRow, err := cfg.DbQueries.GetRefreshToken(
		context.Background(),
		token,
	)
	if strings.Contains(fmt.Sprintf("%v", err), "no row") ||
		refreshTokenRow.ExpiresAt.Before(time.Now().UTC()) ||
		refreshTokenRow.RevokedAt.Valid {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, err := cfg.DbQueries.GetUserFromRefreshToken(
		context.Background(),
		refreshTokenRow.Token,
	)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	parsedExpiresIn, err := time.ParseDuration("1h")
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jwtToken, err := auth.MakeJWT(user.ID, string(cfg.JWTSecret), parsedExpiresIn)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(
		struct {
			Token string `json:"token"`
		}{
			Token: jwtToken,
		},
	)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (cfg *ApiConfig) RevokeHandler(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	revokedAtParams := database.SetRevokedAtParams{
		Token:     token,
		RevokedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true},
	}
	if err := cfg.DbQueries.SetRevokedAt(context.Background(), revokedAtParams); err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *ApiConfig) UsersPutHandler(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	params := parameters{}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	authToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateJWT(authToken, string(cfg.JWTSecret))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	newHashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	changeEmailPasswordParams := database.ChangeEmailPasswordParams{
		ID:             userID,
		Email:          params.Email,
		HashedPassword: newHashedPassword,
	}
	if err := cfg.DbQueries.ChangeEmailPassword(
		context.Background(),
		changeEmailPasswordParams,
	); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	userRow, err := cfg.DbQueries.GetUser(
		context.Background(),
		userID,
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(
		struct {
			Id        string `json:"id"`
			Email     string `json:"email"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
		}{
			Id:        userID.String(),
			Email:     userRow.Email,
			CreatedAt: userRow.CreatedAt.String(),
			UpdatedAt: userRow.UpdatedAt.String(),
		},
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (cfg *ApiConfig) ChirpDeleteHandler(w http.ResponseWriter, req *http.Request) {
	chirpId, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	authToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateJWT(authToken, string(cfg.JWTSecret))
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	chirpRow, err := cfg.DbQueries.GetChirp(
		context.Background(),
		chirpId,
	)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if chirpRow.UserID != userID {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := cfg.DbQueries.DeleteChirp(
		context.Background(),
		chirpId,
	); err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
