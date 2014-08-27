package registry

import (
	"bytes"
	"fmt"
	"testing"
)

func TestHTTPRoute(t *testing.T) {
	testData := []struct {
		route, expected string
		count           int
		values          map[string]string
	}{
		{"/path/{foo}/here", "/path/to/here", 1, map[string]string{"foo": "to"}},
		{"/path/to/here", "/path/to/here", 0, map[string]string{}},
		{"/{bar:.*}/{foo}/here", "/path/to/here", 2, map[string]string{"bar": "path", "foo": "to"}},
		{"/{bar:.*}/{foo}/{baz}", "/path/to/{baz}", 3, map[string]string{"bar": "path", "foo": "to"}},
	}
	for _, td := range testData {
		got := HTTPRoute(td.route).Format(td.values)
		if td.expected != got {
			t.Errorf("expected %q, got %q", td.expected, got)
		}
		if len(HTTPRoute(td.route).Keys()) != td.count {
			t.Errorf("%q: expected %d, got %d", td.route, len(HTTPRoute(td.route).Keys()), td.count)
		}
	}
}

func TestV2Stuffs(t *testing.T) {
	r := spawnTestRegistrySession(t)
	r.indexEndpoint.Version = APIVersion2

	regInfo, err := r.GetV2Version(TOKEN)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(regInfo)

	manifestBuf, err := r.GetV2ImageManifest("foo42/bar", "stable", TOKEN)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(len(manifestBuf))

	blobAvialable, err := r.PostV2ImageMountBlob("foo42/bar", "tarsum+sha256", "deadbeef", TOKEN)
	if err != nil {
		t.Error(err)
	}
	if !blobAvialable {
		fmt.Println("PUSH IT!")
	}

	checksum, err := r.PostV2ImageBlob("foo42/bar", "tarsum+sha256", bytes.NewReader([]byte("BLOB BLOB BLOB")), TOKEN)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(checksum)

	err = r.PutV2ImageManifest("foo42/bar", "latest", bytes.NewReader([]byte("BLOB BLOB BLOB")), TOKEN)
	if err != nil {
		t.Error(err)
	}

}
