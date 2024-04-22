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

			sub, err := client.Subscribe(stream)
			require.NoError(t, err)
			defer sub.Close()

			event := <-sub.Events()
			require.NotNil(t, event)
			require.Equal(t, event.Type, "status")

			createQ, _ := fauna.FQL(`StreamingTest.create({ foo: 'bar' })`, nil)
			_, err = client.Query(createQ)
			require.NoError(t, err)

			event = <-sub.Events()
			require.NotNil(t, event)
			require.Equal(t, event.Type, "add")

			var doc TestDoc
			require.NoError(t, event.Unmarshal(&doc))
			require.Equal(t, doc.Foo, "bar")

			require.NoError(t, sub.Close())
			require.NoError(t, sub.Error())
		})

		t.Run("Handle subscription errors", func(t *testing.T) {
			_, err := client.Subscribe(fauna.Stream("abc1234=="))
			require.IsType(t, err, &fauna.ErrInvalidRequest{})
		})

		t.Run("Handle error events", func(t *testing.T) {
			streamQ, _ := fauna.FQL(`StreamingTest.all().map(doc => abort('oops')).toStream()`, nil)
			res, err := client.Query(streamQ)
			require.NoError(t, err)

			var stream fauna.Stream
			require.NoError(t, res.Unmarshal(&stream))

			sub, err := client.Subscribe(stream)
			require.NoError(t, err)
			defer sub.Close()

			event := <-sub.Events()
			require.NotNil(t, event)
			require.Equal(t, event.Type, "status")

			createQ, _ := fauna.FQL(`StreamingTest.create({ foo: 'bar' })`, nil)
			_, err = client.Query(createQ)
			require.NoError(t, err)

			event = <-sub.Events()
			require.NotNil(t, event)
			require.Equal(t, event.Type, "error")
			require.Equal(t, event.Error.Code, "abort")
			require.Equal(t, event.Error.Message, "Query aborted.")

			var msg string
			require.NoError(t, event.Error.Unmarshal(&msg))
			require.Equal(t, msg, "oops")
		})
	})
}
