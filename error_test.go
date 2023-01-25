package fauna_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/fauna/fauna-go"
)

func TestGetServiceError(t *testing.T) {
	type args struct {
		httpStatus   int
		serviceError *fauna.ServiceError
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
				serviceError: &fauna.ServiceError{Code: "invalid_query", Message: ""},
				errType:      fauna.QueryCheckError{},
			},
			wantErr: true,
		},
		{
			name: "Query runtime error",
			args: args{
				httpStatus:   http.StatusBadRequest,
				serviceError: &fauna.ServiceError{Code: "", Message: ""},
				errType:      fauna.QueryRuntimeError{},
			},
			wantErr: true,
		},
		{
			name: "Unauthorized",
			args: args{
				httpStatus:   http.StatusUnauthorized,
				serviceError: &fauna.ServiceError{Code: "", Message: ""},
				errType:      fauna.AuthenticationError{},
			},
			wantErr: true,
		},
		{
			name: "Access not granted",
			args: args{
				httpStatus:   http.StatusForbidden,
				serviceError: &fauna.ServiceError{Code: "", Message: ""},
				errType:      fauna.AuthorizationError{},
			},
			wantErr: true,
		},
		{
			name: "Too many requests",
			args: args{
				httpStatus:   http.StatusTooManyRequests,
				serviceError: &fauna.ServiceError{Code: "", Message: ""},
				errType:      fauna.ThrottlingError{},
			},
			wantErr: true,
		},
		{
			name: "Query timeout",
			args: args{
				httpStatus:   440,
				serviceError: &fauna.ServiceError{Code: "", Message: ""},
				errType:      fauna.QueryTimeoutError{},
			},
			wantErr: true,
		},
		{
			name: "Internal error",
			args: args{
				httpStatus:   http.StatusInternalServerError,
				serviceError: &fauna.ServiceError{Code: "", Message: ""},
				errType:      fauna.ServiceInternalError{},
			},
			wantErr: true,
		},
		{
			name: "Service timeout",
			args: args{
				httpStatus:   http.StatusServiceUnavailable,
				serviceError: &fauna.ServiceError{Code: "", Message: ""},
				errType:      fauna.ServiceTimeoutError{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fauna.GetServiceError(tt.args.httpStatus, tt.args.serviceError, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServiceError() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.wantErr && fmt.Sprintf("%T", err) != fmt.Sprintf("%T", tt.args.errType) {
				t.Errorf("got [%T] wanted [%T]", err, tt.args.errType)
			}
		})
	}
}
