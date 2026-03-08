// Package validator provides validation utilities.
package validator

import (
	"errors"
	"regexp"
)

var (
	// ErrPasswordTooShort is returned when password is too short.
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	// ErrPasswordNoUppercase is returned when password lacks uppercase.
	ErrPasswordNoUppercase = errors.New("password must contain at least one uppercase letter")
	// ErrPasswordNoNumber is returned when password lacks a number.
	ErrPasswordNoNumber = errors.New("password must contain at least one number")
	// ErrPasswordForbiddenChar is returned when password contains forbidden characters.
	ErrPasswordForbiddenChar = errors.New("password contains forbidden characters")
)

var (
	passwordMinLength   = 8
	passwordUppercaseRE = regexp.MustCompile(`[A-Z]`)
	passwordNumberRE    = regexp.MustCompile(`\d`)
)

// PasswordValidator validates passwords.
type PasswordValidator struct {
	minLength        int
	requireUppercase bool
	requireNumber    bool
}

// NewPasswordValidator creates a new PasswordValidator.
func NewPasswordValidator() *PasswordValidator {
	return &PasswordValidator{
		minLength:        passwordMinLength,
		requireUppercase: true,
		requireNumber:    true,
	}
}

// Validate validates a password.
func (v *PasswordValidator) Validate(password string) error {
	if len(password) < v.minLength {
		return ErrPasswordTooShort
	}
	if v.requireUppercase && !passwordUppercaseRE.MatchString(password) {
		return ErrPasswordNoUppercase
	}
	if v.requireNumber && !passwordNumberRE.MatchString(password) {
		return ErrPasswordNoNumber
	}
	for _, c := range password {
		if isForbiddenPasswordChar(c) {
			return ErrPasswordForbiddenChar
		}
	}
	return nil
}

func isForbiddenPasswordChar(c rune) bool {
	forbidden := []rune{'\'', '"', '`', '\\', '/'}
	for _, f := range forbidden {
		if c == f {
			return true
		}
	}
	return false
}

// IsValid checks if a password is valid.
func (v *PasswordValidator) IsValid(password string) bool {
	return v.Validate(password) == nil
}

// ValidatePassword validates a password using default rules.
func ValidatePassword(password string) error {
	return NewPasswordValidator().Validate(password)
}

// IsValidPassword checks if a password is valid using default rules.
func IsValidPassword(password string) bool {
	return ValidatePassword(password) == nil
}
