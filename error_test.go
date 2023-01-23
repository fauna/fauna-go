package fauna

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNoSecretAuthenticationErrorRequest(t *testing.T) {
	client, err := DefaultClient()
	assert.NoError(t, err)
	client.url = previewUrl
	client.secret = "foobar"
	err = client.Query("", nil, nil)
	assert.Error(t, err)
	assert.IsType(t, AuthenticationError{}, err)
}
