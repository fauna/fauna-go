package fauna_test

import (
	"fmt"
	"log"

	"github.com/fauna/fauna-go/v2"
)

func ExampleEventFeed_Events() {
	client := fauna.NewClient("secret", fauna.DefaultTimeouts(), fauna.URL(fauna.EndpointLocal))

	query, queryErr := fauna.FQL(`Collection.byName("EventFeedTest")?.delete()
Collection.create({ name: "EventFeedTest" })
EventFeedTest.all().toStream()`, nil)
	if queryErr != nil {
		log.Fatal(queryErr.Error())
	}

	feed, feedErr := client.FeedFromQuery(query)
	if feedErr != nil {
		log.Fatal(feedErr.Error())
	}

	addOne, _ := fauna.FQL(`EventFeedTest.create({ foo: 'bar' })`, nil)
	_, addOneErr := client.Query(addOne)
	if addOneErr != nil {
		log.Fatal(addOneErr.Error())
	}

	for {
		res, eventErr := feed.Events()
		if eventErr != nil {
			log.Fatal(eventErr.Error())
		}

		for _, event := range res.Events {
			fmt.Println(event.Type)
		}

		if !res.HasNext {
			break
		}
	}

	// Output: add
}
