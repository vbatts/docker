package server

import (
	"log"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
)

func mountBlob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("Mount Blob: %#v\n", vars)

	_, ok := vars["imagename"]
	if !ok {
		w.WriteHeader(404)
		return
	}

	sumType, ok := vars["sumtype"]
	if !ok {
		w.WriteHeader(404)
		return
	}

	sum, ok := vars["sum"]
	if !ok {
		w.WriteHeader(404)
		return
	}

	prefix1, prefix2 := sum[:2], sum[2:4]

	blobPath := path.Join(blobsDirectory, sumType, prefix1, prefix2, sum)
	fileInfo, err := os.Lstat(blobPath)
	if err != nil {
		errStatus := 500
		if os.IsNotExist(err) {
			// The blob does not exist. Indicate to the client that they should upload it.
			errStatus = 300
		}
		log.Printf("unable to open blob file %q: %s\n", blobPath, err)
		w.WriteHeader(errStatus)
		return
	}

	if !fileInfo.Mode().IsRegular() {
		log.Printf("unable to associate blob file %q: not a regular file", blobPath)
		w.WriteHeader(500)
		return
	}

	// The blob exists and is a regular file! On this naive server, that's OK.
	// We don't really have access control lists to worry about, everything is public.
	// TODO: return some content.
}
