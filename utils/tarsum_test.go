package utils

import (
	"bytes"
	"crypto/rand"
	"github.com/dotcloud/docker/vendor/src/code.google.com/p/go/src/pkg/archive/tar"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

type testLayer struct {
	filename string
	reader   io.Reader
	jsonfile string
	gzip     bool
	tarsum   string
}

var testLayers = []testLayer{
	{
		filename: "testdata/46af0962ab5afeb5ce6740d4d91652e69206fc991fd5328c1a94d364ad00e457/layer.tar",
		jsonfile: "testdata/46af0962ab5afeb5ce6740d4d91652e69206fc991fd5328c1a94d364ad00e457/json",
		tarsum:   "tarsum+sha256:e58fcf7418d4390dec8e8fb69d88c06ec07039d651fedd3aa72af9972e7d046b"},
	{
		filename: "testdata/46af0962ab5afeb5ce6740d4d91652e69206fc991fd5328c1a94d364ad00e457/layer.tar",
		jsonfile: "testdata/46af0962ab5afeb5ce6740d4d91652e69206fc991fd5328c1a94d364ad00e457/json",
		gzip:     true,
		tarsum:   "tarsum+sha256:e58fcf7418d4390dec8e8fb69d88c06ec07039d651fedd3aa72af9972e7d046b"},
	{
		filename: "testdata/511136ea3c5a64f264b78b5433614aec563103b4d4702f3ba7d4d2698e22c158/layer.tar",
		jsonfile: "testdata/511136ea3c5a64f264b78b5433614aec563103b4d4702f3ba7d4d2698e22c158/json",
		tarsum:   "tarsum+sha256:ac672ee85da9ab7f9667ae3c32841d3e42f33cc52c273c23341dabba1c8b0c8b"},
	{
		reader: sizedTar(1024*1024, false), // a 1mb file,
		tarsum: "tarsum+sha256:6a239fc02a5ed61f07ccb88ba2d59979778a77e9d0976bf61162d65e0ba5cfa8"},
}

func sizedTar(size int64, isRand bool) io.Reader {
	b := bytes.NewBuffer([]byte{})
	tarW := tar.NewWriter(b)
	err := tarW.WriteHeader(&tar.Header{
		Name: "/testdata",
		Mode: 0755,
		Uid:  0,
		Gid:  0,
		Size: size,
	})
	if err != nil {
		return nil
	}
	var rBuf []byte
	if isRand {
		rBuf = make([]byte, 8)
		_, err = rand.Read(rBuf)
		if err != nil {
			return nil
		}
	} else {
		rBuf = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	}

	for i := int64(0); i < size/int64(8); i++ {
		tarW.Write(rBuf)
	}
	return b
}

func TestTarSums(t *testing.T) {
	for _, layer := range testLayers {
		var (
			fh  io.Reader
			err error
		)
		if len(layer.filename) > 0 {
			t.Log(os.Getwd())
			fh, err = os.Open(layer.filename)
			if err != nil {
				t.Errorf("failed to open %s: %s", layer.filename, err)
				continue
			}
		} else if layer.reader != nil {
			fh = layer.reader
		} else {
			// What else is there to test?
			t.Errorf("what to do with %#V", layer)
			continue
		}
		if file, ok := fh.(*os.File); ok {
			defer file.Close()
		}

		//                                  double negatives!
		ts := &TarSum{Reader: fh, DisableCompression: !layer.gzip}
		_, err = io.Copy(ioutil.Discard, ts)
		if err != nil {
			t.Errorf("failed to copy from %s: %s", layer.filename, err)
			continue
		}
		var gotSum string
		if len(layer.jsonfile) > 0 {
			jfh, err := os.Open(layer.jsonfile)
			if err != nil {
				t.Errorf("failed to open %s: %s", layer.jsonfile, err)
				continue
			}
			buf, err := ioutil.ReadAll(jfh)
			if err != nil {
				t.Errorf("failed to readAll %s: %s", layer.jsonfile, err)
				continue
			}
			gotSum = ts.Sum(buf)
		} else {
			gotSum = ts.Sum(nil)
		}

		if layer.tarsum != gotSum {
			t.Errorf("expecting [%s], but got [%s]", layer.tarsum, gotSum)
		}
	}
}

func Benchmark9kFile(b *testing.B) {
	buf := bytes.NewBuffer([]byte{})
	fh, err := os.Open("testdata/46af0962ab5afeb5ce6740d4d91652e69206fc991fd5328c1a94d364ad00e457/layer.tar")
	if err != nil {
		b.Error(err)
		return
	}
	n, err := io.Copy(buf, fh)
	fh.Close()

	b.SetBytes(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts := &TarSum{Reader: buf, DisableCompression: true}
		io.Copy(ioutil.Discard, ts)
		ts.Sum(nil)
	}
}

func Benchmark9kFileGzip(b *testing.B) {
	buf := bytes.NewBuffer([]byte{})
	fh, err := os.Open("testdata/46af0962ab5afeb5ce6740d4d91652e69206fc991fd5328c1a94d364ad00e457/layer.tar")
	if err != nil {
		b.Error(err)
		return
	}
	n, err := io.Copy(buf, fh)
	fh.Close()

	b.SetBytes(n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts := &TarSum{Reader: buf, DisableCompression: false}
		io.Copy(ioutil.Discard, ts)
		ts.Sum(nil)
	}
}

// this is a single big file in the tar archive
func Benchmark1mbFile(b *testing.B) {
	var buf *bytes.Buffer
	tarReader := sizedTar(1024*1024, true)
	if br, ok := tarReader.(*bytes.Buffer); ok {
		buf = br
	}
	b.SetBytes(1024 * 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts := &TarSum{Reader: buf, DisableCompression: true}
		io.Copy(ioutil.Discard, ts)
		ts.Sum(nil)
	}
}

// this is a single big file in the tar archive
func Benchmark1mbFileGzip(b *testing.B) {
	var buf *bytes.Buffer
	tarReader := sizedTar(1024*1024, true)
	if br, ok := tarReader.(*bytes.Buffer); ok {
		buf = br
	}
	b.SetBytes(1024 * 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts := &TarSum{Reader: buf, DisableCompression: false}
		io.Copy(ioutil.Discard, ts)
		ts.Sum(nil)
	}
}

// TODO make a tar archive full of sparse empty files
