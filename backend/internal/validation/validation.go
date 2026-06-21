package validation

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	slugRegex  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "; ")
}

func ValidateEmail(email string) *ValidationError {
	if email == "" {
		return &ValidationError{Field: "email", Message: "email is required"}
	}
	if len(email) > 255 {
		return &ValidationError{Field: "email", Message: "email must be less than 255 characters"}
	}
	if !emailRegex.MatchString(email) {
		return &ValidationError{Field: "email", Message: "invalid email format"}
	}
	return nil
}

func ValidatePassword(password string) *ValidationError {
	if password == "" {
		return &ValidationError{Field: "password", Message: "password is required"}
	}
	if len(password) < 6 {
		return &ValidationError{Field: "password", Message: "password must be at least 6 characters"}
	}
	if len(password) > 128 {
		return &ValidationError{Field: "password", Message: "password must be less than 128 characters"}
	}

	return nil
}

func ValidateName(name string) *ValidationError {
	if name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if len(name) < 2 {
		return &ValidationError{Field: "name", Message: "name must be at least 2 characters"}
	}
	if len(name) > 100 {
		return &ValidationError{Field: "name", Message: "name must be less than 100 characters"}
	}
	return nil
}

func ValidateSlug(slug string) *ValidationError {
	if slug == "" {
		return &ValidationError{Field: "slug", Message: "slug is required"}
	}
	if len(slug) < 2 {
		return &ValidationError{Field: "slug", Message: "slug must be at least 2 characters"}
	}
	if len(slug) > 50 {
		return &ValidationError{Field: "slug", Message: "slug must be less than 50 characters"}
	}
	if !slugRegex.MatchString(slug) {
		return &ValidationError{Field: "slug", Message: "slug must contain only lowercase letters, numbers, and hyphens"}
	}
	return nil
}

func ValidateOrgName(name string) *ValidationError {
	if name == "" {
		return &ValidationError{Field: "name", Message: "organization name is required"}
	}
	if len(name) < 2 {
		return &ValidationError{Field: "name", Message: "organization name must be at least 2 characters"}
	}
	if len(name) > 100 {
		return &ValidationError{Field: "name", Message: "organization name must be less than 100 characters"}
	}
	return nil
}

func ValidateConnectionType(connType string) *ValidationError {
	if connType == "" {
		return &ValidationError{Field: "type", Message: "connection type is required"}
	}
	if connType != "docker" && connType != "k8s" {
		return &ValidationError{Field: "type", Message: "connection type must be docker or k8s"}
	}
	return nil
}

func ValidateRole(role string) *ValidationError {
	if role == "" {
		return &ValidationError{Field: "role", Message: "role is required"}
	}
	validRoles := map[string]bool{
		"owner":  true,
		"admin":  true,
		"member": true,
		"viewer": true,
	}
	if !validRoles[role] {
		return &ValidationError{Field: "role", Message: "role must be owner, admin, member, or viewer"}
	}
	return nil
}

func SanitizeString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\x00", "")
	return s
}
