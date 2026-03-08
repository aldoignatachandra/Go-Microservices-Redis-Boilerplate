package validator_test

import (
	"testing"

	"github.com/ignata/go-microservices-boilerplate/pkg/validator"
)

func TestPasswordValidator(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		err      error
	}{
		{
			name:     "valid password",
			password: "Password1",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "valid with special character",
			password: "Password1!",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "valid with multiple special characters",
			password: "Pass@word1",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "too short",
			password: "Pass1",
			wantErr:  true,
			err:      validator.ErrPasswordTooShort,
		},
		{
			name:     "exactly 8 characters - valid",
			password: "Password1",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "no uppercase",
			password: "password1",
			wantErr:  true,
			err:      validator.ErrPasswordNoUppercase,
		},
		{
			name:     "only uppercase - no lowercase",
			password: "PASSWORD1",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "no number",
			password: "Password",
			wantErr:  true,
			err:      validator.ErrPasswordNoNumber,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
			err:      validator.ErrPasswordTooShort,
		},
		{
			name:     "forbidden single quote",
			password: "Password1'",
			wantErr:  true,
			err:      validator.ErrPasswordForbiddenChar,
		},
		{
			name:     "forbidden double quote",
			password: `Password1"`,
			wantErr:  true,
			err:      validator.ErrPasswordForbiddenChar,
		},
		{
			name:     "forbidden backtick",
			password: "Password1`",
			wantErr:  true,
			err:      validator.ErrPasswordForbiddenChar,
		},
		{
			name:     "forbidden backslash",
			password: "Password1\\",
			wantErr:  true,
			err:      validator.ErrPasswordForbiddenChar,
		},
		{
			name:     "forbidden forward slash",
			password: "Password1/",
			wantErr:  true,
			err:      validator.ErrPasswordForbiddenChar,
		},
		{
			name:     "maximum length password",
			password: "P@ssw0rd1234567890!",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "with allowed special chars - underscore",
			password: "Password1_",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "with allowed special chars - hyphen",
			password: "Password1-",
			wantErr:  false,
			err:      nil,
		},
		{
			name:     "with allowed special chars - bracket",
			password: "Password1[]",
			wantErr:  false,
			err:      nil,
		},
	}

	v := validator.NewPasswordValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.password)
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

func TestPasswordValidator_IsValid(t *testing.T) {
	v := validator.NewPasswordValidator()

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{
			name:     "valid password",
			password: "Password1",
			want:     true,
		},
		{
			name:     "invalid password - too short",
			password: "Pass1",
			want:     false,
		},
		{
			name:     "invalid password - no uppercase",
			password: "password1",
			want:     false,
		},
		{
			name:     "invalid password - no number",
			password: "Password",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.IsValid(tt.password)
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid",
			password: "Password1",
			wantErr:  false,
		},
		{
			name:     "invalid",
			password: "pass",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{
			name:     "valid",
			password: "Password1",
			want:     true,
		},
		{
			name:     "invalid",
			password: "pass",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.IsValidPassword(tt.password)
			if got != tt.want {
				t.Errorf("IsValidPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}
