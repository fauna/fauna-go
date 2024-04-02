package fauna_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/fauna/fauna-go"
	"github.com/stretchr/testify/assert"
)

func TestStreaming(t *testing.T) {
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)
	t.Setenv(fauna.EnvFaunaSecret, "secret")

	client, clientErr := fauna.NewDefaultClient()
	if !assert.NoError(t, clientErr) {
		return
	}

	t.Run("basic stream", func(t *testing.T) {
		t.Run("receive events", func(t *testing.T) {
			var (
				stream *fauna.Stream
				err    error
			)

			token := os.Getenv("TOKEN")
			stream, err = client.StreamToken(token)
			if !assert.NoError(t, err) {
				return
			}

			for {
				if event := <-stream.Start(); event != nil {
					fmt.Printf("=> %+v\n", event)
					continue
				}
				break
			}

			assert.NoError(t, stream.Error())
		})
	})
}
