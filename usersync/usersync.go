package usersync

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// APIURL is a URL of the user sync API
var APIURL = "https://www.everycloudtech.com/api/bakery"

// APIUser is a username used during authentication
var APIUser = os.Getenv("USERSYNC_API_USER")

// APIPassword is a password used during authentication
var APIPassword = os.Getenv("USERSYNC_API_PASSWORD")

// PushUser sends user details to the main server and returns error if something is wrong
func PushUser(id int64, username, email, fullName, password string, rid, pid int64) error {
	params := url.Values{
		"userid":   {strconv.FormatInt(id, 10)},
		"username": {username},
		"fullname": {fullName},
		"email":    {email},
		"password": {password},
		"partner":  {strconv.FormatInt(pid, 10)},
		"roles":    {strconv.FormatInt(rid, 10)},
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", APIURL+"/create", strings.NewReader(params.Encode()))
	req.SetBasicAuth(APIUser, APIPassword)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("Unable to sync user - status code: " + strconv.Itoa(resp.StatusCode))
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	respData := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{}

	err = json.Unmarshal(body, &respData)

	if err != nil {
		return err
	}

	if !respData.Success {
		msg := "Unable to sync user - got unexpected response from the main server"

		if respData.Message != "" {
			msg = respData.Message
		}

		return errors.New(msg)
	}

	return nil
}
