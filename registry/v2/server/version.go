package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/docker/docker/registry"
)

func getVersion(w http.ResponseWriter, r *http.Request) {
	log.Println("Get Version")

	versionInfo := &registry.RegistryInfo{
		Version:    "2.0",
		Standalone: true,
	}

	encoder := json.NewEncoder(w)
	err := encoder.Encode(versionInfo)
	if err != nil {
		log.Printf("unable to JSON encode version info: %s\n", err)
		w.WriteHeader(500)
	}
}
