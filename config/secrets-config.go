package config

import (
	"os"
	"strconv"
)

type SecretsConfig struct {
	JWTSecret        string
	HMACSecret       string
	PGPPassphrase    string
	CBRMarginPercent float64
}

func NewSecretsConfigFromEnv() *SecretsConfig {
	margin := 5.0
	if v := os.Getenv("CBR_MARGIN_PERCENT"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			margin = f
		}
	}
	return &SecretsConfig{
		JWTSecret:        os.Getenv("JWT_SECRET"),
		HMACSecret:       os.Getenv("HMAC_SECRET"),
		PGPPassphrase:    os.Getenv("PGP_PASSPHRASE"),
		CBRMarginPercent: margin,
	}
}
