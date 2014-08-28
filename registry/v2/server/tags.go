package server

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
)

func getTags(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("Get Tags: %#v\n", vars)

	imageName, ok := vars["imagename"]
	if !ok {
		w.WriteHeader(404)
		return
	}

	manifestDir := path.Join(imagesDirectory, imageName)
	fileInfos, err := ioutil.ReadDir(manifestDir)
	if err != nil {
		errStatus := 500
		if os.IsNotExist(err) {
			errStatus = 404
		}
		log.Printf("unable to list image tags %q: %s\n", manifestDir, err)
		w.WriteHeader(errStatus)
		return
	}

	tagNames := make([]string, 0, len(fileInfos))

	for _, fileInfo := range fileInfos {
		if !fileInfo.Mode().IsRegular() {
			continue
		}
		tagNames = append(tagNames, fileInfo.Name())
	}

	encoder := json.NewEncoder(w)

	type tagSet struct {
		Tags []string `json:"tags"`
	}

	err = encoder.Encode(tagSet{tagNames})
	if err != nil {
		log.Printf("unable to json-encode tags: %s\n", err)
		w.WriteHeader(500)
	}
}
