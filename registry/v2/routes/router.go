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
	VersionRoutename      = "version"
)

func NewRegistryRouter() *mux.Router {
	router := mux.NewRouter()

	v2Router := router.PathPrefix("/v2/").Subrouter()

	// Version Info
	v2Router.Path("/version").Name(VersionRoutename)

	// Image Manifests
	v2Router.Path("/manifest/{imagename:[a-z0-9-._/]+}/{tagname:[a-zA-Z0-9-._]+}").Name(ManifestsRouteName)

	// List Image Tags
	v2Router.Path("/tags/{imagename:[a-z0-9-._/]+}").Name(TagsRouteName)

	// Download a blob
	v2Router.Path("/blob/{imagename:[a-z0-9-._/]+}/{sumtype:[a-z0-9_+-]+}/{sum:[a-fA-F0-9]{4,}}").Name(DownloadBlobRouteName)

	// Upload a blob
	v2Router.Path("/blob/{imagename:[a-z0-9-._/]+}/{sumtype:[a-z0-9_+-]+}").Name(UploadBlobRouteName)

	// Mounting a blob in an image
	v2Router.Path("/mountblob/{imagename:[a-z0-9-._/]+}/{sumtype:[a-z0-9_+-]+}/{sum:[a-fA-F0-9]{4,}}").Name(MountBlobRouteName)

	return router
}
