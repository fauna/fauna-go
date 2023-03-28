package fauna

import (
	"net/http"
)

const httpStatusQueryTimeout = 440

var queryCheckFailureCodes = map[string]struct{}{
	"invalid_function_definition": {},
	"invalid_identifier":          {},
	"invalid_query":               {},
	"invalid_syntax":              {},
	"invalid_type":                {},
}

// A ErrFauna is the base of all errors and provides the underlying `code`,
// `message`, and any [fauna.QueryInfo].
type ErrFauna struct {
	*QueryInfo
	Code    string `json:"code"`
	Message string `json:"message"`
}

// provides the underlying error message.
func (e ErrFauna) Error() string {
	return e.Message
}

// A ErrQueryRuntime is returned when the query fails due to a runtime error.
// The `code` field will vary based on the specific error cause.
type ErrQueryRuntime struct {
	*ErrFauna
}

// A ErrQueryCheck is returned when the query fails one or more validation checks.
type ErrQueryCheck struct {
	*ErrFauna
}

// A ErrQueryTimeout is returned when the client specified timeout was
// exceeded, but the timeout was set lower than the query's expected
// processing time. This response is distinguished from [fauna.ServiceTimeoutError]
// by the fact that a [fauna.QueryTimeoutError] response is considered a
// successful response for the purpose of determining the service's availability.
type ErrQueryTimeout struct {
	*ErrFauna
}

// An ErrAuthentication is returned when Fauna is unable to authenticate
// the request due to an invalid or missing authentication token.
type ErrAuthentication struct {
	*ErrFauna
}

// An ErrAuthorization is returned when a query attempts to access data the
// secret is not allowed to access.
type ErrAuthorization struct {
	*ErrFauna
}

// A ErrThrottling is returned when the query exceeded some capacity limit.
type ErrThrottling struct {
	*ErrFauna
}

// A ErrServiceInternal is returned when an unexpected error occurs.
type ErrServiceInternal struct {
	*ErrFauna
}

// A ErrServiceTimeout is returned when an unexpected timeout occurs.
type ErrServiceTimeout struct {
	*ErrFauna
}

// A ErrNetwork is returned when an unknown error is encounted when attempting
// to send a request to Fauna.
type ErrNetwork error

func getErrFauna(httpStatus int, res *queryResponse) error {
	if res.Error != nil {
		res.Error.QueryInfo = newQueryInfo(res)
	}

	switch httpStatus {
	case http.StatusBadRequest:
		if res.Error == nil {
			err := &ErrQueryRuntime{&ErrFauna{QueryInfo: newQueryInfo(res), Code: "", Message: ""}}
			err.Message += "\n" + res.Summary
			return err
		}

		if _, found := queryCheckFailureCodes[res.Error.Code]; found {
			err := &ErrQueryCheck{res.Error}
			err.Message += "\n" + res.Summary
			return err

		} else {
			err := &ErrQueryRuntime{res.Error}
			err.Message += "\n" + res.Summary
			return err
		}

	case http.StatusUnauthorized:
		return &ErrAuthentication{res.Error}
	case http.StatusForbidden:
		return &ErrAuthorization{res.Error}
	case http.StatusTooManyRequests:
		return &ErrThrottling{res.Error}
	case httpStatusQueryTimeout:
		return &ErrQueryTimeout{res.Error}
	case http.StatusInternalServerError:
		return &ErrServiceInternal{res.Error}
	case http.StatusServiceUnavailable:
		return &ErrServiceTimeout{res.Error}
	}

	return nil
}
