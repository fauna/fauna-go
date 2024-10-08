package fauna_test

import (
	"testing"

	"github.com/fauna/fauna-go/v2"
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

	t.Run("single-step streaming", func(t *testing.T) {
		t.Run("Stream events", func(t *testing.T) {
			streamQ, _ := fauna.FQL(`StreamingTest.all().toStream()`, nil)
			events, err := client.Stream(streamQ)
			require.NoError(t, err)
			defer events.Close()

			var event fauna.Event
			err = events.Next(&event)
			require.NoError(t, err)
			require.Equal(t, fauna.StatusEvent, event.Type)
		})

		t.Run("Fails on non-streamable values", func(t *testing.T) {
			streamQ, _ := fauna.FQL(`"I'm a string"`, nil)
			events, err := client.Stream(streamQ)
			require.ErrorContains(t, err, "expected query to return a fauna.Stream but got string")
			require.Nil(t, events)
		})
	})

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

			var event fauna.Event
			err = events.Next(&event)
			require.NoError(t, err)
			require.Equal(t, fauna.StatusEvent, event.Type)

			createQ, _ := fauna.FQL(`StreamingTest.create({ foo: 'bar' })`, nil)
			_, err = client.Query(createQ)
			require.NoError(t, err)

			err = events.Next(&event)
			require.NoError(t, err)
			require.Equal(t, fauna.AddEvent, event.Type)

			var doc TestDoc
			require.NoError(t, event.Unmarshal(&doc))
			require.Equal(t, "bar", doc.Foo)
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

			var event fauna.Event
			err = events.Next(&event)
			require.NoError(t, err)
			require.Equal(t, fauna.StatusEvent, event.Type)

			createQ, _ := fauna.FQL(`StreamingTest.create({ foo: 'bar' })`, nil)
			_, err = client.Query(createQ)
			require.NoError(t, err)

			err = events.Next(&event)
			require.IsType(t, err, &fauna.ErrEvent{})

			evErr := err.(*fauna.ErrEvent)
			require.Equal(t, "abort", evErr.Code)
			require.Equal(t, "Query aborted.", evErr.Message)

			var msg string
			require.NoError(t, evErr.Unmarshal(&msg))
			require.Equal(t, "oops", msg)
			require.NoError(t, events.Close())
		})

		t.Run("Resume a stream at a given start time", func(t *testing.T) {
			streamQ, _ := fauna.FQL(`StreamingTest.all().toStream()`, nil)
			res, err := client.Query(streamQ)
			require.NoError(t, err)

			var stream fauna.Stream
			require.NoError(t, res.Unmarshal(&stream))

			createFooQ, _ := fauna.FQL(`StreamingTest.create({ foo: 'foo' })`, nil)
			createBarQ, _ := fauna.FQL(`StreamingTest.create({ foo: 'bar' })`, nil)

			foo, err := client.Query(createFooQ)
			require.NoError(t, err)

			bar, err := client.Query(createBarQ)
			require.NoError(t, err)

			events, err := client.Subscribe(stream, fauna.StartTime(foo.TxnTime))
			require.NoError(t, err)
			defer events.Close()

			var event fauna.Event
			err = events.Next(&event)
			require.NoError(t, err)
			require.Equal(t, fauna.StatusEvent, event.Type)
			require.GreaterOrEqual(t, event.TxnTime, foo.TxnTime)

			err = events.Next(&event)
			require.NoError(t, err)
			require.Equal(t, fauna.AddEvent, event.Type)
			require.Equal(t, bar.TxnTime, event.TxnTime)
		})

		t.Run("Resume a stream at a given event cursor", func(t *testing.T) {
			streamQ, _ := fauna.FQL(`StreamingTest.all().toStream()`, nil)
			res, err := client.Query(streamQ)
			require.NoError(t, err)

			var stream fauna.Stream
			require.NoError(t, res.Unmarshal(&stream))

			events, err := client.Subscribe(stream)
			require.NoError(t, err)
			defer events.Close()

			createFooQ, _ := fauna.FQL(`StreamingTest.create({ foo: 'foo' })`, nil)
			createBarQ, _ := fauna.FQL(`StreamingTest.create({ foo: 'bar' })`, nil)

			foo, err := client.Query(createFooQ)
			require.NoError(t, err)

			bar, err := client.Query(createBarQ)
			require.NoError(t, err)

			var event fauna.Event
			err = events.Next(&event)
			require.NoError(t, err)
			require.Equal(t, fauna.StatusEvent, event.Type)

			err = events.Next(&event)
			require.NoError(t, err)
			require.Equal(t, fauna.AddEvent, event.Type)
			require.Equal(t, foo.TxnTime, event.TxnTime)
			events.Close()

			events, err = client.Subscribe(stream, fauna.EventCursor(event.Cursor))
			require.NoError(t, err)
			defer events.Close()

			err = events.Next(&event)
			require.NoError(t, err)
			require.Equal(t, fauna.StatusEvent, event.Type)
			require.GreaterOrEqual(t, foo.TxnTime, event.TxnTime)

			err = events.Next(&event)
			require.NoError(t, err)
			require.Equal(t, fauna.AddEvent, event.Type)
			require.Equal(t, bar.TxnTime, event.TxnTime)
		})
	})
}
