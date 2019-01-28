package chocolatechip

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPropFromEncryptedCookie(t *testing.T) {
	const key = "ENCRYPTIONKEY"
	const cookie = "M2E2MGM4MTM4YWFmNjllYmNmNjkyNGU2ODRhNDQ3Njg0ZTNjZWE0MmJjMzI0YmRlNDEzNTViZmEwZjZiYzJlNqq4tJdJVyJ6rVJcr%2FWqSKAA3EqVA%2Fh%2BnzvbSFzzB29iOnbH240WZB%2B5TFtw%2BR6Vx66CtEOLD7mRE%2F0RppI0RJoh8dYCvQLBWYO7l7zSD7krMucFObkak0ovhpkkNnuhH7b0v9venKFJvGozJckppFl60nNQPWHGjgF4mWIPzol%2FrV%2FSsg5rXEB%2FCprUZGHflv9AAkbqNpeoBP8I4a8NbWuTafjO19RIUOmhYqPGuySbfl8FmHYr5qu%2B8s2HqbnyrSHSCthvTifBD3oHM1dd9KcJxNpA95csNg7db1ZEa59SY%2FmfVSc3kEM8UlwVV5u9NQ%3D%3D"

	val, err := GetPropFromEncryptedCookie("mail", cookie, key)
	assert.NoError(t, err)
	assert.Equal(t, "eugene.mastervip@gmail.com", val)
}
