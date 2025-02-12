package encoder

import (
	"crypto/sha256"
	"strconv"
)

func GenerateSecureShortId(url string, seed int) (string, error) {
	const base63Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_"
	sha := sha256.Sum256([]byte(url + strconv.Itoa(seed)))
	b := make([]byte, 10)
	for i := range b {
		b[i] = base63Chars[sha[i]%63]
	}
	return string(b), nil
}
