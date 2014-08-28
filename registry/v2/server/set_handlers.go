package server

import (
	"net/http"

	"github.com/docker/docker/registry/v2/routes"
	"github.com/gorilla/mux"
)

func setRegistryRouteHandlers(registryRouter *mux.Router) {
	routeHandlers := map[string]map[string]http.Handler{
		routes.ManifestsRouteName: {
			"GET":    http.HandlerFunc(getManifest),
			"PUT":    http.HandlerFunc(putManifest),
			"DELETE": http.HandlerFunc(deleteManifest),
		},
		routes.TagsRouteName: {
			"GET": http.HandlerFunc(getTags),
		},
		routes.DownloadBlobRouteName: {
			"GET": http.HandlerFunc(getBlob),
		},
		routes.UploadBlobRouteName: {
			"PUT": http.HandlerFunc(putBlob),
		},
		routes.MountBlobRouteName: {
			"POST": http.HandlerFunc(mountBlob),
		},
	}

	for routeName, handlerMapping := range routeHandlers {
		route := registryRouter.Get(routeName)

		subRouter := route.Subrouter()
		for methodName, handler := range handlerMapping {
			subRouter.Methods(methodName).Handler(handler)
		}
	}
}

func NewRegistryHandler() http.Handler {
	router := routes.NewRegistryRouter()
	setRegistryRouteHandlers(router)
	return router
}
