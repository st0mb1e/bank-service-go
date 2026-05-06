package cryptoutil

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

func EncryptPGPSymmetric(plaintext []byte, passphrase []byte) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, fmt.Errorf("empty PGP passphrase")
	}
	cfg := &packet.Config{}
	buf := &bytes.Buffer{}
	w, err := openpgp.SymmetricallyEncrypt(buf, passphrase, nil, cfg)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(plaintext); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecryptPGPSymmetric(ciphertext []byte, passphrase []byte) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, fmt.Errorf("empty PGP passphrase")
	}
	md, err := openpgp.ReadMessage(bytes.NewReader(ciphertext), nil, func(keys []openpgp.Key, symmetric bool) ([]byte, error) {
		return passphrase, nil
	}, nil)
	if err != nil {
		return nil, err
	}
	out, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return nil, err
	}
	return out, nil
}
