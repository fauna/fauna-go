package fauna

import (
	"encoding/json"
	"errors"
	"io"
	"net"
)

// EventType represents a Fauna's event type.
type EventType string

const (
	// AddEvent happens when a new value is added to the stream's watched set.
	AddEvent EventType = "add"
	// UpdateEvent happens when a value in the stream's watched set changes.
	UpdateEvent EventType = "update"
	// RemoveEvent happens when a value in the stream's watched set is removed.
	RemoveEvent EventType = "remove"
	// StatusEvent happens periodically and communicates the stream's latest
	// transaction time as well as ops acquired during its idle period.
	StatusEvent EventType = "status"
)

// Event represents a streaming event.
//
// EventStream of type [fauna.StatusEvent] have its [fauna.Event.Data] field set to
// nil. Other event's [fauna.Data] can be unmarshaled via the
// [fauna.Event.Unmarshal] method.
type Event struct {
	// Type is this event's type.
	Type EventType
	// TxnTime is the transaction time that produce this event.
	TxnTime int64
	// Cursor is the event's cursor, used for resuming streams after crashes.
	Cursor string
	// Data is the event's data.
	Data any
	// Stats contains the ops acquired to process the event.
	Stats Stats
}

// Unmarshal will unmarshal the raw [fauna.Event.Data] (if present) into the
// known type provided as `into`. `into` must be a pointer to a map or struct.
func (e *Event) Unmarshal(into any) error {
	return decodeInto(e.Data, into)
}

// ErrEvent contains error information present in error events.
//
// Error events with "abort" code contain its aborting value present in the
// [fauna.ErrEvent.Abort]. The aborting values can be unmarshaled with the
// [fauna.ErrEvent.Unmarshal] method.
type ErrEvent struct {
	// Code is the error's code.
	Code string `json:"code"`

	// Message is the error's message.
	Message string `json:"message"`

	// Abort is the error's abort data, present if [fauna.ErrEvent.Code] is
	// equals to "abort".
	Abort any `json:"abort,omitempty"`
}

// Error provides the underlying error message.
func (e *ErrEvent) Error() string {
	return e.Message
}

// Unmarshal will unmarshal the raw [fauna.ErrEvent.Abort] (if present) into the
// known type provided as `into`. `into` must be a pointer to a map or struct.
func (e *ErrEvent) Unmarshal(into any) error {
	return decodeInto(e.Abort, into)
}

// EventStream is an iterator of Fauna events.
//
// The next available event can be obtained by calling the
// [fauna.EventStream.Next] method. Note this method blocks until the next
// event is available or the events iterator is closed via the
// [fauna.EventStream.Close] method.
//
// The events iterator wraps an [http.Response.Body] reader. As per Go's current
// [http.Response] implementation, environments using HTTP/1.x may not reuse its
// TCP connections for the duration of its "keep-alive" time if response body is
// not read to completion and closed. By default, Fauna's region groups use the
// HTTP/2.x protocol where this restriction doesn't apply. However, if connecting
// to Fauna via an HTTP/1.x proxy, be aware of the events iterator closing time.
type EventStream struct {
	client     *Client
	stream     EventSource
	byteStream io.ReadCloser
	decoder    *json.Decoder
	lastCursor string
	closed     bool
}

func subscribe(client *Client, stream EventSource, opts ...StreamOptFn) (*EventStream, error) {
	events := &EventStream{client: client, stream: stream}
	if err := events.reconnect(opts...); err != nil {
		return nil, err
	}
	return events, nil
}

func (es *EventStream) reconnect(opts ...StreamOptFn) error {
	req := streamRequest{
		apiRequest: apiRequest{
			es.client.ctx,
			es.client.headers,
		},
		Stream: es.stream,
		Cursor: es.lastCursor,
	}

	for _, streamOptionFn := range opts {
		streamOptionFn(&req)
	}

	byteStream, err := req.do(es.client)
	if err != nil {
		return err
	}

	es.byteStream = byteStream
	es.decoder = json.NewDecoder(byteStream)
	return nil
}

// Close gracefully closes the events iterator. See [fauna.EventStream] for details.
func (es *EventStream) Close() (err error) {
	if !es.closed {
		es.closed = true
		err = es.byteStream.Close()
	}
	return
}

type rawEvent = struct {
	Type    EventType `json:"type"`
	TxnTime int64     `json:"txn_ts"`
	Cursor  string    `json:"cursor"`
	Data    any       `json:"data,omitempty"`
	Error   *ErrEvent `json:"error,omitempty"`
	Stats   Stats     `json:"stats"`
}

// Next blocks until the next event is available.
//
// Note that network errors of type [fauna.ErrEvent] are considered fatal and
// close the underlying stream. Calling next after an error event occurs will
// return an error.
func (es *EventStream) Next(event *Event) (err error) {
	raw := rawEvent{}
	if err = es.decoder.Decode(&raw); err == nil {
		es.onNextEvent(&raw)
		err = convertRawEvent(&raw, event)
		var errEvent *ErrEvent
		if errors.As(err, &errEvent) {
			_ = es.Close() // no more events are coming
		}
	} else if !es.closed {
		// NOTE: This code tries to resume streams on network and IO errors. It
		// presumes that if the service is unavailable, the reconnect call will
		// fail. Automatic retries and backoff mechanisms are implemented at the
		// Client level.
		var netError net.Error
		if errors.As(err, &netError) || err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
			if err = es.reconnect(); err == nil {
				err = es.Next(event)
			}
		}
	}
	return
}

func (es *EventStream) onNextEvent(event *rawEvent) {
	es.client.lastTxnTime.sync(event.TxnTime)
	es.lastCursor = event.Cursor
}

func convertRawEvent(raw *rawEvent, event *Event) (err error) {
	if raw.Error != nil {
		if raw.Error.Abort != nil {
			if raw.Error.Abort, err = convert(false, raw.Error.Abort); err != nil {
				return
			}
		}
		err = raw.Error
	} else {
		if raw.Data != nil {
			if raw.Data, err = convert(false, raw.Data); err != nil {
				return
			}
		}
		event.Type = raw.Type
		event.TxnTime = raw.TxnTime
		event.Cursor = raw.Cursor
		event.Data = raw.Data
		event.Stats = raw.Stats
	}
	return
}
