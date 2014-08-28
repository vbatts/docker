package server

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gorilla/mux"
)

func getBlob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("Get Blob: %#v\n", vars)

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
	blobFile, err := os.Open(blobPath)
	if err != nil {
		errStatus := 500
		if os.IsNotExist(err) {
			errStatus = 404
		}
		log.Printf("unable to open blob file %q: %s\n", blobPath, err)
		w.WriteHeader(errStatus)
		return
	}

	bytesCopied, err := io.Copy(w, blobFile)
	if err != nil {
		log.Printf("unable to copy blob file %q: %s\n", blobPath, err)
		w.WriteHeader(500)
	} else {
		log.Printf("copied %d bytes from blob file %q\n", bytesCopied, blobPath)
	}
}

func putBlob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("Put Blob: %#v\n", vars)

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

	blobDir := path.Join(blobsDirectory, sumType)
	err := os.MkdirAll(blobDir, os.FileMode(0755))
	if err != nil {
		log.Printf("unable to create blob directory %q: %s\n", blobDir, err)
		w.WriteHeader(500)
		return
	}

	tempBlobFile, err := ioutil.TempFile(blobDir, "temp")
	if err != nil {
		log.Printf("unable to open temporary blob file %q: %s\n", tempBlobFile.Name(), err)
		w.WriteHeader(500)
		return
	}

	sumReader, err := NewSumReader(sumType, io.TeeReader(r.Body, tempBlobFile))
	if err != nil {
		log.Printf("unable to create %q sum reader: %s\n", sumType, err)
		tempBlobFile.Close()
		os.Remove(tempBlobFile.Name())
		if err == ErrSumTypeNotSupported {
			// sumType is not Supported
			w.WriteHeader(501)
		} else {
			// content type must not be what the sumReader expects.
			w.WriteHeader(400)
		}
		return
	}

	bytesCopied, err := io.Copy(ioutil.Discard, sumReader)
	tempBlobFile.Close()
	sumReader.Close()
	if err != nil {
		log.Printf("unable to copy request body to temp blob file %q: %s\n", tempBlobFile.Name(), err)
		// Delete temp file.
		os.Remove(tempBlobFile.Name())
		w.WriteHeader(500)
		return
	}

	log.Printf("copied %d bytes from request body to temp blob file %q\n", bytesCopied, tempBlobFile.Name())

	type sumReturn struct {
		Checksum string `json:"checksum"`
	}

	sumInfo := sumReturn{
		Checksum: strings.ToLower(sumReader.Sum(nil)),
	}

	// Split on the sumType delimiter to get the sum value.
	sum := strings.SplitN(sumInfo.Checksum, ":", 2)[1]

	prefix1, prefix2 := sum[:2], sum[2:4]

	blobDir = path.Join(blobsDirectory, sumType, prefix1, prefix2)
	err = os.MkdirAll(blobDir, os.FileMode(0755))
	if err != nil {
		log.Printf("unable to create blob directory %q: %s\n", blobDir, err)
		os.Remove(tempBlobFile.Name())
		w.WriteHeader(500)
		return
	}

	// Rename temp file.
	blobPath := path.Join(blobDir, sum)
	os.Rename(tempBlobFile.Name(), blobPath)
	// Set 201 Header.
	w.WriteHeader(201)

	// Write JSON body.
	encoder := json.NewEncoder(w)
	encoder.Encode(sumInfo)
}
