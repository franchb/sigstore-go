package root

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSigstoreTrustedRoot(t *testing.T) {
	trustedRoot, err := GetDefaultTrustedRoot()
	assert.Nil(t, err)
	assert.NotNil(t, trustedRoot)
}