// Package utils provides password hashing utilities using bcrypt.
package utils

import (
	"golang.org/x/crypto/bcrypt"
)

// DefaultCost is the default bcrypt cost (12 is recommended for production).
const DefaultCost = 12

// HashPassword hashes a password using bcrypt.
func HashPassword(password string) (string, error) {
	return HashPasswordWithCost(password, DefaultCost)
}

// HashPasswordWithCost hashes a password with a specific bcrypt cost.
func HashPasswordWithCost(password string, cost int) (string, error) {
	if cost < bcrypt.MinCost {
		cost = bcrypt.MinCost
	}
	if cost > bcrypt.MaxCost {
		cost = bcrypt.MaxCost
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// CheckPassword verifies if a password matches the hash.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// CheckPasswordAndRehash checks password and suggests rehash if needed.
// This is useful when you want to upgrade password hashes over time.
func CheckPasswordAndRehash(password, hash string, currentCost int) (bool, bool) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return false, false
	}

	// Check if rehash is needed (cost has changed)
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return true, false
	}

	needsRehash := cost != currentCost
	return true, needsRehash
}

// ValidatePasswordStrength performs basic password strength validation.
// Returns true if password meets minimum requirements.
func ValidatePasswordStrength(password string) bool {
	return len(password) >= 8
}

// ValidatePasswordStrengthWithRules validates password against custom rules.
type PasswordRules struct {
	MinLength      int
	MaxLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireDigit   bool
	RequireSpecial bool
}

// DefaultPasswordRules returns default password rules.
func DefaultPasswordRules() PasswordRules {
	return PasswordRules{
		MinLength:      8,
		MaxLength:      128,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: false,
	}
}

// Validate validates a password against the rules.
func (r PasswordRules) Validate(password string) bool {
	if len(password) < r.MinLength || len(password) > r.MaxLength {
		return false
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	if r.RequireUpper && !hasUpper {
		return false
	}
	if r.RequireLower && !hasLower {
		return false
	}
	if r.RequireDigit && !hasDigit {
		return false
	}
	if r.RequireSpecial && !hasSpecial {
		return false
	}

	return true
}
