package main

import "net/http"

func main() {
	serveMux := http.NewServeMux()
	serveMux.Handle("/", http.FileServer(http.Dir(".")))

	port := "8080"

	server := &http.Server{
		Handler: serveMux,
		Addr:    ":" + port,
	}

	server.ListenAndServe()
}
