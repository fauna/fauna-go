package fauna_test

import (
	"fmt"
	"log"

	"github.com/fauna/fauna-go/v3"
)

func ExampleEventFeed_Next() {
	client := fauna.NewClient("secret", fauna.DefaultTimeouts(), fauna.URL(fauna.EndpointLocal))

	query, queryErr := fauna.FQL(`Collection.byName("EventFeedTest")?.delete()
Collection.create({ name: "EventFeedTest" })
EventFeedTest.all().eventSource()`, nil)
	if queryErr != nil {
		log.Fatal(queryErr.Error())
	}

	feed, feedErr := client.FeedFromQuery(query, nil)
	if feedErr != nil {
		log.Fatal(feedErr.Error())
	}

	addOne, _ := fauna.FQL(`EventFeedTest.create({ foo: 'bar' })`, nil)
	_, addOneErr := client.Query(addOne)
	if addOneErr != nil {
		log.Fatal(addOneErr.Error())
	}

	for {
		var page fauna.FeedPage
		eventErr := feed.Next(&page)
		if eventErr != nil {
			log.Fatal(eventErr.Error())
		}

		for _, event := range page.Events {
			fmt.Println(event.Type)
		}

		if !page.HasNext {
			break
		}
	}

	// Output: add
}
