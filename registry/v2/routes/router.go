package routes

import (
	"github.com/gorilla/mux"
)

const (
	ManifestsRouteName    = "manifests"
	TagsRouteName         = "tags"
	DownloadBlobRouteName = "downloadBlob"
	UploadBlobRouteName   = "uploadBlob"
	MountBlobRouteName    = "mountBlob"
)

func NewRegistryRouter() *mux.Router {
	router := mux.NewRouter()

	v2Route := router.PathPrefix("/v2/").Subrouter()

	// Image Manifests
	v2Route.Path("/manifest/{imagename:[a-z0-9-._/]+}/{tagname:[a-zA-Z0-9-._]+}").Name(ManifestsRouteName)

	// List Image Tags
	v2Route.Path("/tags/{imagename:[a-z0-9-._/]+}").Name(TagsRouteName)

	// Download a blob
	v2Route.Path("/blob/{imagename:[a-z0-9-._/]+}/{sumtype:[a-z0-9_+-]+}/{sum:[a-fA-F0-9]{4,}}").Name(DownloadBlobRouteName)

	// Upload a blob
	v2Route.Path("/blob/{imagename:[a-z0-9-._/]+}/{sumtype:[a-z0-9_+-]+}").Name(UploadBlobRouteName)

	// Mounting a blob in an image
	v2Route.Path("/mountblob/{imagename:[a-z0-9-._/]+}/{sumtype:[a-z0-9_+-]+}/{sum:[a-fA-F0-9]{4,}}").Name(MountBlobRouteName)

	return router
}
