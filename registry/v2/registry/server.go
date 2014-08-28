package main

import (
	"log"
	"net/http"
	"time"

	registryServer "github.com/docker/docker/registry/v2/server"
)

func main() {
	server := &http.Server{
		Addr:           ":8080",
		Handler:        registryServer.NewRegistryHandler(),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(server.ListenAndServe())
}
