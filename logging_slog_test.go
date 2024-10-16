//go:build go1.21

package fauna_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/fauna/fauna-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlogLogger(t *testing.T) {
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)
	t.Setenv(fauna.EnvFaunaSecret, "secret")

	query, queryErr := fauna.FQL(`42`, nil)
	require.NoError(t, queryErr)

	t.Run("should be able to use the default logger", func(t *testing.T) {
		output, pipeErr := pipeStdOut(func() {
			t.Setenv(fauna.EnvFaunaDebug, "-4")

			client, clientErr := fauna.NewDefaultClient()
			require.NoError(t, clientErr)

			_, err := client.Query(query)
			require.NoError(t, err)
		})
		require.NoError(t, pipeErr)
		require.NotEmpty(t, output)
		t.Logf("output: %s", string(output))
	})

	t.Run("no output with warn level", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaDebug, "1")

		client, clientErr := fauna.NewDefaultClient()
		require.NoError(t, clientErr)

		output, pipeErr := pipeStdOut(func() {
			_, err := client.Query(query)
			require.NoError(t, err)
		})
		require.NoError(t, pipeErr)
		require.Empty(t, output)
	})

	t.Run("should be able to provide a custom logger", func(t *testing.T) {
		client := fauna.NewClient("secret", fauna.DefaultTimeouts(), fauna.Logger(CustomLogger{}), fauna.URL(fauna.EndpointLocal))
		assert.NotNil(t, client)

		res, err := client.Query(query)
		require.NoError(t, err)

		var value int
		err = res.Unmarshal(&value)
		require.NoError(t, err)
		require.Equal(t, 42, value)
	})
}

type CustomLogger struct {
	fauna.DriverLogger
}

func (c CustomLogger) Info(msg string, _ ...any) {
	_, _ = fmt.Fprintf(os.Stdout, msg)
}

func (c CustomLogger) LogResponse(_ context.Context, requestBody []byte, res *http.Response) {
	_, _ = fmt.Fprintf(os.Stdout, "URL: %s\nStatus: %s\nBody: %s\n", res.Request.URL.String(), res.Status, string(requestBody))
}
