package fauna

import (
	"context"
	"encoding/json"
	"io"
	"time"
)

type StreamOptFn func(*streamRequest)

// TODO: move to client.go?
func StreamStartTime(time *time.Time) StreamOptFn {
	return func(req *streamRequest) {
		// TODO
		// req.StartTime = time
	}
}

// TODO: move to client.go?
func (c *Client) StreamToken(token string, opts ...StreamOptFn) (*Stream, error) {
	req := &streamRequest{
		apiRequest: apiRequest{
			Context: c.ctx,
			Headers: c.headers,
		},
		Token: token,
		// StartTime: nil,
	}

	for _, streamOptFn := range opts {
		streamOptFn(req)
	}

	return req.do(c)
}

type Stream struct {
	context context.Context
	bytes   io.ReadCloser
	events  chan *StreamEvent
	err     error
}

func (s *Stream) Error() error {
	return s.err
}

func (s *Stream) Start() chan *StreamEvent {
	if s.events == nil {
		s.events = make(chan *StreamEvent)

		go func() { // start decoding events
			defer close(s.events)
			decoder := json.NewDecoder(s.bytes)
			for {
				var (
					obj map[string]any
					err error
				)
				if err = decoder.Decode(&obj); err == nil {
					if data, ok := obj["data"]; ok {
						if data, err = convert(false, data); err == nil {
							obj["data"] = data
						}
					}
				}
				if err == nil {
					s.events <- &StreamEvent{obj}
				} else {
					if s.err == nil {
						s.err = err
					}
					break
				}
			}
		}()

		// FIXME: if closing on the go routine above, this one will remain stuck forever.
		go func() { // close the events channel on ctx completion
			// TODO: Is closing gonna leave the connection unusable?
			// How can we drain lingering bytes here?
			defer s.bytes.Close()
			defer close(s.events)

			<-s.context.Done()
			if s.err == nil {
				s.err = s.context.Err()
			}
		}()
	}

	return s.events
}

type StreamEvent struct {
	body any
}
