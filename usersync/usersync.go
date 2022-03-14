package usersync

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Jeffail/gabs"
	log "github.com/binodlamsal/zerophish/logger"
)

// Debug if set to true will route all API requests to a local endpoint
var Debug = false

// APIURL is a URL of the user sync API
var APIURL = "https://www.everycloud.com/api"

// TrainingAPIURL is a URL of the security awareness training API
var TrainingAPIURL = "https://awareness-stage.everycloud.com:4433/api"

// APIUser is a username used during authentication
var APIUser = os.Getenv("USERSYNC_API_USER")

// APIPassword is a password used during authentication
var APIPassword = os.Getenv("USERSYNC_API_PASSWORD")

// SlaveURL URL to identify origin website
const SlaveURL = "https://simulation.everycloud.com/"

func init() {
	if Debug {
		APIURL = "https://localhost:3333/api/mock"
		TrainingAPIURL = "https://localhost:3333/api/mock"
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
}

// PushUser sends user details to the main server and returns error if something is wrong and
// in case of success it returns a master user id assigned to the newly created user.
// sso flag - when set to true means this sync op should not create new user record but only update user id.
func PushUser(id int64, username, email, fullName, password string, rid, pid int64, sso bool) (int64, error) {
	params := url.Values{
		"userid":   {strconv.FormatInt(id, 10)},
		"username": {username},
		"fullname": {fullName},
		"email":    {email},
		"password": {password},
		"partner":  {strconv.FormatInt(pid, 10)},
		"roles":    {strconv.FormatInt(rid, 10)},
		"sso":      {strconv.FormatBool(sso)},
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", APIURL+"/bakery/create", strings.NewReader(params.Encode()))
	req.SetBasicAuth(APIUser, APIPassword)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if dump, err := httputil.DumpRequestOut(req, true); err == nil {
		log.WithFields(map[string]interface{}{"tag": "usersync.PushUser ->"}).Infof("%q", dump)
	}

	resp, err := client.Do(req)

	if err != nil {
		return 0, err
	}

	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		log.WithFields(map[string]interface{}{"tag": "usersync.PushUser <-"}).Infof("%q", dump)
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
	req, err := http.NewRequest("POST", APIURL+"/bakery/update", strings.NewReader(params.Encode()))
	req.SetBasicAuth(APIUser, APIPassword)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if dump, err := httputil.DumpRequestOut(req, true); err == nil {
		log.WithFields(map[string]interface{}{"tag": "usersync.UpdateUser ->"}).Infof("%q", dump)
	}

	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		log.WithFields(map[string]interface{}{"tag": "usersync.UpdateUser <-"}).Infof("%q", dump)
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
	req, err := http.NewRequest("POST", APIURL+"/bakery/delete", strings.NewReader(params.Encode()))
	req.SetBasicAuth(APIUser, APIPassword)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if dump, err := httputil.DumpRequestOut(req, true); err == nil {
		log.WithFields(map[string]interface{}{"tag": "usersync.DeleteUser ->"}).Infof("%q", dump)
	}

	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		log.WithFields(map[string]interface{}{"tag": "usersync.DeleteUser <-"}).Infof("%q", dump)
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

// ResetPassword instructs the main server to perform password reset procedure for the given bakery user id
func ResetPassword(buid int64) error {
	if buid == 0 {
		return nil
	}

	params := url.Values{"uid": {strconv.FormatInt(buid, 10)}}

	client := &http.Client{}
	req, err := http.NewRequest("POST", APIURL+"/v1/passwordreset", strings.NewReader(params.Encode()))
	req.SetBasicAuth(APIUser, APIPassword)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if dump, err := httputil.DumpRequestOut(req, true); err == nil {
		log.WithFields(map[string]interface{}{"tag": "usersync.ResetPassword ->"}).Infof("%q", dump)
	}

	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		log.WithFields(map[string]interface{}{"tag": "usersync.ResetPassword <-"}).Infof("%q", dump)
	}

	if resp.StatusCode != 200 {
		return errors.New("Unable to initiate password reset for user on the main server - status code: " + strconv.Itoa(resp.StatusCode))
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
		msg := "Could not reset password of the user on the main server"

		if respData.Message != "" {
			msg = respData.Message
		}

		return errors.New(msg)
	}

	return nil
}

// DeleteTrainingCampaigns tells the main server to delete training campaigns of user with the given uid
func DeleteTrainingCampaigns(uid int64) {
	id := strconv.FormatInt(uid, 10)
	params := url.Values{"uid": {id}}
	client := &http.Client{}

	req, err := http.NewRequest("POST", TrainingAPIURL+"/v1/delete_campaigns", strings.NewReader(params.Encode()))
	req.SetBasicAuth(APIUser, APIPassword)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if dump, err := httputil.DumpRequestOut(req, true); err == nil {
		log.WithFields(map[string]interface{}{"tag": "usersync.DeleteTrainingCampaigns ->"}).Infof("%q", dump)
	}

	resp, err := client.Do(req)

	if err != nil {
		log.Error(err)
		return
	}

	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		log.WithFields(map[string]interface{}{"tag": "usersync.DeleteTrainingCampaigns <-"}).Infof("%q", dump)
	}

	if resp.StatusCode != 200 {
		log.Errorf(
			"Unable to delete learning campaigns on the main server - status code: %s",
			strconv.Itoa(resp.StatusCode),
		)

		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Error(err)
		return
	}

	respData := struct {
		Code    bool   `json:"code"`
		Message string `json:"message"`
	}{}

	err = json.Unmarshal(body, &respData)

	if err != nil {
		log.Errorf("Could not parse response from the main server - %s", err.Error())
		return
	}

	if !respData.Code {
		msg := "Unable to delete training campaigns on the main server"

		if respData.Message != "" {
			msg = respData.Message
		}

		log.Error(errors.New(msg))
	}
}

// GetUserDetails requests details of user with the given bakery id from the main server
func GetUserDetails(buid int64) (fullname, domain string, sendEmail bool, err error) {
	params := url.Values{"uid": {strconv.FormatInt(buid, 10)}}

	client := &http.Client{}
	req, err := http.NewRequest("POST", APIURL+"/v1/userdetails", strings.NewReader(params.Encode()))
	req.SetBasicAuth(APIUser, APIPassword)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if dump, err := httputil.DumpRequestOut(req, true); err == nil {
		log.WithFields(map[string]interface{}{"tag": "usersync.GetUserDetails ->"}).Infof("%q", dump)
	}

	resp, err := client.Do(req)

	if err != nil {
		return
	}

	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		log.WithFields(map[string]interface{}{"tag": "usersync.GetUserDetails <-"}).Infof("%q", dump)
	}

	if resp.StatusCode != 200 {
		err = errors.New("Unable to retrieve user details from the main server - status code: " + strconv.Itoa(resp.StatusCode))
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return
	}

	data, err := gabs.ParseJSON(body)

	if err != nil {
		err = fmt.Errorf("Could not parse response from the main server - %s", err.Error())
		return
	}

	success, ok := data.S("success").Data().(bool)

	if !ok || !success {
		msg := "Could not get user details from the main server"

		if m, ok := data.S("message").Data().(string); ok {
			msg = m
		}

		err = errors.New(msg)
		return
	}

	if container, err := data.JSONPointer("/data/field_full_name/und/0/value"); err == nil {
		fullname = container.Data().(string)
	}

	if container, err := data.JSONPointer("/data/field_domain_url/und/0/value"); err == nil {
		domain = container.Data().(string)
	}

	if send, ok := data.S("sendemail").Data().(bool); ok {
		sendEmail = send
	}

	return
}
