package fauna

import (
	"encoding/json"
	"io"
)

// Event represents a streaming event.
//
// All events contain the [fauna.Event.Type] and [fauna.Event.Stats] fields.
//
// Events of type "add", "update", and "remove" will contain the
// [fauna.Event.Data] field with the event's data in it. Data events have their
// [fauna.Event.Error] field set to nil. Data events can be umarmshalled into a
// user-defined struct via the [fauna.Event.Unmarshal] method.
//
// Events of type "status" and "error" will have their [fauna.Event.Data] field
// set to nil. Error events contain the [fauna.Event.Error] field present with
// the underlying error information.
type Event struct {
	// Type is this event's type.
	Type string `json:"type"`

	// TxnTime is the transaction time that produce this event.
	TxnTime int64 `json:"txn_ts,omitempty"`

	// Data is the event's data. Data is set to nil if the Type field is set to
	// "status" or "error".
	Data any `json:"data,omitempty"`

	// Error contains error information when the event Type is set to "error".
	Error *ErrEvent `json:"error,omitempty"`

	// Stats contains the ops acquired to process the event.
	Stats Stats `json:"stats"`
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

	// Abort is the error's abort data, present if Code == "abort".
	Abort any `json:"abort"`
}

// Unmarshal will unmarshal the raw [fauna.ErrEvent.Abort] (if present) into the
// known type provided as `into`. `into` must be a pointer to a map or struct.
func (e *ErrEvent) Unmarshal(into any) error {
	return decodeInto(e.Abort, into)
}

// Subscription is a Fauna stream subscription.
//
// Events can be obtained by reading from the [fauna.Subscription.Events]
// channel. Note that the events channel emits a nil event on closing.
//
// If the subscription gets closed unexpectedly, its closing error can be
// retrieved via the [fauna.Subscription.Error] method.
//
// A stream subscription can be gracefully closed via the
// [fauna.Subscription.Close] method.
type Subscription struct {
	byteStream io.ReadCloser
	events     chan *Event
	error      error
	closed     bool
}

// Events return the subscription's events channel.
func (s *Subscription) Events() <-chan *Event { return s.events }

// Error returns the subscription's closing error, if any.
func (s *Subscription) Error() error { return s.error }

// Close gracefully closes the stream subscription.
func (s *Subscription) Close() (err error) {
	if !s.closed {
		s.closed = true
		err = s.byteStream.Close()
	}
	return
}

func (s *Subscription) consume() {
	defer close(s.events)
	decoder := json.NewDecoder(s.byteStream)

	for {
		event := &Event{}
		if err := decoder.Decode(event); err != nil {
			// NOTE: When closing the stream, a network error may occur as due
			// to its socket closing while the json decoder is blocked reading
			// it. Errors to close the socket are already emitted by the Close()
			// method, therefore, we don't want to propagate them here again.
			if !s.closed {
				s.error = err
			}
			break
		}
		if err := convertEvent(event); err != nil {
			s.error = err
			break
		}
		s.events <- event
	}
}

func convertEvent(event *Event) (err error) {
	if event.Data != nil {
		if event.Data, err = convert(false, event.Data); err != nil {
			return
		}
	}
	if event.Error != nil && event.Error.Abort != nil {
		if event.Error.Abort, err = convert(false, event.Error.Abort); err != nil {
			return
		}
	}
	return
}
