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
	"time"

	"github.com/binodlamsal/gophish/models"
	"github.com/binodlamsal/gophish/util"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

// APIURL is a URL of the user sync API
var APIURL = "https://www.everycloudtech.com/api/bakery"

// APIUser is a username used during authentication
var APIUser = os.Getenv("USERSYNC_API_USER")

// APIPassword is a password used during authentication
var APIPassword = os.Getenv("USERSYNC_API_PASSWORD")

// CreateUser creates a new user with the given props and returns it
func CreateUser(username, email, password string, rid int64) (*models.User, error) {
	if username == "" {
		return nil, errors.New("Username must not be empty")
	}

	if email == "" {
		return nil, errors.New("E-mail must not be empty")
	}

	if password == "" {
		return nil, errors.New("Password must not be empty")
	}

	if rid < 1 || rid > 5 {
		return nil, errors.New("Role ID (rid) must be in range: 1-5")
	}

	_, err1 := models.GetUserByUsername(username)
	_, err2 := models.GetUserByUsername(email)

	if err1 == nil || err2 == nil {
		return nil, errors.New("Username or e-mail is already taken")
	}

	if err1 != nil && err1 != gorm.ErrRecordNotFound {
		return nil, err1
	}

	if err2 != nil && err2 != gorm.ErrRecordNotFound {
		return nil, err2
	}

	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return nil, err
	}

	u := models.User{
		Username:  username,
		Email:     email,
		Hash:      string(h),
		ApiKey:    util.GenerateSecureKey(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err = models.PutUser(&u); err != nil {
		return nil, err
	}

	// u, _ = models.GetUserByUsername(username)

	err = models.PutUserRole(&models.UserRole{
		Uid: u.Id,
		Rid: rid,
	})

	if err != nil {
		return nil, err
	}

	return &u, nil
}

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

// UpdateUser updates a user identified by either id or username or email.
// If for example the id is supplied then the username and/or email + password can be updated
// and if the username is supplied then email + password can be updated.
func UpdateUser(id uint64, username, email, password string) (*models.User, error) {
	// u := models.User{}

	// if id != 0 {
	// 	u, err := models.GetUser(id)

	// 	if err != nil {
	// 		return nil, errors.New(fmt.Sprintf("Could not find user with id %d", id))
	// 	}
	// } else if username != "" {
	// 	u, err := models.GetUserByUsername(username)

	// 	if err != nil {
	// 		return nil, errors.New(fmt.Sprintf("Could not find user with username %s", username))
	// 	}
	// } else if email != "" {
	// 	u, err := models.GetUserByUsername(email)

	// 	if err != nil {
	// 		return nil, errors.New(fmt.Sprintf("Could not find user with e-mail %s", email))
	// 	}
	// } else {
	// 	return nil, errors.New("")
	// }

	return nil, nil
}
