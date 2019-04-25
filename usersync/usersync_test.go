package usersync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPushUser(t *testing.T) {
	APIURL = "https://www.everycloudtech.com/api/bakery"
	APIUser = ""
	APIPassword = ""
	uid, err := PushUser(1001, "test1001", "test1001@test.com", "Test 1001", "qwerty", 3, 0, false)
	t.Log(err)
	assert.NotEqual(t, 0, uid)
	assert.NoError(t, err)
}

func TestPushUserMockFailed(t *testing.T) {
	APIURL = "https://d7a855bb-2a45-4954-a53e-e680831db088.mock.pstmn.io/failed"
	_, err := PushUser(1, "woody", "woody@forest.net", "Woody Woodpecker", "w00d", 1, 0, false)
	t.Log(err)
	assert.Error(t, err)
}

func TestPushUserMockSuccessful(t *testing.T) {
	APIURL = "https://d7a855bb-2a45-4954-a53e-e680831db088.mock.pstmn.io/successful"
	_, err := PushUser(1, "woody", "woody@forest.net", "Woody Woodpecker", "w00d", 1, 0, false)
	t.Log(err)
	assert.NoError(t, err)
}
