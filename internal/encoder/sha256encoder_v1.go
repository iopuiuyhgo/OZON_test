package encoder

import (
	"crypto/sha256"
	"golang.org/x/crypto/openpgp/errors"
	"strconv"
)

func GenerateSecureShortId(url string, seed int, keyLen int) (string, error) {
	if keyLen > 32 {
		return "", errors.InvalidArgumentError("keyLen > 32")
	}
	const base63Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_"
	sha := sha256.Sum256([]byte(url + strconv.Itoa(seed)))
	b := make([]byte, keyLen)
	for i := range b {
		b[i] = base63Chars[sha[i]%byte(len(base63Chars))]
	}
	return string(b), nil
}
