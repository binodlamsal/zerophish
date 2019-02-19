package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidPassword(t *testing.T) {
	tooShort := "qwerty"
	noNum := "fwefliciwcifrfrf"
	noSpecial := "eiuvnervjnkj5"
	noAlpha := "189813165466"
	good := "qw3rty_$"
	withSpace := "qwerty1 $"

	assert.False(t, IsValidPassword(tooShort))
	assert.False(t, IsValidPassword(noNum))
	assert.False(t, IsValidPassword(noSpecial))
	assert.False(t, IsValidPassword(noAlpha))
	assert.True(t, IsValidPassword(good))
	assert.False(t, IsValidPassword(withSpace))
}
