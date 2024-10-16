package fauna_test

import (
	"io"
	"os"
	"testing"

	"github.com/fauna/fauna-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pipeStdOut(handler func()) ([]byte, error) {
	storeStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handler()

	_ = w.Close()

	out, _ := io.ReadAll(r)
	os.Stdout = storeStdout

	return out, nil
}

func TestLogger(t *testing.T) {
	t.Run("should not log by default", func(t *testing.T) {
		out, outErr := pipeStdOut(func() {
			logger := fauna.DefaultLogger()
			logger.Info("testing")
		})

		require.NoError(t, outErr)
		require.Empty(t, string(out))
	})

	t.Run("should write to stdout", func(t *testing.T) {
		logMessage := "now you see me"

		out, outErr := pipeStdOut(func() {
			t.Setenv("FAUNA_DEBUG", "0")

			logger := fauna.DefaultLogger()
			logger.Info(logMessage)
		})
		require.NoError(t, outErr)

		outStr := string(out)

		require.Contains(t, outStr, logMessage)
		t.Logf("out: %s", outStr)
	})

	t.Run("should be able to provide a custom logger", func(t *testing.T) {
		client := fauna.NewClient("secret", fauna.DefaultTimeouts(), fauna.Logger(CustomLogger{}), fauna.URL(fauna.EndpointLocal))
		assert.NotNil(t, client)

		query, queryErr := fauna.FQL(`42`, nil)
		require.NoError(t, queryErr)

		res, err := client.Query(query)
		require.NoError(t, err)

		var value int
		err = res.Unmarshal(&value)
		require.NoError(t, err)
		require.Equal(t, 42, value)
	})
}
