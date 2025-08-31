package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/dmitriy-zverev/chirpy/internal/database"
	"github.com/dmitriy-zverev/chirpy/internal/handlers"
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

	cfg := handlers.ApiConfig{
		DbQueries: dbQueries,
		Platform:  platform,
	}

	serveMux := http.NewServeMux()
	serveMux.Handle(
		appPrefix,
		cfg.MiddlewareMetricsInc(
			http.StripPrefix(
				appPrefix,
				http.FileServer(http.Dir(".")),
			),
		),
	)
	serveMux.HandleFunc("GET "+healthzPath, handlers.HealthzHandler)
	serveMux.HandleFunc("GET "+metricsPath, cfg.MetricsHandler().ServeHTTP)
	serveMux.HandleFunc("POST "+resetPath, cfg.ResetHandler().ServeHTTP)
	serveMux.HandleFunc("POST "+usersPath, cfg.UsersHandler)
	serveMux.HandleFunc("POST "+chirpsPath, cfg.ChirpsHandler)
	serveMux.HandleFunc("GET "+chirpsPath, cfg.ChirpsGetHandler)

	port := "8080"
	server := &http.Server{
		Handler: serveMux,
		Addr:    ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", appPrefix, port)
	log.Fatal(server.ListenAndServe())
}
