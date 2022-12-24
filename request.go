package fauna

import (
	"net/http"
	"os"
	"strings"
)

type Request struct {
	Raw       *http.Request
	Query     string                 `json:"query"`
	Arguments map[string]interface{} `json:"arguments"`
	TypeCheck bool                   `json:"typecheck"`
}

func NewRequest(query string, arguments map[string]interface{}) *Request {
	typeCheck := true
	if val, found := os.LookupEnv(EnvFaunaTypeCheckEnabled); found {
		typeCheck = strings.ToLower(val) == "true"
	}

	return &Request{
		Query:     query,
		Arguments: arguments,
		TypeCheck: typeCheck,
	}
}
