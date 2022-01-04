package mock

// HttpServerMock -
type HttpServerMock struct {
}

// ListenAndServe -
func (hsm *HttpServerMock) ListenAndServe() error {
	return nil
}

//Close -
func (hsm *HttpServerMock) Close() error {
	return nil
}
