package usersync

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

func TestPushUser(t *testing.T) {
	APIURL = "https://localhost:3333/api/bakery"
	uid, err := PushUser(1001, "test1001", "test1001@test.com", "Test 1001", "qwerty", 3, 0, false)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, uid)
	assert.FailNow(t, "Stop")
}

func TestUpdateUser(t *testing.T) {
	APIURL = "https://localhost:3333/api/bakery"
	err := UpdateUser(1001, "test1001", "test1001@test.com", "qwerty", 3, 0)
	assert.NoError(t, err)
	assert.FailNow(t, "Stop")
}

func TestDeleteUser(t *testing.T) {
	APIURL = "https://localhost:3333/api/bakery"
	err := DeleteUser(1001)
	assert.NoError(t, err)
	assert.FailNow(t, "Stop")
}
