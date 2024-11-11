package fauna

import (
	"encoding/json"
)

// EventFeed represents an event feed subscription.
type EventFeed struct {
	client *Client

	source EventSource

	decoder *json.Decoder

	opts       *feedOptions
	lastCursor string
}

type feedOptions struct {
	PageSize *int
	Cursor   *string
	StartTS  *int64
}

func newEventFeed(client *Client, source EventSource, opts *feedOptions) (*EventFeed, error) {
	feed := &EventFeed{
		client: client,
		source: source,
		opts:   opts,
	}

	return feed, nil
}

func (ef *EventFeed) newFeedRequest() (*feedRequest, error) {
	req := feedRequest{
		apiRequest: apiRequest{
			ef.client.ctx,
			ef.client.headers,
		},
		Source: ef.source,
		Cursor: ef.lastCursor,
	}
	if ef.opts.StartTS != nil {
		req.StartTS = *ef.opts.StartTS
	}
	if ef.opts.Cursor != nil {
		req.Cursor = *ef.opts.Cursor
	}
	if ef.opts.PageSize != nil {
		req.PageSize = *ef.opts.PageSize
	}

	return &req, nil
}

func (ef *EventFeed) open() error {
	req, err := ef.newFeedRequest()
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
	ef.opts = &feedOptions{}

	return nil
}
