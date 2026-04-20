package validate

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
)

var (
	ErrEmptyField    = errors.New("field cannot be empty")
	ErrInvalidEmail  = errors.New("invalid email format")
	ErrWeakPassword  = errors.New("password does not meet requirements")
	ErrInvalidUUID   = errors.New("invalid UUID format")
	ErrTooShort      = errors.New("field is too short")
	ErrTooLong       = errors.New("field is too long")
	ErrInvalidFormat = errors.New("invalid format")
	ErrInvalidRange  = errors.New("value out of range")
)

func Email(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email: %w", ErrEmptyField)
	}

	if len(email) > 254 {
		return fmt.Errorf("email: %w (max 254)", ErrTooLong)
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("email: %w", ErrInvalidEmail)
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("email: %w", ErrInvalidEmail)
	}

	return nil
}

func Password(password string, minLen, maxLen int) error {
	if password == "" {
		return fmt.Errorf("password: %w", ErrEmptyField)
	}

	if len(password) < minLen {
		return fmt.Errorf("password: %w (min %d characters)", ErrTooShort, minLen)
	}

	if len(password) > maxLen {
		return fmt.Errorf("password: %w (max %d characters)", ErrTooLong, maxLen)
	}

	hasLetter := false
	hasDigit := false
	hasSpecial := false

	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if !hasLetter || (!hasDigit && !hasSpecial) {
		return fmt.Errorf("password: %w (must contain letters and at least one digit or special character)", ErrWeakPassword)
	}

	return nil
}

func UUID(uuid string) error {
	if uuid == "" {
		return fmt.Errorf("uuid: %w", ErrEmptyField)
	}

	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidPattern.MatchString(strings.ToLower(uuid)) {
		return fmt.Errorf("uuid: %w", ErrInvalidUUID)
	}

	return nil
}

func String(value string, fieldName string, minLen, maxLen int, pattern *regexp.Regexp) error {
	if value == "" {
		return fmt.Errorf("%s: %w", fieldName, ErrEmptyField)
	}

	if len(value) < minLen {
		return fmt.Errorf("%s: %w (min %d characters)", fieldName, ErrTooShort, minLen)
	}

	if len(value) > maxLen {
		return fmt.Errorf("%s: %w (max %d characters)", fieldName, ErrTooLong, maxLen)
	}

	if pattern != nil && !pattern.MatchString(value) {
		return fmt.Errorf("%s: %w", fieldName, ErrInvalidFormat)
	}

	return nil
}

func NumberRange(value, min, max float64, fieldName string) error {
	if value < min || value > max {
		return fmt.Errorf("%s: %w (must be between %v and %v)", fieldName, ErrInvalidRange, min, max)
	}
	return nil
}

func NotEmpty(value interface{}, fieldName string) error {
	switch v := value.(type) {
	case string:
		if v == "" {
			return fmt.Errorf("%s: %w", fieldName, ErrEmptyField)
		}
	case nil:
		return fmt.Errorf("%s: %w", fieldName, ErrEmptyField)
	default:
	}
	return nil
}
