package actionlog

import (
	"errors"
	"fmt"
	"testing"

	"github.com/sambeau/kanbanzai/internal/service"
)

func TestClassifyError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error returns empty",
			err:  nil,
			want: "",
		},
		{
			name: "ErrNotFound",
			err:  service.ErrNotFound,
			want: ErrorNotFound,
		},
		{
			name: "ErrReferenceNotFound",
			err:  service.ErrReferenceNotFound,
			want: ErrorNotFound,
		},
		{
			name: "wrapped ErrNotFound",
			err:  fmt.Errorf("get entity: %w", service.ErrNotFound),
			want: ErrorNotFound,
		},
		{
			name: "ErrValidationFailed",
			err:  service.ErrValidationFailed,
			want: ErrorValidationError,
		},
		{
			name: "ErrInvalidTransition",
			err:  service.ErrInvalidTransition,
			want: ErrorValidationError,
		},
		{
			name: "ErrImmutableField",
			err:  service.ErrImmutableField,
			want: ErrorPreconditionError,
		},
		{
			name: "gate keyword",
			err:  errors.New("missing required document for gate"),
			want: ErrorGateFailure,
		},
		{
			name: "prerequisite keyword",
			err:  errors.New("prerequisite not satisfied"),
			want: ErrorGateFailure,
		},
		{
			name: "unknown error",
			err:  errors.New("something unexpected happened"),
			want: ErrorInternalError,
		},
		{
			name: "already exists",
			err:  errors.New("entity already exists"),
			want: ErrorPreconditionError,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyError(tc.err)
			if got != tc.want {
				t.Errorf("ClassifyError(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}
