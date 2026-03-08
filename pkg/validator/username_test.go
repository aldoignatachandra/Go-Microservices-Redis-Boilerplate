package validator_test

import (
	"testing"

	"github.com/ignata/go-microservices-boilerplate/pkg/validator"
)

func TestUsernameValidator(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
		err      error
	}{
		{
			name:     "valid username",
			username: "john_doe",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "valid with numbers",
			username: "john123",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "valid uppercase letters",
			username: "JohnDoe",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "valid mix of all",
			username: "John_Doe123",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "too short - 2 characters",
			username: "ab",
			wantErr:  true,
			err:      validator.ErrUsernameTooShort,
		},
		{
			name:     "exactly minimum length",
			username: "abc",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "exactly maximum length",
			username: "abcdefghijklmnopqrstuvwxyz1234567890abcdefghijklmn",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "too long - 51 characters",
			username: "abcdefghijklmnopqrstuvwxyz1234567890abcdefghijklmno",
			wantErr:  true,
			err:      validator.ErrUsernameTooLong,
		},
		{
			name:     "invalid - contains hyphen",
			username: "john-doe",
			wantErr:  true,
			err:      validator.ErrUsernameInvalid,
		},
		{
			name:     "invalid - contains space",
			username: "john doe",
			wantErr:  true,
			err:      validator.ErrUsernameInvalid,
		},
		{
			name:     "invalid - contains dot",
			username: "john.doe",
			wantErr:  true,
			err:      validator.ErrUsernameInvalid,
		},
		{
			name:     "invalid - contains special char",
			username: "john@doe",
			wantErr:  true,
			err:      validator.ErrUsernameInvalid,
		},
		{
			name:     "invalid - contains emoji",
			username: "john😀",
			wantErr:  true,
			err:      validator.ErrUsernameInvalid,
		},
		{
			name:     "empty username",
			username: "",
			wantErr:  true,
			err:      validator.ErrUsernameTooShort,
		},
		{
			name:     "single character",
			username: "a",
			wantErr:  true,
			err:      validator.ErrUsernameTooShort,
		},
		{
			name:     "only numbers",
			username: "12345",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "only underscores",
			username: "_____",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "starts with number",
			username: "123john",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "ends with underscore",
			username: "john_",
			wantErr:  false,
			err:      nil,
		},
	}

	v := validator.NewUsernameValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != tt.err {
				t.Errorf("Validate() error = %v, want %v", err, tt.err)
			}
		})
	}
}

func TestUsernameValidator_IsValid(t *testing.T) {
	v := validator.NewUsernameValidator()

	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{
			name:     "valid username",
			username: "john_doe",
			want:     true,
		},
		{
			name:     "invalid username - too short",
			username: "ab",
			want:     false,
		},
		{
			name:     "invalid username - too long",
			username: "this_is_a_very_long_username_that_exceeds_limithere",
			want:     false,
		},
		{
			name:     "invalid username - contains hyphen",
			username: "john-doe",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.IsValid(tt.username)
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{
			name:     "valid",
			username: "john_doe",
			wantErr:  false,
		},
		{
			name:     "invalid",
			username: "ab",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{
			name:     "valid",
			username: "john_doe",
			want:     true,
		},
		{
			name:     "invalid",
			username: "ab",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.IsValidUsername(tt.username)
			if got != tt.want {
				t.Errorf("IsValidUsername() = %v, want %v", got, tt.want)
			}
		})
	}
}
