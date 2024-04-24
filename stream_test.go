package fauna_test

import (
	"testing"

	"github.com/fauna/fauna-go"
	"github.com/stretchr/testify/require"
)

func TestStreaming(t *testing.T) {
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)
	t.Setenv(fauna.EnvFaunaSecret, "secret")

	client, clientErr := fauna.NewDefaultClient()
	require.NoError(t, clientErr)

	setupQ, _ := fauna.FQL(`
		Collection.byName('StreamingTest')?.delete()
		Collection.create({ name: 'StreamingTest' })
	`, nil)

	_, err := client.Query(setupQ)
	require.NoError(t, err)

	type TestDoc struct {
		Foo string `fauna:"foo"`
	}

	t.Run("multi-step streaming", func(t *testing.T) {
		t.Run("Stream events", func(t *testing.T) {
			streamQ, _ := fauna.FQL(`StreamingTest.all().toStream()`, nil)
			res, err := client.Query(streamQ)
			require.NoError(t, err)

			var stream fauna.Stream
			require.NoError(t, res.Unmarshal(&stream))

			events, err := client.Subscribe(stream)
			require.NoError(t, err)
			defer events.Close()

			event, err := events.Next()
			require.NoError(t, err)
			require.Equal(t, event.Type, fauna.StatusEvent)

			createQ, _ := fauna.FQL(`StreamingTest.create({ foo: 'bar' })`, nil)
			_, err = client.Query(createQ)
			require.NoError(t, err)

			event, err = events.Next()
			require.NoError(t, err)
			require.Equal(t, event.Type, fauna.AddEvent)

			var doc TestDoc
			require.NoError(t, event.Unmarshal(&doc))
			require.Equal(t, doc.Foo, "bar")
			require.NoError(t, events.Close())
		})

		t.Run("Handle subscription errors", func(t *testing.T) {
			events, err := client.Subscribe(fauna.Stream("abc1234=="))
			require.IsType(t, err, &fauna.ErrInvalidRequest{})
			require.Nil(t, events)
		})

		t.Run("Handle error events", func(t *testing.T) {
			streamQ, _ := fauna.FQL(`StreamingTest.all().map(doc => abort('oops')).toStream()`, nil)
			res, err := client.Query(streamQ)
			require.NoError(t, err)

			var stream fauna.Stream
			require.NoError(t, res.Unmarshal(&stream))

			events, err := client.Subscribe(stream)
			require.NoError(t, err)
			defer events.Close()

			event, err := events.Next()
			require.NoError(t, err)
			require.Equal(t, event.Type, fauna.StatusEvent)

			createQ, _ := fauna.FQL(`StreamingTest.create({ foo: 'bar' })`, nil)
			_, err = client.Query(createQ)
			require.NoError(t, err)

			event, err = events.Next()
			require.IsType(t, err, &fauna.ErrEvent{})
			require.Nil(t, event)

			evErr := err.(*fauna.ErrEvent)
			require.Equal(t, evErr.Code, "abort")
			require.Equal(t, evErr.Message, "Query aborted.")

			var msg string
			require.NoError(t, evErr.Unmarshal(&msg))
			require.Equal(t, msg, "oops")
			require.NoError(t, events.Close())
		})
	})
}
