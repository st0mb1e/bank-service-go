package dto

import (
	"errors"
	"net/mail"
	"regexp"
	"unicode/utf8"
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

func ValidateEmail(s string) error {
	if s == "" || len(s) > 254 {
		return errors.New("invalid email")
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address == "" {
		return errors.New("invalid email")
	}
	return nil
}

func ValidateUsername(s string) error {
	if s == "" {
		return errors.New("username required")
	}
	if !usernameRe.MatchString(s) {
		return errors.New("username must be 3–32 chars: letters, digits, underscore")
	}
	return nil
}

func ValidatePassword(p string) error {
	if utf8.RuneCountInString(p) < 8 {
		return errors.New("password too short")
	}
	if len(p) > 72 {
		return errors.New("password too long")
	}
	return nil
}

func ValidateRegisterFields(email, username, password string) error {
	if err := ValidateEmail(email); err != nil {
		return err
	}
	if err := ValidateUsername(username); err != nil {
		return err
	}
	return ValidatePassword(password)
}

func ValidateLoginFields(email, password string) error {
	if err := ValidateEmail(email); err != nil {
		return err
	}
	if password == "" {
		return errors.New("password required")
	}
	return ValidatePassword(password)
}
