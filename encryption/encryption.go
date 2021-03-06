// Package encryption provides methods for deterministic encryption, a wrapper type
// EncryptedString for transparent encryption/decryption of strings during database operations
// and custom JSON marshaller/unmarshaller for correct serialization/unserialization.
package encryption

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// A prefix which indicates encrypted content
const marker = "·"

var (
	// Encryption key
	key []byte

	// Disabled indicates if encryption is disabled
	Disabled bool
)

// EncryptedString is a wrappper around plain string allowing it to be transparently encrypted and decrypted
type EncryptedString struct {
	S string
}

// SetKey sets encryption key
func SetKey(secretKey []byte) error {
	keyLen := len(secretKey)

	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return fmt.Errorf("invalid encryption key; must be 16, 24, or 32 bytes (got %d)", keyLen)
	}

	key = secretKey
	return nil
}

func (es EncryptedString) String() string {
	return es.S
}

// Equals tells if this ecrypted string is the same as the given encrypted string
func (es EncryptedString) Equals(estr EncryptedString) bool {
	return es.String() == estr.String()
}

// MarshalJSON marshals nested cleartext string
func (es *EncryptedString) MarshalJSON() ([]byte, error) {
	return json.Marshal(es.S)
}

// UnmarshalJSON unmarshals string into S var
func (es *EncryptedString) UnmarshalJSON(b []byte) error {
	var value string
	err := json.Unmarshal(b, &value)

	if err != nil {
		return err
	}

	es.S = value
	return nil
}

// Scan implements sql.Scanner and decryptes incoming sql column data
func (es *EncryptedString) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		rawString, err := Decrypt(v)
		if err != nil {
			return err
		}
		es.S = rawString
	case []byte:
		rawString, err := Decrypt(string(v))
		if err != nil {
			return err
		}
		es.S = rawString
	default:
		return fmt.Errorf("couldn't scan %+v", reflect.TypeOf(value))
	}

	return nil
}

// Value implements driver.Valuer and encrypts outgoing bind values
func (es EncryptedString) Value() (value driver.Value, err error) {
	return Encrypt(es.S)
}

// Encrypt AES-encrypts the given string and then base64-encode's it
func Encrypt(text string) (string, error) {
	if Disabled || text == "" || strings.HasPrefix(text, marker) {
		return text, nil
	}

	plaintext := []byte(text)
	block, err := aes.NewCipher(key)

	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	hasher := md5.New()
	hasher.Write(plaintext)
	hash := hasher.Sum(nil)
	copy(iv, hash)
	cipher.NewCFBEncrypter(block, iv).XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	return marker + base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt base64-decodes and then AES decrypts the given string
func Decrypt(cryptoText string) (string, error) {
	if Disabled || cryptoText == "" || !strings.HasPrefix(cryptoText, marker) {
		return cryptoText, nil
	}

	cryptoText = strings.TrimPrefix(cryptoText, marker)
	ciphertext, err := base64.URLEncoding.DecodeString(cryptoText)

	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)

	if err != nil {
		return "", err
	}

	if byteLen := len(ciphertext); byteLen < aes.BlockSize {
		return "", fmt.Errorf("invalid cipher size %d", byteLen)
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	cipher.NewCFBDecrypter(block, iv).XORKeyStream(ciphertext, ciphertext)
	hasher := md5.New()
	hasher.Write(ciphertext)

	if !bytes.Equal(hasher.Sum(nil), iv) {
		return "", errors.New("unable to decrypt: incorrect encryption key")
	}

	return string(ciphertext), nil
}
