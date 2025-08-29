package main

import (
	"log"
	"net/http"
)

func main() {
	appPath := "/app/"
	healthzPath := "/healthz"

	serveMux := http.NewServeMux()
	serveMux.Handle(
		appPath,
		http.StripPrefix(
			appPath,
			http.FileServer(http.Dir(".")),
		),
	)
	serveMux.HandleFunc(healthzPath, healthz)

	port := "8080"

	server := &http.Server{
		Handler: serveMux,
		Addr:    ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", appPath, port)
	log.Fatal(server.ListenAndServe())
}

func healthz(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
