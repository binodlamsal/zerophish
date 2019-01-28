package chocolatechip

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/url"
	"regexp"
)

// GetPropFromEncryptedCookie decrypts an HMAC-signed cookie encrypted with AES-128 (ECB mode)
// and returns a value of the given string property. The content of the decrypted cookie data
// is expected to be in the form of a serialize()'d PHP array.
func GetPropFromEncryptedCookie(prop, cookie, key string) (string, error) {
	unescapedBase64EncodedSigAndData, err := url.QueryUnescape(cookie)

	if err != nil {
		return "", err
	}

	sigAndData, err := base64.StdEncoding.DecodeString(unescapedBase64EncodedSigAndData)

	if err != nil {
		return "", err
	}

	sig, err := hex.DecodeString(string(sigAndData[:64]))

	if err != nil {
		return "", err
	}

	data := sigAndData[64:]

	if !isValidHMACSignature(data, sig, []byte(key)) {
		return "", errors.New("bad HMAC signature or message")
	}

	block, err := aes.NewCipher([]byte(key)[:32])

	if err != nil {
		return "", err
	}

	mode := newECBDecrypter(block)
	mode.CryptBlocks(data, data)
	m := regexp.MustCompile(prop + `";s:\d+:"(\S+?)";`).FindStringSubmatch(string(data))

	if len(m) < 2 {
		return "", errors.New("no such prop - " + prop)
	}

	return m[1], nil
}

func isValidHMACSignature(message, signature, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	expectedSignature := mac.Sum(nil)
	return hmac.Equal(signature, expectedSignature)
}

type ecb struct {
	b         cipher.Block
	blockSize int
}

type ecbDecrypter ecb

func newECB(b cipher.Block) *ecb {
	return &ecb{
		b:         b,
		blockSize: b.BlockSize(),
	}
}

func newECBDecrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbDecrypter)(newECB(b))
}

func (x *ecbDecrypter) BlockSize() int { return x.blockSize }

func (x *ecbDecrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}

	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}

	for len(src) > 0 {
		x.b.Decrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}
