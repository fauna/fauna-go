package fauna

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

func GetServiceError(httpStatus int, e *ServiceError) error {
	switch httpStatus {
	case 400:
		if _, found := queryCheckFailureCodes[e.Code]; found {
			return NewQueryCheckError(e)
		} else {
			return NewQueryRuntimeError(e)
		}
	case 401:
		return NewAuthenticationError(e)
	case 403:
		return NewAuthorizationError(e)
	case 429:
		return NewThrottlingError(e)
	case 440:
		return NewQueryTimeoutError(e)
	case 500:
		return NewServiceInternalError(e)
	case 503:
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

func NewQueryCheckError(e *ServiceError) QueryCheckError {
	return QueryCheckError{
		ServiceError: *e,
	}
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
