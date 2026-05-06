package cryptoutil

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func GenerateLuhnPAN(length int) (string, error) {
	if length != 16 {
		return "", fmt.Errorf("only 16-digit PAN supported")
	}
	prefix := "4"
	buf := make([]byte, 14)
	for i := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		buf[i] = byte('0' + n.Int64())
	}
	body := prefix + string(buf)
	for d := 0; d < 10; d++ {
		candidate := body + string(byte('0'+d))
		if ValidLuhn(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("luhn check digit not found")
}

func ValidLuhn(pan string) bool {
	if len(pan) < 2 {
		return false
	}
	sum := 0
	alternate := false
	for i := len(pan) - 1; i >= 0; i-- {
		if pan[i] < '0' || pan[i] > '9' {
			return false
		}
		d := int(pan[i] - '0')
		if alternate {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alternate = !alternate
	}
	return sum%10 == 0
}
