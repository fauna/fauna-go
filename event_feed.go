package fauna

import (
	"encoding/json"
)

// EventFeed represents an event feed subscription.
type EventFeed struct {
	client *Client

	source EventSource

	decoder *json.Decoder

	opts       []FeedOptFn
	lastCursor string
}

func newEventFeed(client *Client, source EventSource, opts ...FeedOptFn) (*EventFeed, error) {
	feed := &EventFeed{
		client: client,
		source: source,
		opts:   opts,
	}

	return feed, nil
}

func (ef *EventFeed) open(opts ...FeedOptFn) error {
	req := feedRequest{
		apiRequest: apiRequest{
			ef.client.ctx,
			ef.client.headers,
		},
		Source: ef.source,
		Cursor: ef.lastCursor,
	}

	if (opts != nil) && (len(opts) > 0) {
		ef.opts = append(ef.opts, opts...)
	}

	for _, optFn := range ef.opts {
		optFn(&req)
	}

	byteStream, err := req.do(ef.client)
	if err != nil {
		return err
	}

	ef.decoder = json.NewDecoder(byteStream)

	return nil
}

// FeedPage represents the response from [fauna.EventFeed.Next]
type FeedPage struct {
	Events  []Event `json:"events"`
	Cursor  string  `json:"cursor"`
	HasNext bool    `json:"has_next"`
	Stats   Stats   `json:"stats"`
}

// Next retrieves the next FeedPage from the [fauna.EventFeed]
func (ef *EventFeed) Next(page *FeedPage) error {
	if err := ef.open(); err != nil {
		return err
	}

	if err := ef.decoder.Decode(&page); err != nil {
		return err
	}

	ef.lastCursor = page.Cursor

	return nil
}
