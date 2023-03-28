package fauna

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetErrFauna(t *testing.T) {
	type args struct {
		httpStatus   int
		serviceError *ErrFauna
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
				serviceError: &ErrFauna{Code: "invalid_query", Message: ""},
				errType:      &ErrQueryCheck{},
			},
			wantErr: true,
		},
		{
			name: "Query runtime error",
			args: args{
				httpStatus:   http.StatusBadRequest,
				serviceError: &ErrFauna{Code: "", Message: ""},
				errType:      &ErrQueryRuntime{},
			},
			wantErr: true,
		},
		{
			name: "Unauthorized",
			args: args{
				httpStatus:   http.StatusUnauthorized,
				serviceError: &ErrFauna{Code: "", Message: ""},
				errType:      &ErrAuthentication{},
			},
			wantErr: true,
		},
		{
			name: "Access not granted",
			args: args{
				httpStatus:   http.StatusForbidden,
				serviceError: &ErrFauna{Code: "", Message: ""},
				errType:      &ErrAuthorization{},
			},
			wantErr: true,
		},
		{
			name: "Too many requests",
			args: args{
				httpStatus:   http.StatusTooManyRequests,
				serviceError: &ErrFauna{Code: "", Message: ""},
				errType:      &ErrThrottling{},
			},
			wantErr: true,
		},
		{
			name: "Query timeout",
			args: args{
				httpStatus:   440,
				serviceError: &ErrFauna{Code: "", Message: ""},
				errType:      &ErrQueryTimeout{},
			},
			wantErr: true,
		},
		{
			name: "Internal error",
			args: args{
				httpStatus:   http.StatusInternalServerError,
				serviceError: &ErrFauna{Code: "", Message: ""},
				errType:      &ErrServiceInternal{},
			},
			wantErr: true,
		},
		{
			name: "Service timeout",
			args: args{
				httpStatus:   http.StatusServiceUnavailable,
				serviceError: &ErrFauna{Code: "", Message: ""},
				errType:      &ErrServiceTimeout{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := &queryResponse{Error: tt.args.serviceError, Summary: ""}
			err := getErrFauna(tt.args.httpStatus, res)
			if tt.wantErr {
				assert.ErrorAs(t, err, &tt.args.errType)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
