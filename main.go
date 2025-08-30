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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
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

func middlewareLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
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
	serveMux.HandleFunc("GET "+healthzPath, healthzHandler)
	serveMux.HandleFunc("GET "+metricsPath, cfg.metricsHandler().ServeHTTP)
	serveMux.HandleFunc("POST "+resetPath, cfg.resetHandler().ServeHTTP)

	port := "8080"
	server := &http.Server{
		Handler: serveMux,
		Addr:    ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", appPath, port)
	log.Fatal(server.ListenAndServe())
}
