// Package validator provides validation utilities.
package validator

import (
	"errors"
	"regexp"
)

var (
	// ErrUsernameTooShort is returned when username is too short.
	ErrUsernameTooShort = errors.New("username must be at least 3 characters")
	// ErrUsernameTooLong is returned when username is too long.
	ErrUsernameTooLong = errors.New("username must be at most 50 characters")
	// ErrUsernameInvalid is returned when username contains invalid characters.
	ErrUsernameInvalid = errors.New("username can only contain letters, numbers, and underscores")
)

var (
	usernameMinLength = 3
	usernameMaxLength = 50
	usernameRE        = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

// UsernameValidator validates usernames.
type UsernameValidator struct {
	minLength int
	maxLength int
}

// NewUsernameValidator creates a new UsernameValidator.
func NewUsernameValidator() *UsernameValidator {
	return &UsernameValidator{
		minLength: usernameMinLength,
		maxLength: usernameMaxLength,
	}
}

// Validate validates a username.
func (v *UsernameValidator) Validate(username string) error {
	if len(username) < v.minLength {
		return ErrUsernameTooShort
	}
	if len(username) > v.maxLength {
		return ErrUsernameTooLong
	}
	if !usernameRE.MatchString(username) {
		return ErrUsernameInvalid
	}
	return nil
}

// IsValid checks if a username is valid.
func (v *UsernameValidator) IsValid(username string) bool {
	return v.Validate(username) == nil
}

// ValidateUsername validates a username using default rules.
func ValidateUsername(username string) error {
	return NewUsernameValidator().Validate(username)
}

// IsValidUsername checks if a username is valid using default rules.
func IsValidUsername(username string) bool {
	return ValidateUsername(username) == nil
}
