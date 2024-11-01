package fauna

import (
	"encoding/json"
	"fmt"
)

// EventFeed represents an event feed subscription.
type EventFeed struct {
	client *Client

	source EventSource

	decoder *json.Decoder

	opts       []FeedOptFn
	lastCursor string
}

func newEventFeed(client *Client, source EventSource, fromQuery bool, opts ...FeedOptFn) (*EventFeed, error) {
	feed := &EventFeed{
		client: client,
		source: source,
		opts:   opts,
	}

	// init a feed request to validate feed options
	req, err := feed.newFeedRequest()
	if err != nil {
		return nil, err
	}
	if fromQuery && len(req.Cursor) > 0 {
		return nil, fmt.Errorf("cannot use EventFeedCursor with FeedFromQuery")
	}
	if req.StartTS > 0 && len(req.Cursor) > 0 {
		return nil, fmt.Errorf("cannot set both EventFeedStartTime and EventFeedCursor")
	}

	return feed, nil
}

func (ef *EventFeed) newFeedRequest(opts ...FeedOptFn) (*feedRequest, error) {
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

	return &req, nil
}

func (ef *EventFeed) open(opts ...FeedOptFn) error {
	req, err := ef.newFeedRequest(opts...)
	if err != nil {
		return err
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
