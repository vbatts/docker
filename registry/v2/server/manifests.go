package server

import (
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
)

func getManifest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("Get Manifest: %#v\n", vars)

	imageName, ok := vars["imagename"]
	if !ok {
		w.WriteHeader(404)
		return
	}

	tagName, ok := vars["tagname"]
	if !ok {
		w.WriteHeader(404)
		return
	}

	manifestPath := path.Join(imagesDirectory, imageName, tagName)
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		errStatus := 500
		if os.IsNotExist(err) {
			errStatus = 404
		}
		log.Printf("unable to open manifest file %q: %s\n", manifestPath, err)
		w.WriteHeader(errStatus)
		return
	}

	// Manifest should be a JSON Web Signature
	w.Header().Set("Content-Type", "application/json")

	bytesCopied, err := io.Copy(w, manifestFile)
	if err != nil {
		log.Printf("unable to copy manifest file %q: %s\n", manifestPath, err)
		w.WriteHeader(500)
	} else {
		log.Printf("copied %d bytes from manifest file %q\n", bytesCopied, manifestPath)
	}
}

func putManifest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("Put Manifest: %#v\n", vars)

	imageName, ok := vars["imagename"]
	if !ok {
		w.WriteHeader(404)
		return
	}

	tagName, ok := vars["tagname"]
	if !ok {
		w.WriteHeader(404)
		return
	}

	manifestDir := path.Join(imagesDirectory, imageName)
	err := os.MkdirAll(manifestDir, os.FileMode(0755))
	if err != nil {
		log.Printf("unable to create manifest directory %q: %s\n", manifestDir, err)
		w.WriteHeader(500)
		return
	}

	manifestPath := path.Join(manifestDir, tagName)
	manifestFile, err := os.OpenFile(manifestPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		log.Printf("unable to open manifest file %q: %s\n", manifestPath, err)
		w.WriteHeader(500)
		return
	}

	bytesCopied, err := io.Copy(manifestFile, r.Body)
	if err != nil {
		log.Printf("unable to copy request body to manifest file %q: %s\n", manifestPath, err)
		w.WriteHeader(500)
	} else {
		log.Printf("copied %d bytes from request body to manifest file %q\n", bytesCopied, manifestPath)
	}

	w.WriteHeader(201)
}

func deleteManifest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("Delete Manifest: %#v\n", vars)

	imageName, ok := vars["imagename"]
	if !ok {
		w.WriteHeader(404)
		return
	}

	tagName, ok := vars["tagname"]
	if !ok {
		w.WriteHeader(404)
		return
	}

	manifestPath := path.Join(imagesDirectory, imageName, tagName)
	err := os.Remove(manifestPath)
	if err != nil {
		errStatus := 500
		if os.IsNotExist(err) {
			errStatus = 404
		}
		log.Printf("unable to remove manifest file %q: %s\n", manifestPath, err)
		w.WriteHeader(errStatus)
		return
	}

	w.WriteHeader(204)
}
