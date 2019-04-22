package usersync

import (
	"encoding/json"
	"errors"
	"fmt"
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

// SlaveURL URL to identify origin website
const SlaveURL = "https://awareness.everycloudtech.com/"

// PushUser sends user details to the main server and returns error if something is wrong and
// in case of success it returns a master user id assigned to the newly created user.
func PushUser(id int64, username, email, fullName, password string, rid, pid int64) (int64, error) {
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
		return 0, err
	}

	if resp.StatusCode != 200 {
		return 0, errors.New("Unable to sync user - status code: " + strconv.Itoa(resp.StatusCode))
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return 0, err
	}

	respData := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		User    struct {
			UID string `json:"uid"`
		} `json:"user"`
	}{}

	err = json.Unmarshal(body, &respData)

	if err != nil {
		return 0, fmt.Errorf("Could not parse response from the main server - %s", err.Error())
	}

	if !respData.Success {
		msg := "Unable to sync user - got unexpected response from the main server"

		if respData.Message != "" {
			msg = respData.Message
		}

		return 0, errors.New(msg)
	}

	uid, err := strconv.ParseInt(respData.User.UID, 10, 0)

	if err != nil {
		return 0, fmt.Errorf("Could not parse returned master user id - %s", err.Error())
	}

	return uid, nil
}

// UpdateUser tells the main server to update username, email, password, role and partner id of
// a user with the given bakery user id (the main server itself must decide which props have changed)
func UpdateUser(buid int64, username, email, password string, role, partner int64) error {
	if buid == 0 {
		return nil
	}

	params := url.Values{
		"masteruserid": {strconv.FormatInt(buid, 10)},
		"username":     {username},
		"email":        {email},
		"password":     {password},
		"roles":        {strconv.FormatInt(role, 10)},
		"partner":      {strconv.FormatInt(partner, 10)},
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", APIURL+"/update", strings.NewReader(params.Encode()))
	req.SetBasicAuth(APIUser, APIPassword)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("Unable to update user on the main server - status code: " + strconv.Itoa(resp.StatusCode))
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
		return fmt.Errorf("Could not parse response from the main server - %s", err.Error())
	}

	if !respData.Success {
		msg := "Unable to update user - got unexpected response from the main server"

		if respData.Message != "" {
			msg = respData.Message
		}

		return errors.New(msg)
	}

	return nil
}

// DeleteUser tells the main server to delete a user with the given bakery user id
func DeleteUser(buid int64) error {
	if buid == 0 {
		return nil
	}

	params := url.Values{
		"slave_uid": {strconv.FormatInt(buid, 10)},
		"slave":     {SlaveURL},
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", APIURL+"/delete", strings.NewReader(params.Encode()))
	req.SetBasicAuth(APIUser, APIPassword)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("Unable to delete user on the main server - status code: " + strconv.Itoa(resp.StatusCode))
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
		return fmt.Errorf("Could not parse response from the main server - %s", err.Error())
	}

	if !respData.Success {
		msg := "Unable to delete user on the main server"

		if respData.Message != "" {
			msg = respData.Message
		}

		return errors.New(msg)
	}

	return nil
}
