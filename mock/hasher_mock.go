package mock

import "crypto/sha256"

// HasherMock that will be used for testing
type HasherMock struct {
}

// Compute will output the SHA's equivalent of the input string
func (sha HasherMock) Compute(s string) []byte {
	h := sha256.New()
	_, _ = h.Write([]byte(s))
	return h.Sum(nil)
}

// Size returns the required size in bytes
func (HasherMock) Size() int {
	return sha256.Size
}

// IsInterfaceNil returns true if there is no value under the interface
func (sha HasherMock) IsInterfaceNil() bool {
	return false
}
