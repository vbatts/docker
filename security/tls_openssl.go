// +build openssl

package security

type opensslSessionProfile struct {
}

func (osp *opensslSessionProfile) AppendRootCACertsFromPEM(file []byte) bool {
	//return gsp.certPool.AppendCertsFromPEM(file)
	return false
}

func NewSessionProfile() (*SessionProfile, error) {
	return &opensslSessionProfile{}, nil
}
