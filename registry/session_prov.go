package registry

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"github.com/docker/docker/utils"
)

// APIVersion2 /v2/
var v2HTTPRoutes = map[string]map[string]HTTPRoute{
	"GET": {
		"Version":       HTTPRoute("version"),                           // XXX DONE
		"ImageManifest": HTTPRoute("manifest/{imgname:.*}/{tagname}"),   // XXX DONE
		"ImageTags":     HTTPRoute("tags/{imgname:.*}"),                 // XXX
		"ImageBlobSum":  HTTPRoute("blob/{imgname:.*}/{sumtype}/{sum}"), // XXX
	},
	"PUT": {
		"ImageManifest": HTTPRoute("manifest/{imgname:.*}/{tagname}"),   // XXX
		"ImageBlobSum":  HTTPRoute("blob/{imgname:.*}/{sumtype}/{sum}"), // XXX
	},
	"POST": {
		"ImageBlob":      HTTPRoute("blob/{imgname:.*}/{sumtype}"),            // XXX DONE
		"ImageMountBlob": HTTPRoute("mountblob/{imgname:.*}/{sumtype}/{sum}"), // XXX DONE
	},
	"DELETE": {
		"ImageManifest": HTTPRoute("manifest/{imgname:.*}/{tagname}"), // XXX
	},
}

func (r *Session) GetV2Version(token []string) (*RegistryInfo, error) {
	if r.indexEndpoint.Version != APIVersion2 {
		return nil, ErrIncorrectAPIVersion
	}

	method := "GET"
	hr := v2HTTPRoutes[method]["Version"]
	log.Printf("[registry] Calling %q %s", method, r.indexEndpoint.String()+string(hr))

	req, err := r.reqFactory.NewRequest(method, r.indexEndpoint.String()+string(hr), nil)
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
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return &RegistryInfo{Version: strings.TrimSpace(string(buf))}, nil
}

// V2 Provenance POC
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

	method := "GET"
	hr := v2HTTPRoutes[method]["ImageManifest"]
	values := map[string]string{
		"imgname": imageName,
		"tagname": tagName,
	}
	log.Printf("[registry] Calling %q %s", method, r.indexEndpoint.String()+hr.Format(values))

	req, err := r.reqFactory.NewRequest(method, r.indexEndpoint.String()+hr.Format(values), nil)
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

	method := "POST"
	hr := v2HTTPRoutes[method]["ImageMountBlob"]
	values := map[string]string{
		"imgname": imageName,
		"sumtype": sumType,
		"sum":     sum,
	}
	log.Printf("[registry] Calling %q %s", method, r.indexEndpoint.String()+hr.Format(values))

	req, err := r.reqFactory.NewRequest(method, r.indexEndpoint.String()+hr.Format(values), nil)
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

// Push the image to the server for storage.
// 'layer' is an uncompressed reader of the blob to be pushed.
// The server will generate it's own checksum calculation.
func (r *Session) PostV2ImageBlob(imageName, sumType string, blobRdr io.Reader, token []string) (serverChecksum string, err error) {
	if r.indexEndpoint.Version != APIVersion2 {
		return "", ErrIncorrectAPIVersion
	}

	method := "POST"
	hr := v2HTTPRoutes[method]["ImageBlob"]
	values := map[string]string{
		"imgname": imageName,
		"sumtype": sumType,
	}
	log.Printf("[registry] Calling %q %s", method, r.indexEndpoint.String()+hr.Format(values))
	req, err := r.reqFactory.NewRequest(method, r.indexEndpoint.String()+hr.Format(values), blobRdr)
	if err != nil {
		return "", err
	}
	setTokenAuth(req, token)
	res, _, err := r.doRequest(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		if res.StatusCode == 401 {
			return "", errLoginRequired
		}
		return "", utils.NewHTTPRequestError(fmt.Sprintf("Server error: %d trying to push %s blob", res.StatusCode, imageName), res)
	}

	jsonBuf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("Error while reading the http response: %s", err)
	}

	// XXX this is a json struct from the registry, with its checksum
	return string(jsonBuf), nil
}

// Finally Push the (signed) manifest of the blobs we've just pushed
func (r *Session) PutV2ImageManifest(imageName, tagName string, manifestRdr io.Reader, token []string) error {
	if r.indexEndpoint.Version != APIVersion2 {
		return ErrIncorrectAPIVersion
	}

	method := "PUT"
	hr := v2HTTPRoutes[method]["ImageManifest"]
	values := map[string]string{
		"imgname": imageName,
		"tagname": tagName,
	}
	log.Printf("[registry] Calling %q %s", method, r.indexEndpoint.String()+hr.Format(values))
	req, err := r.reqFactory.NewRequest(method, r.indexEndpoint.String()+hr.Format(values), manifestRdr)
	if err != nil {
		return err
	}
	setTokenAuth(req, token)
	res, _, err := r.doRequest(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		if res.StatusCode == 401 {
			return errLoginRequired
		}
		return utils.NewHTTPRequestError(fmt.Sprintf("Server error: %d trying to push %s:%s manifest", res.StatusCode, imageName, tagName), res)
	}

	return nil
}
