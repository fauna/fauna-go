package fauna

import (
	"net/http"
	"time"
)

type Response[T any] struct {
	raw        *http.Response
	err        error
	Data       T         `json:"data"`
	TxnTime    time.Time `json:"txn_time"`
	StaticType string    `json:"static_type"`
	Summary    string    `json:"summary"`
}

func NewResponse[T any]() *Response[T] {
	return &Response[T]{
		TxnTime: time.Time{},
	}
}

func ErrorResponse[T any](err error) *Response[T] {
	return &Response[T]{
		err: err,
	}
}
