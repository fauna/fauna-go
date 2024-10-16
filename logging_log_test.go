//go:build !go1.21

package fauna_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/fauna/fauna-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomLogger(t *testing.T) {
	t.Run("should be able to provide a custom logger", func(t *testing.T) {
		buf := new(bytes.Buffer)

		client := fauna.NewClient("secret", fauna.DefaultTimeouts(), fauna.Logger(CustomLogger{
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
	fauna.DriverLogger

	Output *bytes.Buffer
}

func (c CustomLogger) Info(msg string) {
	_, _ = fmt.Fprintf(os.Stdout, msg)
}

func (c CustomLogger) LogResponse(_ context.Context, res *http.Response) {
	_, _ = fmt.Fprintf(c.Output, "URL: %s\nStatus: %s\n", res.Request.URL.String(), res.Status)
}
