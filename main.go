package main

import (
	"log"
	"net/http"
)

func main() {
	filePathRoot := "/"

	serveMux := http.NewServeMux()
	serveMux.Handle(filePathRoot, http.FileServer(http.Dir(".")))

	port := "8080"

	server := &http.Server{
		Handler: serveMux,
		Addr:    ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", filePathRoot, port)
	log.Fatal(server.ListenAndServe())
}
