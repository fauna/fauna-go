package fauna

import (
	"encoding/json"
	"io"
)

// EventType represents a Fauna's event type.
type EventType string

const (
	// AddEvent happens when a new value is added to the stream's watched set.
	AddEvent EventType = "add"
	// UpdateEvent happens when a value in the stream's watched set changes.
	UpdateEvent EventType = "update"
	// Remove event happens when a value in the stream's watched set is removed.
	RemoveEvent EventType = "remove"
	// StatusEvent happens periodically and comunicates the stream's latest
	// transacion time as well as ops aquired during its idle period.
	StatusEvent EventType = "status"
)

// Event represents a streaming event.
//
// Events of type [fauna.StatusEvent] have its [fauna.Event.Data] field set to
// nil. Other event's [fauna.Data] can be unmarshalled via the
// [fauna.Event.Unmarshal] method.
type Event struct {
	// Type is this event's type.
	Type EventType
	// TxnTime is the transaction time that produce this event.
	TxnTime int64
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
// [fauan.ErrEvent.Abort]. The aborting values can be unmarshalled with the
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

// Events is an iterator of Fauna events.
//
// The next available event can be obtained by calling the
// [fauna.Subscription.Next] method. Note this method blocks until the next
// event is available or until the events iterator is closed via the
// [fauna.Events.Close] method.
type Events struct {
	byteStream io.ReadCloser
	decoder    *json.Decoder
}

func newEvents(byteStream io.ReadCloser) *Events {
	return &Events{
		byteStream: byteStream,
		decoder:    json.NewDecoder(byteStream),
	}
}

// Close gracefully closes the stream subscription.
func (es *Events) Close() (err error) {
	// XXX: Is there a way to make sure there are no bytes left on the stream
	// after closing it? According to go's docs, the underlying connection will
	// remain unusable for the duration of its idle time if there are bytes left
	// in its read buffer.
	return es.byteStream.Close()
}

type rawEvent = struct {
	Type    EventType `json:"type"`
	TxnTime int64     `json:"txn_ts"`
	Data    any       `json:"data,omitempty"`
	Error   *ErrEvent `json:"error,omitempty"`
	Stats   Stats     `json:"stats"`
}

// Next blocks until the next event is available.
//
// Note that network errors of type [fauna.ErrEvent] are considered fatal and
// close the underlying stream. Calling next after an error event occurs will
// return an error.
func (es *Events) Next() (event *Event, err error) {
	raw := rawEvent{}
	if err = es.decoder.Decode(&raw); err == nil {
		event, err = convertRawEvent(&raw)
		if _, ok := err.(*ErrEvent); ok {
			es.Close() // no more events are comming
		}
	}
	return
}

func convertRawEvent(raw *rawEvent) (event *Event, err error) {
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
		event = &Event{
			Type:    raw.Type,
			TxnTime: raw.TxnTime,
			Data:    raw.Data,
			Stats:   raw.Stats,
		}
	}
	return
}
