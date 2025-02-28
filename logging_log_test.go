//go:build !go1.21

package fauna_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/fauna/fauna-go/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogLogger(t *testing.T) {
	t.Run("should be able to provide a custom logger", func(t *testing.T) {
		buf := new(bytes.Buffer)

		client := fauna.NewClient("secret", fauna.DefaultTimeouts(), fauna.WithLogger(CustomLogger{
			Output: buf,
		}), fauna.URL(fauna.EndpointLocal))
		assert.NotNil(t, client)

		query, queryErr := fauna.FQL(`42`, nil)
		require.NoError(t, queryErr)

		res, err := client.Query(query)
		require.NoError(t, err)

		var value int
		err = res.Unmarshal(&value)
		require.NoError(t, err)
		require.Equal(t, 42, value)

		assert.NotEmpty(t, buf)
	})
}

type CustomLogger struct {
	fauna.Logger

	Output *bytes.Buffer
}

func (c CustomLogger) Info(msg string) {
	_, _ = fmt.Fprint(os.Stdout, msg)
}

func (c CustomLogger) LogResponse(_ context.Context, requestBody []byte, res *http.Response) {
	_, _ = fmt.Fprintf(c.Output, "URL: %s\nStatus: %s\nBody: %s\n", res.Request.URL.String(), res.Status, string(requestBody))
}
