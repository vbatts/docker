package registry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// scans string for api version in the URL path. returns the trimmed hostname, if version found, string and API version.
func scanForApiVersion(hostname string) (string, APIVersion) {
	var (
		chunks        []string
		apiVersionStr string
	)
	if strings.HasSuffix(hostname, "/") {
		chunks = strings.Split(hostname[:len(hostname)-1], "/")
		apiVersionStr = chunks[len(chunks)-1]
	} else {
		chunks = strings.Split(hostname, "/")
		apiVersionStr = chunks[len(chunks)-1]
	}
	for k, v := range apiVersions {
		if apiVersionStr == v {
			hostname = strings.Join(chunks[:len(chunks)-1], "/")
			return hostname, k
		}
	}
	return hostname, DefaultAPIVersion
}

func NewEndpoint(hostname string) (*Endpoint, error) {
	var (
		endpoint        Endpoint
		trimmedHostname string
		err             error
	)
	trimmedHostname, endpoint.Version = scanForApiVersion(hostname)
	endpoint.URL, err = url.Parse(trimmedHostname)
	if err != nil {
		return nil, err
	}

	// TODO find a way to do scheme determination, with preference for https
	if len(endpoint.URL.Scheme) == 0 {
		endpoint.URL.Scheme = determineEndpointScheme(endpoint)
	}

	//if _, err := pingRegistryEndpoint(hostname); err != nil {
	//return nil, errors.New("Invalid Registry endpoint: " + err.Error())
	//}

	return &endpoint, nil
}

func determineEndpointScheme(e Endpoint) string {
	return DefaultScheme
}

var DefaultScheme = "https"

type Endpoint struct {
	URL     *url.URL
	Version APIVersion
}

// Get the formated URL for the root of this registry Endpoint
func (e Endpoint) String() string {
	return fmt.Sprintf("%s/v%d/", e.URL.String(), e.Version)
}

func (e Endpoint) Ping() (RegistryInfo, error) {
	if e.String() == IndexServerAddress() {
		// Skip the check, we now this one is valid
		// (and we never want to fallback to http in case of error)
		return RegistryInfo{Standalone: false}, nil
	}

	//
	req, err := http.NewRequest("GET", e.String()+"_ping", nil)
	if err != nil {
		return RegistryInfo{Standalone: false}, err
	}

	resp, _, err := doRequest(req, nil, ConnectTimeout)
	if err != nil {
		return RegistryInfo{Standalone: false}, err
	}

	defer resp.Body.Close()

	jsonString, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return RegistryInfo{Standalone: false}, fmt.Errorf("Error while reading the http response: %s", err)
	}

	// If the header is absent, we assume true for compatibility with earlier
	// versions of the registry. default to true
	info := RegistryInfo{
		Standalone: true,
	}
	if err := json.Unmarshal(jsonString, &info); err != nil {
		log.Printf("Error unmarshalling the _ping RegistryInfo: %s", err)
		// don't stop here. Just assume sane defaults
	}
	if hdr := resp.Header.Get("X-Docker-Registry-Version"); hdr != "" {
		log.Printf("Registry version header: '%s'", hdr)
		info.Version = hdr
	}
	log.Printf("RegistryInfo.Version: %q", info.Version)

	standalone := resp.Header.Get("X-Docker-Registry-Standalone")
	log.Printf("Registry standalone header: '%s'", standalone)
	// Accepted values are "true" (case-insensitive) and "1".
	if strings.EqualFold(standalone, "true") || standalone == "1" {
		info.Standalone = true
	} else if len(standalone) > 0 {
		// there is a header set, and it is not "true" or "1", so assume fails
		info.Standalone = false
	}
	log.Printf("RegistryInfo.Standalone: %q", info.Standalone)
	return info, nil
}
