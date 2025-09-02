package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dmitriy-zverev/chirpy/internal/database"
	"github.com/dmitriy-zverev/chirpy/internal/handlers"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Config holds all configuration values for the application
type Config struct {
	DBUrl     string
	Platform  string
	JWTSecret []byte
	PolkaKey  []byte
	Port      string
}

// Route path constants
const (
	appPrefix        = "/app/"
	apiPrefix        = "/api"
	adminPrefix      = "/admin"
	healthzPath      = apiPrefix + "/healthz"
	metricsPath      = adminPrefix + "/metrics"
	resetPath        = adminPrefix + "/reset"
	chirpsPath       = apiPrefix + "/chirps"
	chirpPath        = apiPrefix + "/chirps/{chirpID}"
	usersPath        = apiPrefix + "/users"
	loginPath        = apiPrefix + "/login"
	refreshPath      = apiPrefix + "/refresh"
	revokePath       = apiPrefix + "/revoke"
	polkaWebhookPath = apiPrefix + "/polka/webhooks"
)

func main() {
	config, err := loadConfiguration()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	dbQueries, db, err := initializeDatabase(config.DBUrl)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	apiConfig := &handlers.ApiConfig{
		DbQueries: dbQueries,
		Platform:  config.Platform,
		JWTSecret: config.JWTSecret,
		PolkaKey:  config.PolkaKey,
	}

	mux := setupRoutes(apiConfig)

	log.Printf("Serving files from %s on port: %s\n", appPrefix, config.Port)
	if err := startServer(mux, config.Port); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

// loadConfiguration loads and validates all environment variables
func loadConfiguration() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DB_URL environment variable is required")
	}

	platform := os.Getenv("PLATFORM")
	if platform == "" {
		platform = "prod" // default to production
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	polkaKey := os.Getenv("POLKA_KEY")
	if polkaKey == "" {
		return nil, fmt.Errorf("POLKA_KEY environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // default port
	}

	return &Config{
		DBUrl:     dbURL,
		Platform:  platform,
		JWTSecret: []byte(jwtSecret),
		PolkaKey:  []byte(polkaKey),
		Port:      port,
	}, nil
}

// initializeDatabase sets up the database connection and returns queries instance
func initializeDatabase(dbURL string) (*database.Queries, *sql.DB, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("failed to ping database: %w", err)
	}

	dbQueries := database.New(db)
	return dbQueries, db, nil
}

// setupRoutes configures all HTTP routes and returns the configured ServeMux
func setupRoutes(cfg *handlers.ApiConfig) *http.ServeMux {
	mux := http.NewServeMux()

	// File server for static assets
	mux.Handle(
		appPrefix,
		cfg.MiddlewareMetricsInc(
			http.StripPrefix(
				appPrefix,
				http.FileServer(http.Dir(".")),
			),
		),
	)

	// API routes
	mux.HandleFunc("GET "+healthzPath, handlers.HealthzHandler)
	mux.HandleFunc("GET "+metricsPath, cfg.MetricsHandler().ServeHTTP)
	mux.HandleFunc("POST "+resetPath, cfg.ResetHandler().ServeHTTP)

	// User routes
	mux.HandleFunc("POST "+usersPath, cfg.UsersHandler)
	mux.HandleFunc("PUT "+usersPath, cfg.UsersPutHandler)
	mux.HandleFunc("POST "+loginPath, cfg.LoginHandler)

	// Token routes
	mux.HandleFunc("POST "+refreshPath, cfg.RefreshHandler)
	mux.HandleFunc("POST "+revokePath, cfg.RevokeHandler)

	// Chirp routes
	mux.HandleFunc("POST "+chirpsPath, cfg.ChirpsPostHandler)
	mux.HandleFunc("GET "+chirpsPath, cfg.ChirpsGetHandler)
	mux.HandleFunc("GET "+chirpPath, cfg.ChirpGetHandler)
	mux.HandleFunc("DELETE "+chirpPath, cfg.ChirpDeleteHandler)

	// Webhook routes
	mux.HandleFunc("POST "+polkaWebhookPath, cfg.PolkaHookPostHandler)

	return mux
}

// startServer creates and starts the HTTP server
func startServer(mux *http.ServeMux, port string) error {
	server := &http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}

	return server.ListenAndServe()
}
