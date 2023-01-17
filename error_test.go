package fauna_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/fauna/fauna-go"
)

func TestGetServiceError(t *testing.T) {
	type args struct {
		httpStatus int
		e          *fauna.ServiceError
		errType    error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "No error",
			args: args{
				httpStatus: 200,
				e:          nil,
				errType:    nil,
			},
			wantErr: false,
		},
		{
			name: "Query error",
			args: args{
				httpStatus: http.StatusBadRequest,
				e:          &fauna.ServiceError{Code: "invalid_identifier", Message: ""},
				errType:    fauna.QueryCheckError{},
			},
			wantErr: false,
		},
		{
			name: "Query runtime error",
			args: args{
				httpStatus: http.StatusBadRequest,
				e:          &fauna.ServiceError{Code: "", Message: ""},
				errType:    fauna.QueryRuntimeError{},
			},
			wantErr: false,
		},
		{
			name: "Unauthorized",
			args: args{
				httpStatus: http.StatusUnauthorized,
				e:          &fauna.ServiceError{Code: "", Message: ""},
				errType:    fauna.AuthenticationError{},
			},
			wantErr: true,
		},
		{
			name: "Access not granted",
			args: args{
				httpStatus: http.StatusForbidden,
				e:          &fauna.ServiceError{Code: "", Message: ""},
				errType:    fauna.AuthorizationError{},
			},
			wantErr: true,
		},
		{
			name: "Too many requests",
			args: args{
				httpStatus: http.StatusTooManyRequests,
				e:          &fauna.ServiceError{Code: "", Message: ""},
				errType:    fauna.ThrottlingError{},
			},
			wantErr: true,
		},
		{
			name: "Query timeout",
			args: args{
				httpStatus: 440,
				e:          &fauna.ServiceError{Code: "", Message: ""},
				errType:    fauna.QueryTimeoutError{},
			},
			wantErr: true,
		},
		{
			name: "Internal error",
			args: args{
				httpStatus: http.StatusInternalServerError,
				e:          &fauna.ServiceError{Code: "", Message: ""},
				errType:    fauna.ServiceInternalError{},
			},
			wantErr: true,
		},
		{
			name: "Service timeout",
			args: args{
				httpStatus: http.StatusServiceUnavailable,
				e:          &fauna.ServiceError{Code: "", Message: ""},
				errType:    fauna.ServiceTimeoutError{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fauna.GetServiceError(tt.args.httpStatus, tt.args.e)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServiceError() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.wantErr && !errors.Is(err, tt.args.errType) {
				t.Errorf("error [%v] wanted [%v]", err, tt.args.errType)
			}
		})
	}
}
