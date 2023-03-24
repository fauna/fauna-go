package fauna

import (
	"net/http"
)

const HttpStatusQueryTimeout = 440

var queryCheckFailureCodes = map[string]struct{}{
	"invalid_function_definition": {},
	"invalid_identifier":          {},
	"invalid_query":               {},
	"invalid_syntax":              {},
	"invalid_type":                {},
}

type ServiceError struct {
	*QueryInfo
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e ServiceError) Error() string {
	return e.Message
}

type QueryRuntimeError struct {
	*ServiceError
}

type QueryCheckError struct {
	*ServiceError
}

type QueryTimeoutError struct {
	*ServiceError
}

type AuthenticationError struct {
	*ServiceError
}

type AuthorizationError struct {
	*ServiceError
}

type ThrottlingError struct {
	*ServiceError
}

type ServiceInternalError struct {
	*ServiceError
}

type ServiceTimeoutError struct {
	*ServiceError
}

type NetworkError error

// GetServiceError return a typed error based on the http status code
// and ServiceError response from fauna
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
	case HttpStatusQueryTimeout:
		return &QueryTimeoutError{res.Error}
	case http.StatusInternalServerError:
		return &ServiceInternalError{res.Error}
	case http.StatusServiceUnavailable:
		return &ServiceTimeoutError{res.Error}
	}

	return nil
}
