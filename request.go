package fauna

import "net/http"

type Request struct {
	raw       *http.Request
	Query     string            `json:"query"`
	Arguments map[string]string `json:"arguments"`
	Typecheck bool              `json:"typecheck"`
}

func NewRequest(query string) *Request {
	return &Request{
		Query:     query,
		Arguments: make(map[string]string),
		Typecheck: true,
	}
}
