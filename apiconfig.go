package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/dmitriy-zverev/chirpy/internal/database"
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
