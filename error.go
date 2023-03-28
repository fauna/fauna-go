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

// A ServiceError is the base of all errors and provides the underlying `code`,
// `message`, and any [fauna.QueryInfo].
type ServiceError struct {
	*QueryInfo
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error provides the underlying error message.
func (e ServiceError) Error() string {
	return e.Message
}

// A QueryRuntimeError is returned when the query fails due to a runtime error.
// The `code` field will vary based on the specific error cause.
type QueryRuntimeError struct {
	*ServiceError
}

// A QueryCheckError is returned when the query fails one or more validation checks.
type QueryCheckError struct {
	*ServiceError
}

// A QueryTimeoutError is returned when the client specified timeout was
// exceeded, but the timeout was set lower than the query's expected
// processing time. This response is distinguished from [fauna.ServiceTimeoutError]
// by the fact that a [fauna.QueryTimeoutError] response is considered a
// successful response for the purpose of determining the service's availability.
type QueryTimeoutError struct {
	*ServiceError
}

// An AuthenticationError is returned when Fauna is unable to authenticate
// the request due to an invalid or missing authentication token.
type AuthenticationError struct {
	*ServiceError
}

// An AuthorizationError is returned when a query attempts to access data the
// secret is not allowed to access.
type AuthorizationError struct {
	*ServiceError
}

// A ThrottlingError is returned when the query exceeded some capacity limit.
type ThrottlingError struct {
	*ServiceError
}

// A ServiceInternalError is returned when an unexpected error occurs.
type ServiceInternalError struct {
	*ServiceError
}

// A ServiceTimeoutError is returned when an unexpected timeout occurs.
type ServiceTimeoutError struct {
	*ServiceError
}

// A NetworkError is returned when an unknown error is encounted when attempting
// to send a request to Fauna.
type NetworkError error

func getServiceError(httpStatus int, res *queryResponse) error {
	if res.Error != nil {
		res.Error.QueryInfo = newQueryInfo(res)
	}

	switch httpStatus {
	case http.StatusBadRequest:
		if res.Error == nil {
			err := &QueryRuntimeError{&ServiceError{QueryInfo: newQueryInfo(res), Code: "", Message: ""}}
			err.Message += "\n" + res.Summary
			return err
		}

		if _, found := queryCheckFailureCodes[res.Error.Code]; found {
			err := &QueryCheckError{res.Error}
			err.Message += "\n" + res.Summary
			return err

		} else {
			err := &QueryRuntimeError{res.Error}
			err.Message += "\n" + res.Summary
			return err
		}

	case http.StatusUnauthorized:
		return &AuthenticationError{res.Error}
	case http.StatusForbidden:
		return &AuthorizationError{res.Error}
	case http.StatusTooManyRequests:
		return &ThrottlingError{res.Error}
	case httpStatusQueryTimeout:
		return &QueryTimeoutError{res.Error}
	case http.StatusInternalServerError:
		return &ServiceInternalError{res.Error}
	case http.StatusServiceUnavailable:
		return &ServiceTimeoutError{res.Error}
	}

	return nil
}
