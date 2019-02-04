package bakery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	SetKey("YOURENCRYPTIONKEY")
}

func TestParseChocolatechipCookie(t *testing.T) {
	const cookie = "M2E2MGM4MTM4YWFmNjllYmNmNjkyNGU2ODRhNDQ3Njg0ZTNjZWE0MmJjMzI0YmRlNDEzNTViZmEwZjZiYzJlNqq4tJdJVyJ6rVJcr%2FWqSKAA3EqVA%2Fh%2BnzvbSFzzB29iOnbH240WZB%2B5TFtw%2BR6Vx66CtEOLD7mRE%2F0RppI0RJoh8dYCvQLBWYO7l7zSD7krMucFObkak0ovhpkkNnuhH7b0v9venKFJvGozJckppFl60nNQPWHGjgF4mWIPzol%2FrV%2FSsg5rXEB%2FCprUZGHflv9AAkbqNpeoBP8I4a8NbWuTafjO19RIUOmhYqPGuySbfl8FmHYr5qu%2B8s2HqbnyrSHSCthvTifBD3oHM1dd9KcJxNpA95csNg7db1ZEa59SY%2FmfVSc3kEM8UlwVV5u9NQ%3D%3D"
	c, err := ParseCookie(cookie)
	assert.NoError(t, err)
	assert.True(t, c.IsChocolateChip)
	assert.False(t, c.IsOatmeal)
	assert.Equal(t, "eugene.mastervip@gmail.com", c.User)
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
