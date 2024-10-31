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

	lastCursor *string
	pageSize   *int
	startTs    *int64
}

func newEventFeed(client *Client, source EventSource, args *FeedArgs) (*EventFeed, error) {
	feed := &EventFeed{
		client: client,
		source: source,
	}

	if args != nil {
		if args.StartTs != nil && args.Cursor != nil {
			return nil, fmt.Errorf("StartTs and Cursor cannot be used simultaneously")
		}
		if args.Cursor != nil {
			feed.lastCursor = args.Cursor
		}

		if args.StartTs != nil {
			unixTime := args.StartTs.UnixMicro()
			feed.startTs = &unixTime
		}

		feed.pageSize = args.PageSize
	}

	if err := feed.open(); err != nil {
		return nil, err
	}

	return feed, nil
}

func (ef *EventFeed) open() error {
	req := feedRequest{
		apiRequest: apiRequest{
			ef.client.ctx,
			ef.client.headers,
		},
		Source:   ef.source,
		Cursor:   ef.lastCursor,
		PageSize: ef.pageSize,
		StartTS:  ef.startTs,
	}

	byteStream, err := req.do(ef.client)
	if err != nil {
		return err
	}

	ef.decoder = json.NewDecoder(byteStream)

	return nil
}

// FeedPage represents the response from the EventFeed.Events
type FeedPage struct {
	Events  []Event `json:"events"`
	Cursor  string  `json:"cursor"`
	HasNext bool    `json:"has_next"`
	Stats   Stats   `json:"stats"`
}

// Next retrieves the next FeedPage from the EventFeed
func (ef *EventFeed) Next(page *FeedPage) error {
	if err := ef.open(); err != nil {
		return err
	}

	if err := ef.decoder.Decode(&page); err != nil {
		return err
	}

	ef.lastCursor = &page.Cursor

	return nil
}
