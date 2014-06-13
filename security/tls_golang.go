// +build !openssl

package security

import (
	"crypto/x509"
)

type golangSessionProfile struct {
	certPool *x509.CertPool
}

func (gsp *golangSessionProfile) AppendRootCACertsFromPEM(file []byte) bool {
	return gsp.certPool.AppendCertsFromPEM(file)
}

func NewSessionProfile() (*SessionProfile, error) {
	return &golangSessionProfile{certPool: x509.NewCertPool()}, nil
}
