package fauna

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetServiceError(t *testing.T) {
	type args struct {
		httpStatus   int
		serviceError *ServiceError
		errType      error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "No error",
			args: args{
				httpStatus:   200,
				serviceError: nil,
				errType:      nil,
			},
			wantErr: false,
		},
		{
			name: "Query check error",
			args: args{
				httpStatus:   http.StatusBadRequest,
				serviceError: &ServiceError{Code: "invalid_query", Message: ""},
				errType:      &QueryCheckError{},
			},
			wantErr: true,
		},
		{
			name: "Query runtime error",
			args: args{
				httpStatus:   http.StatusBadRequest,
				serviceError: &ServiceError{Code: "", Message: ""},
				errType:      &QueryRuntimeError{},
			},
			wantErr: true,
		},
		{
			name: "Unauthorized",
			args: args{
				httpStatus:   http.StatusUnauthorized,
				serviceError: &ServiceError{Code: "", Message: ""},
				errType:      &AuthenticationError{},
			},
			wantErr: true,
		},
		{
			name: "Access not granted",
			args: args{
				httpStatus:   http.StatusForbidden,
				serviceError: &ServiceError{Code: "", Message: ""},
				errType:      &AuthorizationError{},
			},
			wantErr: true,
		},
		{
			name: "Too many requests",
			args: args{
				httpStatus:   http.StatusTooManyRequests,
				serviceError: &ServiceError{Code: "", Message: ""},
				errType:      &ThrottlingError{},
			},
			wantErr: true,
		},
		{
			name: "Query timeout",
			args: args{
				httpStatus:   440,
				serviceError: &ServiceError{Code: "", Message: ""},
				errType:      &QueryTimeoutError{},
			},
			wantErr: true,
		},
		{
			name: "Internal error",
			args: args{
				httpStatus:   http.StatusInternalServerError,
				serviceError: &ServiceError{Code: "", Message: ""},
				errType:      &ServiceInternalError{},
			},
			wantErr: true,
		},
		{
			name: "Service timeout",
			args: args{
				httpStatus:   http.StatusServiceUnavailable,
				serviceError: &ServiceError{Code: "", Message: ""},
				errType:      &ServiceTimeoutError{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := &queryResponse{Error: tt.args.serviceError, Summary: ""}
			err := getServiceError(tt.args.httpStatus, res)
			if tt.wantErr {
				assert.ErrorAs(t, err, &tt.args.errType)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
