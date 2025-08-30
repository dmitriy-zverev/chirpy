package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		cfg.fileserverHits.Store(0)
		w.Write([]byte("Hits have been reset"))
	}
	return http.HandlerFunc(handler)
}

func healthzHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func middlewareLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func validateChirpHandler(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		log.Printf("Error decoding parameters: %v", err)
		w.WriteHeader(http.StatusInternalServerError)

		type returnVals struct {
			Error string `json:"error"`
		}

		respBody := returnVals{
			Error: "Something went wrong",
		}
		dat, err := json.Marshal(respBody)
		if err != nil {
			log.Printf("Error marshalling JSON: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(dat)
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
			log.Printf("Error marshalling JSON: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(dat)
		return
	}

	type returnVals struct {
		CleanedBody string `json:"cleaned_body"`
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

	respBody := returnVals{
		CleanedBody: strings.Join(splittedBody, " "),
	}
	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(dat)
}

func main() {
	appPrefix := "/app/"
	apiPrefix := "/api"
	adminPrefix := "/admin"

	healthzPath := apiPrefix + "/healthz"
	metricsPath := adminPrefix + "/metrics"
	resetPath := adminPrefix + "/reset"
	validateChirpPath := apiPrefix + "/validate_chirp"

	cfg := apiConfig{}

	serveMux := http.NewServeMux()
	serveMux.Handle(
		appPrefix,
		cfg.middlewareMetricsInc(
			http.StripPrefix(
				appPrefix,
				http.FileServer(http.Dir(".")),
			),
		),
	)
	serveMux.HandleFunc("GET "+healthzPath, healthzHandler)
	serveMux.HandleFunc("GET "+metricsPath, cfg.metricsHandler().ServeHTTP)
	serveMux.HandleFunc("POST "+resetPath, cfg.resetHandler().ServeHTTP)
	serveMux.HandleFunc("POST "+validateChirpPath, validateChirpHandler)

	port := "8080"
	server := &http.Server{
		Handler: serveMux,
		Addr:    ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", appPrefix, port)
	log.Fatal(server.ListenAndServe())
}
