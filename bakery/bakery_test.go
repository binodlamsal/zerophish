package bakery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	SetKey("YOURENCRYPTIONKEY")
}

func TestParseChocolatechipCookie(t *testing.T) {
	const cookie = "MTc1NTFiMTk1ZDNhOTI3MWI0ODJjNjZhNzU2NTIyZDg3OThhN2FjNzdkZjVjZjVhNzY0NjIwZjUzMDgxYmJmMeBEgMt30umvtAH1pS5C%2FwToeDSXyIdmoKFcA7tcZRtfG7lAy26I%2FCXZdS00mNYYgF9i9%2BUpDoMJFp6eV5Nqo4Kia%2BapleibFgUTl5kadvLv9CDrMIyX56YaFzZSonAuXPfeLNo1k1Wz%2FrfzEAEc0PKWju9lrhqF4SrPKBmbfdT0r1aN4gstYfdMr%2FtTZOoWBlVV%2F%2FcuZ%2FANQtDQ3FMu9GBqxi6z%2BAPEVyC8YZrmFQE%2FLXOIH0INq1HLKrlLdBmwIp0XKjww6fltwW3mbJBpnb8eqWKF1bjOpHK0EQ%2B8aunoPeznjWiBUQM1YhKiD6rt6gM8tfjx%2BHt2c0xOqn5JbfInV2eDceN2jcWMXa8F%2Fgd4TYPNNr3o9297UpIMbPWIeaBcf4n82xy%2BEklHGEVybgfhsmPdXLuRDM6rkU6Grb7i"
	c, err := ParseCookie(cookie)
	assert.NoError(t, err)
	assert.True(t, c.IsChocolateChip)
	assert.False(t, c.IsOatmeal)
	assert.Equal(t, "gophishusertest@ec.com", c.User)
	assert.Equal(t, "Partner", c.Role)
	assert.Empty(t, c.Error)
}

func TestParseOatmealCookie(t *testing.T) {
	const cookie = "ODZiMjUxMjdlYTQzYTc0NDk0ZjQ0ZGEzNjFkYWY4ZTM2OGQ2ZTQ3ZmIwYTdiODYzMTA1Njc4YThiYzBjNzVhMKWAFEmG%2FNutdJ93u4DxZKCMaMv1iB5au61d7RxCfvmj9gqjP5spZ4DzTnw3xpyvQTnT7nFrI83Vddj0xMySCtFNP2jq5Ev%2FsSvpFWno6KeyisZkPc7hs7LwfXeng7aYEMNbSl8O9j90G9eNYMVi8nTpqTF%2F3B4d2IBBIjlj2ym1JpNZy1HWtSQelk3YrQH%2BEGNw0M0Rb%2BwzyduNiOo2gy8AyaTgxLSLgJXwUOSEzhy0StwX88dc881UqxUHXybItDIuiCMrDVUwBwopN5kG6%2F1gBOETi01NMKzC3XMllcH1smTF9CBS2GrYfjGn3dEmINTe9Uf78twY0m4TlKiOZsQtc4gfQg2rUcsquUt0GbAisc3kfI6jC23%2FLIoC0fat%2FOV5XsSKzkCYK54FYACr5E3tPtk8xzLzB9i7P73sB0nfeDyiKJ%2BIBpL2ViHlSuUCQw%3D%3D"
	c, err := ParseCookie(cookie)
	assert.NoError(t, err)
	assert.True(t, c.IsOatmeal)
	assert.False(t, c.IsChocolateChip)
	assert.NotEmpty(t, c.Error)
}

func TestCreateOatmealCookie(t *testing.T) {
	username := "binod@everycloud.tech"
	cookie, err := CreateOatmealCookie(username, "password", "dashboard", "https://mailflow.everycloudtech.com/")
	assert.NoError(t, err)
	c, err := ParseCookie(cookie)
	assert.NoError(t, err)
	assert.True(t, c.IsOatmeal)
	assert.False(t, c.IsChocolateChip)
	assert.Empty(t, c.Error)
}
