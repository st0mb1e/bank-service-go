package cryptoutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func ComputeHMAC(data string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	_, _ = h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func VerifyHMAC(data string, secret []byte, macHex string) bool {
	return hmac.Equal([]byte(ComputeHMAC(data, secret)), []byte(macHex))
}
