package security

type SessionProfile interface {
	AppendRootCACertsFromPEM([]byte) bool
}
