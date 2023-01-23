package fauna

import (
	"encoding/json"
	"net/http"
)

type Request struct {
	raw       *http.Request
	Query     string            `json:"query"`
	Arguments map[string]string `json:"arguments"`
	Typecheck bool              `json:"typecheck"`
}

func NewRequest(query string, arguments map[string]string) *Request {
	return &Request{
		Query:     query,
		Arguments: arguments,
		Typecheck: true,
	}
}

func (r *Request) String() string {
	j, e := json.Marshal(r)
	if e != nil {
		return ""
	}
	return string(j)
}
