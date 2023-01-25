package fauna

import (
	"net/http"
)

var queryCheckFailureCodes = map[string]struct{}{
	"invalid_function_definition": {},
	"invalid_identifier":          {},
	"invalid_query":               {},
	"invalid_syntax":              {},
	"invalid_type":                {},
}

type ServiceError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e ServiceError) Error() string {
	return e.Message
}

// GetServiceError return a typed error based on the http status code
// and ServiceError response from fauna
func GetServiceError(httpStatus int, e *ServiceError, summary string) error {
	switch httpStatus {
	case http.StatusBadRequest:
		if _, found := queryCheckFailureCodes[e.Code]; found {
			return NewQueryCheckError(e, summary)
		} else {
			return NewQueryRuntimeError(e)
		}
	case http.StatusUnauthorized:
		return NewAuthenticationError(e)
	case http.StatusForbidden:
		return NewAuthorizationError(e)
	case http.StatusTooManyRequests:
		return NewThrottlingError(e)
	case 440:
		return NewQueryTimeoutError(e)
	case http.StatusInternalServerError:
		return NewServiceInternalError(e)
	case http.StatusServiceUnavailable:
		return NewServiceTimeoutError(e)
	}

	return nil
}

type QueryRuntimeError struct {
	ServiceError
}

func NewQueryRuntimeError(e *ServiceError) QueryRuntimeError {
	return QueryRuntimeError{
		ServiceError: *e,
	}
}

type QueryCheckError struct {
	ServiceError
}

func NewQueryCheckError(e *ServiceError, summary string) QueryCheckError {
	q := QueryCheckError{
		ServiceError: *e,
	}
	q.Message += "\n" + summary

	return q
}

type QueryTimeoutError struct {
	ServiceError
}

func NewQueryTimeoutError(e *ServiceError) QueryTimeoutError {
	return QueryTimeoutError{
		ServiceError: *e,
	}
}

type AuthenticationError struct {
	ServiceError
}

func NewAuthenticationError(e *ServiceError) AuthenticationError {
	return AuthenticationError{
		ServiceError: *e,
	}
}

type AuthorizationError struct {
	ServiceError
}

func NewAuthorizationError(e *ServiceError) AuthorizationError {
	return AuthorizationError{
		ServiceError: *e,
	}
}

type ThrottlingError struct {
	ServiceError
}

func NewThrottlingError(e *ServiceError) ThrottlingError {
	return ThrottlingError{
		ServiceError: *e,
	}
}

type ServiceInternalError struct {
	ServiceError
}

func NewServiceInternalError(e *ServiceError) ServiceInternalError {
	return ServiceInternalError{
		ServiceError: *e,
	}
}

type ServiceTimeoutError struct {
	ServiceError
}

func NewServiceTimeoutError(e *ServiceError) ServiceTimeoutError {
	return ServiceTimeoutError{
		ServiceError: *e,
	}
}

type NetworkError error
