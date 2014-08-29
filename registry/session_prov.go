package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"

	"github.com/docker/docker/registry/v2/routes"
	"github.com/docker/docker/utils"
)

// APIVersion2 /v2/
var v2HTTPRoutes = routes.NewRegistryRouter()

func getV2URL(e *Endpoint, routeName string, vars map[string]string) (*url.URL, error) {
	route := v2HTTPRoutes.Get(routeName)
	if route == nil {
		return nil, fmt.Errorf("unknown regisry v2 route name: %q", routeName)
	}

	varReplace := make([]string, 0, len(vars)*2)
	for key, val := range vars {
		varReplace = append(varReplace, key, val)
	}

	routePath, err := route.URLPath(varReplace...)
	if err != nil {
		return nil, fmt.Errorf("unable to make registry route %q with vars %v: %s", routeName, vars, err)
	}

	return &url.URL{
		Scheme: e.URL.Scheme,
		Host:   e.URL.Host,
		Path:   routePath.Path,
	}, nil
}

// V2 Provenance POC

func (r *Session) GetV2Version(token []string) (*RegistryInfo, error) {
	if r.indexEndpoint.Version != APIVersion2 {
		return nil, ErrIncorrectAPIVersion
	}

	routeURL, err := getV2URL(r.indexEndpoint, routes.VersionRoutename, nil)
	if err != nil {
		return nil, err
	}

	method := "GET"
	log.Printf("[registry] Calling %q %s", method, routeURL.String())

	req, err := r.reqFactory.NewRequest(method, routeURL.String(), nil)
	if err != nil {
		return nil, err
	}
	setTokenAuth(req, token)
	res, _, err := r.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, utils.NewHTTPRequestError(fmt.Sprintf("Server error: %d fetching Version", res.StatusCode), res)
	}

	decoder := json.NewDecoder(res.Body)
	versionInfo := new(RegistryInfo)

	err = decoder.Decode(versionInfo)
	if err != nil {
		return nil, fmt.Errorf("unable to decode GetV2Version JSON response: %s", err)
	}

	return versionInfo, nil
}

//
// 1) Check if TarSum of each layer exists /v2/
//  1.a) if 200, continue
//  1.b) if 300, then push the
//  1.c) if anything else, err
// 2) PUT the created/signed manifest
//
func (r *Session) GetV2ImageManifest(imageName, tagName string, token []string) ([]byte, error) {
	if r.indexEndpoint.Version != APIVersion2 {
		return nil, ErrIncorrectAPIVersion
	}

	vars := map[string]string{
		"imagename": imageName,
		"tagname":   tagName,
	}

	routeURL, err := getV2URL(r.indexEndpoint, routes.ManifestsRouteName, vars)
	if err != nil {
		return nil, err
	}

	method := "GET"
	log.Printf("[registry] Calling %q %s", method, routeURL.String())

	req, err := r.reqFactory.NewRequest(method, routeURL.String(), nil)
	if err != nil {
		return nil, err
	}
	setTokenAuth(req, token)
	res, _, err := r.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		if res.StatusCode == 401 {
			return nil, errLoginRequired
		}
		return nil, utils.NewHTTPRequestError(fmt.Sprintf("Server error: %d trying to fetch for %s:%s", res.StatusCode, imageName, tagName), res)
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Error while reading the http response: %s", err)
	}
	return buf, nil
}

// - Succeeded to mount for this image scope
// - Failed with no error (So continue to Push the Blob)
// - Failed with error
func (r *Session) PostV2ImageMountBlob(imageName, sumType, sum string, token []string) (bool, error) {
	if r.indexEndpoint.Version != APIVersion2 {
		return false, ErrIncorrectAPIVersion
	}

	vars := map[string]string{
		"imagename": imageName,
		"sumtype":   sumType,
		"sum":       sum,
	}

	routeURL, err := getV2URL(r.indexEndpoint, routes.MountBlobRouteName, vars)
	if err != nil {
		return false, err
	}

	method := "POST"
	log.Printf("[registry] Calling %q %s", method, routeURL.String())

	req, err := r.reqFactory.NewRequest(method, routeURL.String(), nil)
	if err != nil {
		return false, err
	}
	setTokenAuth(req, token)
	res, _, err := r.doRequest(req)
	if err != nil {
		return false, err
	}
	res.Body.Close() // close early, since we're not needing a body on this call .. yet?
	switch res.StatusCode {
	case 200:
		// return something indicating no push needed
		return true, nil
	case 300:
		// return something indicating blob push needed
		return false, nil
	}
	return false, fmt.Errorf("Failed to mount %q - %s:%s : %d", imageName, sumType, sum, res.StatusCode)
}

func (r *Session) GetV2ImageBlob(imageName, sumType, sum string, blobWrtr io.Writer, token []string) error {
	if r.indexEndpoint.Version != APIVersion2 {
		return ErrIncorrectAPIVersion
	}

	vars := map[string]string{
		"imagename": imageName,
		"sumtype":   sumType,
		"sum":       sum,
	}

	routeURL, err := getV2URL(r.indexEndpoint, routes.DownloadBlobRouteName, vars)
	if err != nil {
		return err
	}

	method := "GET"
	log.Printf("[registry] Calling %q %s", method, routeURL.String())
	req, err := r.reqFactory.NewRequest(method, routeURL.String(), nil)
	if err != nil {
		return err
	}
	setTokenAuth(req, token)
	res, _, err := r.doRequest(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 201 {
		if res.StatusCode == 401 {
			return errLoginRequired
		}
		return utils.NewHTTPRequestError(fmt.Sprintf("Server error: %d trying to push %s blob", res.StatusCode, imageName), res)
	}

	_, err = io.Copy(blobWrtr, res.Body)
	return err
}

// Push the image to the server for storage.
// 'layer' is an uncompressed reader of the blob to be pushed.
// The server will generate it's own checksum calculation.
func (r *Session) PutV2ImageBlob(imageName, sumType string, blobRdr io.Reader, token []string) (serverChecksum string, err error) {
	if r.indexEndpoint.Version != APIVersion2 {
		return "", ErrIncorrectAPIVersion
	}

	vars := map[string]string{
		"imagename": imageName,
		"sumtype":   sumType,
	}

	routeURL, err := getV2URL(r.indexEndpoint, routes.UploadBlobRouteName, vars)
	if err != nil {
		return "", err
	}

	method := "PUT"
	log.Printf("[registry] Calling %q %s", method, routeURL.String())
	req, err := r.reqFactory.NewRequest(method, routeURL.String(), blobRdr)
	if err != nil {
		return "", err
	}
	setTokenAuth(req, token)
	res, _, err := r.doRequest(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 201 {
		if res.StatusCode == 401 {
			return "", errLoginRequired
		}
		return "", utils.NewHTTPRequestError(fmt.Sprintf("Server error: %d trying to push %s blob", res.StatusCode, imageName), res)
	}

	type sumReturn struct {
		Checksum string `json:"checksum"`
	}

	decoder := json.NewDecoder(res.Body)
	var sumInfo sumReturn

	err = decoder.Decode(&sumInfo)
	if err != nil {
		return "", fmt.Errorf("unable to decode PutV2ImageBlob JSON response: %s", err)
	}

	// XXX this is a json struct from the registry, with its checksum
	return sumInfo.Checksum, nil
}

// Finally Push the (signed) manifest of the blobs we've just pushed
func (r *Session) PutV2ImageManifest(imageName, tagName string, manifestRdr io.Reader, token []string) error {
	if r.indexEndpoint.Version != APIVersion2 {
		return ErrIncorrectAPIVersion
	}

	vars := map[string]string{
		"imagename": imageName,
		"tagname":   tagName,
	}

	routeURL, err := getV2URL(r.indexEndpoint, routes.ManifestsRouteName, vars)
	if err != nil {
		return err
	}

	method := "PUT"
	log.Printf("[registry] Calling %q %s", method, routeURL.String())
	req, err := r.reqFactory.NewRequest(method, routeURL.String(), manifestRdr)
	if err != nil {
		return err
	}
	setTokenAuth(req, token)
	res, _, err := r.doRequest(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode != 201 {
		if res.StatusCode == 401 {
			return errLoginRequired
		}
		return utils.NewHTTPRequestError(fmt.Sprintf("Server error: %d trying to push %s:%s manifest", res.StatusCode, imageName, tagName), res)
	}

	return nil
}
