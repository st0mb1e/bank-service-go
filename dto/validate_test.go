package dto

import "testing"

func TestValidateEmail(t *testing.T) {
	t.Parallel()
	if err := ValidateEmail(""); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidateEmail("user@sub.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEmail("a@b.co"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEmail("not-an-email"); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateUsername(t *testing.T) {
	t.Parallel()
	if err := ValidateUsername("ab"); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidateUsername("good_user_1"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateUsername("bad user"); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidatePassword("12345678"); err != nil {
		t.Fatal(err)
	}
}
