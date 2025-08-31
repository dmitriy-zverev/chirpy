package main

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

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler() http.Handler {
	handler := func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		numberHits := cfg.fileserverHits.Load()
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

func (cfg *apiConfig) resetHandler() http.Handler {
	handler := func(w http.ResponseWriter, req *http.Request) {
		if cfg.platform != "dev" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if err := cfg.dbQueries.ResetUsers(context.Background()); err != nil {
			log.Printf("%v\n", err)
		}
		w.WriteHeader(http.StatusOK)
	}
	return http.HandlerFunc(handler)
}

func (cfg *apiConfig) usersHandler(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	params := parameters{}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&params); err != nil {
		log.Printf("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	user, err := cfg.dbQueries.CreateUser(context.Background(), params.Email)
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

func (cfg *apiConfig) chirpsHandler(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
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

	user, err := cfg.dbQueries.Getuser(context.Background(), params.UserID)
	if err != nil {
		log.Print("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	createChirpParams := database.CreateChirpParams{
		Body:   strings.Join(splittedBody, " "),
		UserID: uuid.NullUUID{UUID: user.ID, Valid: true},
	}
	chirp, err := cfg.dbQueries.CreateChirp(context.Background(), createChirpParams)
	if err != nil {
		log.Print("%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dat, err := json.Marshal(
		struct {
			Body   string        `json:"body"`
			UserID uuid.NullUUID `json:"user_id"`
		}{
			Body:   chirp.Body,
			UserID: chirp.UserID,
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
