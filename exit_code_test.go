package venom

import (
	"errors"
	"testing"
)

// codedError implements ExitCoder for tests.
type codedError struct {
	code int
	msg  string
}

func (e *codedError) Error() string  { return e.msg }
func (e *codedError) ErrorCode() int { return e.code }

// TestErrorExitCode verifies the error-to-exit-code mapping for rules
// CommandSucceeds, CommandFailsWithCode, and CommandFailsDefault in
// venom.allium.
func TestErrorExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "command_succeeds_nil_error",
			err:  nil,
			want: 0,
		},
		{
			name: "command_fails_default_plain_error",
			err:  errors.New("boom"),
			want: 1,
		},
		{
			name: "command_fails_with_code_uses_error_code",
			err:  &codedError{code: 42, msg: "validation failed"},
			want: 42,
		},
		{
			name: "command_fails_with_code_zero_is_honoured",
			err:  &codedError{code: 0, msg: "zero exit"},
			want: 0,
		},
		{
			name: "command_fails_with_code_negative_is_passed_through",
			err:  &codedError{code: -1, msg: "negative"},
			want: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errorExitCode(tt.err)
			if got != tt.want {
				t.Errorf("errorExitCode(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}
