package server

import (
	"os"
	"path"
)

var (
	dataDirectory   = os.ExpandEnv("$HOME/registry_data")
	blobsDirectory  = path.Join(dataDirectory, "blobs")
	imagesDirectory = path.Join(dataDirectory, "images")
)
