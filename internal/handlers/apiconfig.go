package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/dmitriy-zverev/chirpy/internal/database"
	"github.com/google/uuid"
)

type ApiConfig struct {
	FileserverHits atomic.Int32
	DbQueries      *database.Queries
	Platform       string
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
		Email string `json:"email"`
	}

	params := parameters{}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&params); err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	user, err := cfg.DbQueries.CreateUser(context.Background(), params.Email)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
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
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(resp)
}

func (cfg *ApiConfig) ChirpsHandler(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body   string `json:"body"`
		UserID string `json:"user_id"`
	}

	params := parameters{}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
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

	userId, err := uuid.Parse(params.UserID)
	if err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
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
