package domain

import (
	"fmt"
	"time"

	userv1 "exchange-system/proto/user/v1"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	UserID    string
	Email     string
	Username  string
	Password  string
	Role      userv1.UserRole
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func HashPassword(password string, cost int) (string, error) {
	if cost < 10 || cost > 14 {
		return "", fmt.Errorf("invalid bcrypt cost: %d (must be 10-14)", cost)
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (u *User) ToProto() *userv1.User {
	return &userv1.User{
		UserId:    u.UserID,
		Email:     u.Email,
		Username:  u.Username,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}

func UserFromProto(p *userv1.User) *User {
	if p == nil {
		return nil
	}
	return &User{
		UserID:    p.UserId,
		Email:     p.Email,
		Username:  p.Username,
		Role:      p.Role,
		IsActive:  p.IsActive,
		CreatedAt: p.CreatedAt.AsTime(),
		UpdatedAt: p.UpdatedAt.AsTime(),
		Password:  "",
	}
}

func ValidateEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}

	atCount := 0
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			atCount++
			atIndex = i
		}
	}
	if atCount != 1 || atIndex <= 0 || atIndex >= len(email)-1 {
		return false
	}

	local := email[:atIndex]
	domain := email[atIndex+1:]

	if len(local) == 0 || len(domain) < 3 {
		return false
	}

	hasDot := false
	for _, c := range domain {
		if c == '.' {
			hasDot = true
			break
		}
	}
	if !hasDot {
		return false
	}

	for _, c := range email {
		if c == ' ' {
			return false
		}
	}

	return true
}

func ValidatePassword(password string) bool {
	return len(password) >= 8
}
