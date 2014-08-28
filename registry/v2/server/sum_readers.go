package server

import (
	"compress/gzip"
	"crypto"
	"encoding/hex"
	"errors"
	"hash"
	"io"

	"github.com/docker/docker/pkg/tarsum"
)

// SumReader is able to read data from an internal io.Reader
// and produce a checksum of the data which has been read.
type SumReader interface {
	Sum(extra []byte) string
	Read(p []byte) (n int, err error)
	Close() error
}

var (
	supportedSumTypes map[string]func(io.Reader) (SumReader, error)
	// ErrSumTypeNotSupported indicates that the sum type is not supported.
	ErrSumTypeNotSupported = errors.New("sum type not supported")
)

// NewSumReader returns an instance of a SumReader which
// implements the given checksum type. Returns a nil
// SumReader if the given sum type is not supported.
func NewSumReader(sumTypeName string, r io.Reader) (SumReader, error) {
	contstructor, ok := supportedSumTypes[sumTypeName]
	if !ok {
		return nil, ErrSumTypeNotSupported
	}

	return contstructor(r)
}

// HashingSumReader implements SumWriter
// using a cryptographic hashing algorithm.
// The Read method is covered by the internal
// io.TeeReader
type HashingSumReader struct {
	io.Reader
	hash  hash.Hash
	label string
}

// Sum returns the current cryptographic hash
// digest of the data which has been written.
// The format is "<label>" + ":" + "<hexDigest>"
func (sr *HashingSumReader) Sum(extra []byte) string {
	rawSum := sr.hash.Sum(extra)
	hexDigest := hex.EncodeToString(rawSum)
	return sr.label + ":" + hexDigest
}

// Close is a nop since the hasher doesn't need closing.
// This method is defined to implement the HashingSumReader
// interface only, it returns nil.
func (sr *HashingSumReader) Close() error {
	return nil
}

// NewSHA256SumReader creates a new HashingSumReader
// configured to use the SHA 256 hashing algorithm.
func NewSHA256SumReader(r io.Reader) (SumReader, error) {
	hash := crypto.SHA256.New()

	return &HashingSumReader{
		Reader: io.TeeReader(r, hash),
		hash:   hash,
		label:  "sha256",
	}, nil
}

// GZTarSumReader implements a SumReader that firsts uncompresses
// data from the internal reader using GZip, then passes the data
// to TarSum (which expects uncompressed TAR data) which generates
// a checksum from the TAR data. Uncompressed TAR data is read out.
type GZTarSumReader struct {
	*tarsum.TarSum
	gunZipper *gzip.Reader
}

// Close closes the internal gzip decompressor.
func (sr *GZTarSumReader) Close() error {
	return sr.gunZipper.Close()
}

// NewTarSumReader creates a new SumReader that expects
// GZipped content from the given reader
func NewTarSumReader(r io.Reader) (SumReader, error) {
	gunZipper, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &GZTarSumReader{
		TarSum: &tarsum.TarSum{
			Reader:             gunZipper,
			DisableCompression: true,
		},
		gunZipper: gunZipper,
	}, nil
}

func init() {
	supportedSumTypes = map[string]func(io.Reader) (SumReader, error){
		"sha256":        NewSHA256SumReader,
		"tarsum+sha256": NewTarSumReader,
	}
}
