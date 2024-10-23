package fauna

import (
	"encoding/json"
)

type EventFeed struct {
	client *Client

	stream EventSource
	opts   []FeedOptFn

	decoder *json.Decoder

	lastCursor string
}

func newEventFeed(client *Client, token EventSource, opts ...FeedOptFn) (*EventFeed, error) {
	feed := &EventFeed{
		client: client,
		stream: token,
		opts:   opts,
	}

	if err := feed.reconnect(opts...); err != nil {
		return nil, err
	}

	return feed, nil
}

func (ef *EventFeed) reconnect(opts ...FeedOptFn) error {
	req := feedRequest{
		apiRequest: apiRequest{
			ef.client.ctx,
			ef.client.headers,
		},
		Stream: ef.stream,
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

// FeedResponse represents the response from the EventFeed.Events
type FeedResponse struct {
	Events  []Event `json:"events"`
	Cursor  string  `json:"cursor"`
	HasNext bool    `json:"has_next"`
	Stats   Stats   `json:"stats"`
}

// Events return the next FeedResponse from the EventFeed
func (ef *EventFeed) Events() (*FeedResponse, error) {
	var response FeedResponse
	if err := ef.reconnect(); err != nil {
		return nil, err
	}

	if err := ef.decoder.Decode(&response); err != nil {
		return nil, err
	}

	ef.lastCursor = response.Cursor

	return &response, nil
}
