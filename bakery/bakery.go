package bakery

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/elliotchance/phpserialize"
)

var ErrUnknownCookieType = errors.New("bakery: unknown cookie type")
var ErrUnknownUserRole = errors.New("bakery: could not determine user role")
var ErrUnknownUserEmail = errors.New(`bakery: could not determine user email`)

// Cookie contains props of a Bakery SSO cookie
type Cookie struct {
	Raw             string
	IsOatmeal       bool
	IsChocolateChip bool
	User            string
	Role            string
	Error           string
}

var key string

// SetKey sets encryption key
func SetKey(encryptionKey string) {
	key = encryptionKey
}

// Key returns the encryption key or panics if its not set
func Key() string {
	if key == "" {
		panic("bakery: encryption key is not set")
	}

	return key
}

// ParseCookie decrypts an HMAC-signed cookie encrypted with AES-128 (ECB mode),
// parses the underlying serialized PHP data structure and returns some props wrapped in Cookie type.
func ParseCookie(cookie string) (*Cookie, error) {
	serialized, err := decrypt(cookie)

	if err != nil {
		return nil, err
	}

	c := Cookie{}
	c.Raw = serialized

	if strings.Contains(serialized, "OATMEALSSL") {
		c.IsOatmeal = true
		c.IsChocolateChip = false
	} else if strings.Contains(serialized, "CHOCOLATECHIPSSL") {
		c.IsOatmeal = false
		c.IsChocolateChip = true
	} else {
		return nil, ErrUnknownCookieType
	}

	if c.IsChocolateChip {
		mm := regexp.MustCompile(`"mail";s:\d+:"(\S+?)";`).FindStringSubmatch(serialized)

		rm := regexp.
			MustCompile(`"roles";a:\d+:{.*"(administrator|Partner|Security Awareness User|Child User|LMS User)";.*}`).
			FindStringSubmatch(serialized)

		if len(mm) < 2 {
			return nil, ErrUnknownUserEmail
		}

		if len(rm) < 2 {
			return nil, ErrUnknownUserRole
		}

		c.User, c.Role = mm[1], rm[1]
	}

	if c.IsOatmeal {
		m := regexp.
			MustCompile(`s:6:"errors";a:\d+:{s:\d+:"\S+?";s:\d+:"([^<>=;}]+).*?";}`).
			FindStringSubmatch(serialized)

		if len(m) >= 2 {
			c.Error = m[1]
		}
	}

	return &c, nil
}

// CreateOatmealCookie generates an HMAC-signed cookie encrypted with AES-128 (ECB mode).
// Such cookie is to be used with Drupal Bakery SSO module for transferring login credentials.
func CreateOatmealCookie(username, password, destination, slave string) (string, error) {
	props := map[interface{}]interface{}{
		"data": map[string]interface{}{
			"name":        username,
			"pass":        password,
			"op":          "Log in",
			"destination": destination,
			"query":       []string{},
		},

		"name":      username,
		"calories":  320,
		"timestamp": time.Now().UTC().Unix(),
		"master":    0,
		"slave":     slave,
		"type":      "OATMEALSSL",
	}

	serializedProps, err := phpserialize.Marshal(props, nil)

	if err != nil {
		return "", err
	}

	cookie, err := encrypt(string(serializedProps))
	return cookie, err
}

// CreateChocolatechipCookie generates an HMAC-signed cookie encrypted with AES-128 (ECB mode).
// Such cookie is to be used with Drupal Bakery SSO module for transferring user credentials.
func CreateChocolatechipCookie(username string, role string) (string, error) {
	props := map[interface{}]interface{}{
		"data": map[string]interface{}{
			"mail": username,
		},

		"roles": map[int]string{
			2: "authenticated user",
			6: role,
		},

		"mail":      username,
		"calories":  320,
		"timestamp": time.Now().UTC().Unix(),
		"master":    1,
		"type":      "CHOCOLATECHIPSSL",
	}

	serializedProps, err := phpserialize.Marshal(props, nil)

	if err != nil {
		return "", err
	}

	cookie, err := encrypt(string(serializedProps))
	return cookie, err
}

func isValidHMACSignature(message, signature []byte) bool {
	return hmac.Equal(signature, generateHMACSignature(message))
}

func generateHMACSignature(message []byte) []byte {
	mac := hmac.New(sha256.New, []byte(Key()))
	mac.Write(message)
	return mac.Sum(nil)
}

func pad(data []byte) []byte {
	blockSize := 16
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)

}

func unpad(data []byte) []byte {
	length := len(data)
	unpadding := int(data[length-1])
	return data[:(length - unpadding)]
}

func decrypt(cookie string) (string, error) {
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

	if !isValidHMACSignature(data, sig) {
		return "", errors.New("bad HMAC signature or message")
	}

	block, err := aes.NewCipher([]byte(Key())[:32])

	if err != nil {
		return "", err
	}

	decrypter := newECBDecrypter(block)
	decrypter.CryptBlocks(data, data)
	return string(data), nil
}

func encrypt(data string) (string, error) {
	block, err := aes.NewCipher([]byte(Key())[:32])

	if err != nil {
		return "", err
	}

	paddedData := pad([]byte(data))
	encryptedData := make([]byte, len(paddedData))
	encrypter := newECBEncrypter(block)
	encrypter.CryptBlocks(encryptedData, paddedData)
	sig := generateHMACSignature(encryptedData)

	return url.QueryEscape(
		base64.StdEncoding.EncodeToString(
			append([]byte(hex.EncodeToString(sig)), encryptedData...),
		),
	), nil
}

type ecb struct {
	b         cipher.Block
	blockSize int
}

func newECB(b cipher.Block) *ecb {
	return &ecb{
		b:         b,
		blockSize: b.BlockSize(),
	}
}

type ecbEncrypter ecb

func newECBEncrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbEncrypter)(newECB(b))
}

func (x *ecbEncrypter) BlockSize() int {
	return x.blockSize
}

func (x *ecbEncrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Encrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

type ecbDecrypter ecb

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
