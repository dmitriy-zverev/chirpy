package main

import (
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	handler := func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		log.Printf("%d\n", cfg.fileserverHits.Load())
		next.ServeHTTP(w, req)
	}

	return http.HandlerFunc(handler)
}

func (cfg *apiConfig) metricsHandler() http.Handler {
	handler := func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		numberHits := cfg.fileserverHits.Load()
		numberHitsStr := strconv.Itoa(int(numberHits))
		w.Write([]byte("Hits: " + numberHitsStr))
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

func main() {
	appPath := "/app/"
	healthzPath := "/healthz"
	metricsPath := "/metrics"
	resetPath := "/reset"

	cfg := apiConfig{}

	serveMux := http.NewServeMux()
	serveMux.Handle(
		appPath,
		cfg.middlewareMetricsInc(
			http.StripPrefix(
				appPath,
				http.FileServer(http.Dir(".")),
			),
		),
	)
	serveMux.HandleFunc(healthzPath, healthzHandler)
	serveMux.HandleFunc(metricsPath, cfg.metricsHandler().ServeHTTP)
	serveMux.HandleFunc(resetPath, cfg.resetHandler().ServeHTTP)

	port := "8080"
	server := &http.Server{
		Handler: serveMux,
		Addr:    ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", appPath, port)
	log.Fatal(server.ListenAndServe())
}
