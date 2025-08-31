package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/dmitriy-zverev/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	appPrefix := "/app/"
	apiPrefix := "/api"
	adminPrefix := "/admin"

	healthzPath := apiPrefix + "/healthz"
	metricsPath := adminPrefix + "/metrics"
	resetPath := adminPrefix + "/reset"
	chirpsPath := apiPrefix + "/chirps"
	usersPath := apiPrefix + "/users"

	cfg := apiConfig{
		dbQueries: dbQueries,
		platform:  platform,
	}

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
	serveMux.HandleFunc("POST "+usersPath, cfg.usersHandler)
	serveMux.HandleFunc("POST "+chirpsPath, cfg.chirpsHandler)

	port := "8080"
	server := &http.Server{
		Handler: serveMux,
		Addr:    ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", appPrefix, port)
	log.Fatal(server.ListenAndServe())
}
