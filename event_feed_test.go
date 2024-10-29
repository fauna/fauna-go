package fauna_test

import (
	"testing"
	"time"

	"github.com/fauna/fauna-go/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventFeed(t *testing.T) {
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)
	t.Setenv(fauna.EnvFaunaSecret, "secret")

	client, clientErr := fauna.NewDefaultClient()
	require.NoError(t, clientErr)

	resetCollection(t, client)

	t.Run("returns errors correctly", func(t *testing.T) {
		t.Run("should error when query doesn't return an event source", func(t *testing.T) {
			query, queryErr := fauna.FQL(`42`, nil)
			require.NoError(t, queryErr)

			_, feedErr := client.FeedFromQuery(query)
			require.ErrorContains(t, feedErr, "query should return a fauna.EventSource but got int")
		})
	})

	t.Run("can use event feeds from a query", func(t *testing.T) {
		query, queryErr := fauna.FQL(`EventFeedTest.all().eventSource()`, nil)
		require.NoError(t, queryErr, "failed to create a query for EventSource")

		feed, feedErr := client.FeedFromQuery(query)
		require.NoError(t, feedErr, "failed to init events feed")

		var (
			start = 5
			end   = 20
		)

		createOne(t, client, feed)
		createMultipleDocs(t, client, start, end)

		eventsRes, eventsErr := feed.Events()
		require.NoError(t, eventsErr, "failed to get events from EventSource")
		require.Equal(t, end-start, len(eventsRes.Events), "unexpected number of events")
	})

	t.Run("can get events from EventSource", func(t *testing.T) {
		t.Run("can get an EventSource", func(t *testing.T) {
			eventSource := getEventSource(t, client)
			require.NotNil(t, eventSource, "failed to get an EventSource")
		})

		t.Run("get events from an EventSource", func(t *testing.T) {
			eventSource := getEventSource(t, client)

			feed, feedErr := client.Feed(eventSource)
			require.NoError(t, feedErr, "failed to init events feed")

			var (
				start = 5
				end   = 20
			)

			createOne(t, client, feed)
			createMultipleDocs(t, client, start, end)

			eventsRes, eventsErr := feed.Events()
			require.NoError(t, eventsErr, "failed to get events from EventSource")
			require.Equal(t, end-start, len(eventsRes.Events), "unexpected number of events")
		})
	})

	t.Run("can get events from history", func(t *testing.T) {
		resetCollection(t, client)

		createOne(t, client, nil)

		eventSource := getEventSource(t, client)
		require.NotNil(t, eventSource, "failed to get an EventSource")

		feed, feedErr := client.Feed(eventSource)
		require.NoError(t, feedErr, "failed to init events feed")

		eventsRes, eventsErr := feed.Events()
		require.NoError(t, eventsErr, "failed to get events")
		require.Equal(t, 0, len(eventsRes.Events), "unexpected number of events")

		eventSource = getEventSource(t, client)
		require.NotNil(t, eventSource, "failed to get an EventSource")

		feed, feedErr = client.FeedWithStartTime(eventSource, time.Now().Add(-time.Minute*10))
		require.NoError(t, feedErr, "failed to init events feed")

		feedRes, eventsErr := feed.Events()
		require.NoError(t, eventsErr, "failed to get events")
		require.Equal(t, 1, len(feedRes.Events), "unexpected number of events")
	})
}

func resetCollection(t *testing.T, client *fauna.Client) {
	t.Helper()

	setupQuery, setupQueryErr := fauna.FQL(`Collection.byName("EventFeedTest")?.delete()
Collection.create({ name: "EventFeedTest" })`, nil)
	require.NoError(t, setupQueryErr, "setup query error: %s", setupQueryErr)

	_, setupErr := client.Query(setupQuery)
	require.NoError(t, setupErr, "setup error: %s", setupErr)
}

func getEventSource(t *testing.T, client *fauna.Client) fauna.EventSource {
	t.Helper()

	query, queryErr := fauna.FQL(`EventFeedTest.all().eventSource()`, nil)
	require.NoError(t, queryErr, "failed to create a query for EventSource")

	feedRes, feedResErr := client.Query(query)
	require.NoError(t, feedResErr, "failed to init events feed")

	var eventSource fauna.EventSource
	unmarshalErr := feedRes.Unmarshal(&eventSource)
	require.NoError(t, unmarshalErr, "failed to unmarshal EventSource")
	require.NotNil(t, eventSource, "event source is nil")
	require.NotEmpty(t, eventSource, "event source is empty")

	return eventSource
}

func createOne(t *testing.T, client *fauna.Client, feed *fauna.EventFeed) {
	t.Helper()

	createOneQuery, createOneQueryErr := fauna.FQL("EventFeedTest.create({ foo: 'bar' })", nil)
	require.NoError(t, createOneQueryErr, "failed to init query for create statement")
	require.NotNil(t, createOneQuery, "create statement is nil")

	_, createOneErr := client.Query(createOneQuery)
	require.NoError(t, createOneErr, "failed to create a document")

	if feed == nil {
		return
	}

	eventsRes, eventsErr := feed.Events()
	require.NoError(t, eventsErr, "failed to get events")

	assert.Equal(t, 1, len(eventsRes.Events), "unexpected number of events")
}

func createMultipleDocs(t *testing.T, client *fauna.Client, start int, end int) {
	t.Helper()

	query, queryErr := fauna.FQL(`Set.sequence(${start}, ${end}).forEach(n => EventFeedTest.create({ n: n }))`, map[string]any{
		"start": start,
		"end":   end,
	})
	require.NoError(t, queryErr, "failed to init query for create statement")

	_, err := client.Query(query)
	require.NoError(t, err)
}
