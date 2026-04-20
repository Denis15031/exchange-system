package uid

import (
	"fmt"

	"github.com/google/uuid"
)

func New() string {
	return uuid.New().String()
}

func IsValid(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

func Validate(id, fieldName string) error {
	if id == "" {
		return fmt.Errorf("%s: cannot be empty", fieldName)
	}
	if !IsValid(id) {
		return fmt.Errorf("%s: invalid UUID format", fieldName)
	}
	return nil
}
