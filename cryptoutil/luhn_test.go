package cryptoutil

import "testing"

func TestGenerateLuhnPAN(t *testing.T) {
	t.Parallel()
	pan, err := GenerateLuhnPAN(16)
	if err != nil {
		t.Fatal(err)
	}
	if len(pan) != 16 {
		t.Fatalf("len=%d", len(pan))
	}
	if !ValidLuhn(pan) {
		t.Fatal("invalid luhn", pan)
	}
}
