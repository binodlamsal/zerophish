package auth

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/everycloud-technologies/phishing-simulation/encryption"

	"github.com/everycloud-technologies/phishing-simulation/bakery"

	ctx "github.com/everycloud-technologies/phishing-simulation/context"
	log "github.com/everycloud-technologies/phishing-simulation/logger"
	"github.com/everycloud-technologies/phishing-simulation/models"
	"github.com/everycloud-technologies/phishing-simulation/notifier"
	"github.com/everycloud-technologies/phishing-simulation/usersync"
	"github.com/everycloud-technologies/phishing-simulation/util"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

const SSODomain = ".everycloud.com"                                 // ".localhost"
const SSOMasterLoginURL = "https://www.everycloud.com/bakery/login" // "https://localhost:3333/sso/mock"

// LimitedAccessKey holds a combo of user id, ip and route prefix
type LimitedAccessKey struct {
	ID    int64  `json:"id"`
	IP    string `json:"ip"`
	Route string `json:"route"`
}

//init registers the necessary models to be saved in the session later
func init() {
	gob.Register(&models.User{})
	gob.Register(&models.Flash{})
	Store.Options.HttpOnly = true
	// This sets the maxAge to 5 days for all cookies
	Store.MaxAge(86400 * 5)
}

// Store contains the session information for the request
var Store = sessions.NewCookieStore(
	[]byte(securecookie.GenerateRandomKey(64)), //Signing key
	[]byte(securecookie.GenerateRandomKey(32)))

// ErrInvalidPassword is thrown when a user provides an incorrect password.
var ErrInvalidPassword = errors.New("Invalid Password")

// ErrEmptyPassword is thrown when a user provides a blank password to the register
// or change password functions
var ErrEmptyPassword = errors.New("Password cannot be blank")

// ErrPasswordMismatch is thrown when a user provides passwords that do not match
var ErrPasswordMismatch = errors.New("Passwords must match")

// ErrBadPassword is thrown when a user provides passwords that does not conform our password policy
var ErrBadPassword = errors.New("Password must be at least 8 chars long with at least 1 letter, 1 number and 1 special character")

// ErrUsernameTaken is thrown when a user attempts to register a username that is taken.
var ErrUsernameTaken = errors.New("Username already taken")

// ErrSyncUserData is thrown when something is wrong with synchronization of user data
var ErrSyncUserData = errors.New("Could not sync user details with the main server")

// ErrBadEmail is thrown when a user provides a malformed email address
var ErrBadEmail = errors.New("Incorrect e-mail address")

// ErrBadDomain is thrown when a user provides a malformed domain name
var ErrBadDomain = errors.New("Incorrect domain name")

// Login attempts to login the user given a request.
func Login(r *http.Request) (bool, models.User, error) {
	username, password := r.FormValue("username"), r.FormValue("password")
	u, err := models.GetUserByUsername(username)
	if err != nil {
		return false, models.User{}, err
	}
	//If we've made it here, we should have a valid user stored in u
	//Let's check the password
	err = bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(password))
	if err != nil {
		return false, models.User{}, ErrInvalidPassword
	}

	//update the user and set last login time
	u.LastLoginAt = time.Now().UTC()
	err = models.PutUser(&u)

	return true, u, nil
}

// Register attempts to register the user given a request.
func Register(r *http.Request) (bool, error) {
	username := r.FormValue("username")
	newEmail := r.FormValue("email")
	fullName := r.FormValue("full_name")
	newPassword := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")
	role := r.FormValue("roles")
	api := r.FormValue("api")

	rid, _ := strconv.ParseInt(role, 10, 0)

	u, err := models.GetUserByUsername(username)
	// If the given username already exists, throw an error and return false
	if err == nil {
		return false, ErrUsernameTaken
	}

	// If we have an error which is not simply indicating that no user was found, report it
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Warn(err)
		return false, err
	}

	u = models.User{}
	ur := models.UserRole{}
	// If we've made it here, we should have a valid username given
	// Check that the passsword isn't blank
	if newPassword == "" {
		return false, ErrEmptyPassword
	}
	// Make sure passwords match
	if newPassword != confirmPassword {
		return false, ErrPasswordMismatch
	}

	if !IsValidPassword(newPassword) {
		return false, ErrBadPassword
	}

	// Let's create the password hash
	h, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return false, err
	}
	u.Username = username
	u.Email = encryption.EncryptedString{newEmail}
	u.FullName = fullName
	u.Hash = string(h)
	u.ApiKey = util.GenerateSecureKey()
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = time.Now().UTC()

	if api != "1" {
		currentUser := ctx.Get(r, "user").(models.User)
		currentRole, err := models.GetUserRole(currentUser.Id)

		if err != nil {
			log.Error(err)
		}

		if currentRole.Is(models.Administrator) || currentRole.Is(models.Partner) {
			if rid == models.Customer || rid == models.ChildUser {
				u.Partner = ctx.Get(r, "user").(models.User).Id
			}
		} else if currentRole.Is(models.ChildUser) {
			if rid == models.Customer {
				u.Partner = currentUser.Partner
			}
		}
	}

	err = models.PutUser(&u)

	//Getting the inserted U after inserted
	iu, err := models.GetUserByUsername(username)

	ur.Uid = iu.Id
	ur.Rid = rid

	err = models.PutUserRole(&ur)

	if err != nil {
		return false, err
	}

	if api != "1" && os.Getenv("USERSYNC_DISABLE") == "" {
		uid, err := usersync.PushUser(
			iu.Id,
			iu.Username,
			iu.Email.String(),
			iu.FullName,
			newPassword,
			ur.Rid,
			models.GetUserBakeryID(iu.Partner),
			false,
		)

		if err != nil {
			_, _ = models.DeleteUser(iu.Id)
			return false, err
		}

		err = iu.SetBakeryUserID(uid)

		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func UpdateSettings(r *http.Request) error {
	shouldPushUpdates := false
	domainChanged := false
	u := ctx.Get(r, "user").(models.User)
	r.ParseForm() // Parses the request body

	if u.Email.String() != r.Form.Get("email") ||
		u.Username != r.Form.Get("username") {
		shouldPushUpdates = true
	}

	if u.Domain != r.Form.Get("domain") && r.Form.Get("domain") != "" {
		domainChanged = true
	}

	u.UpdatedAt = time.Now().UTC()
	u.Username = r.Form.Get("username")
	u.Email = encryption.EncryptedString{r.Form.Get("email")}
	u.FullName = r.Form.Get("full_name")
	u.Domain = r.Form.Get("domain")
	u.TimeZone = r.Form.Get("time_zone")
	u.NumOfUsers, _ = strconv.ParseInt(r.Form.Get("num_of_users"), 10, 0)
	u.AdminEmail = encryption.EncryptedString{r.Form.Get("admin_email")}
	newPassword := r.Form.Get("new_password")
	confirmPassword := r.Form.Get("confirm_password")

	role, err := models.GetUserRole(u.Id)

	if err != nil {
		return err
	}

	if !util.IsEmail(u.Email.String()) {
		return ErrBadEmail
	}

	if u.Domain != "" && u.Domain != "DELETE" &&
		(!util.IsValidDomain(u.Domain) || (domainChanged && !models.IsUniqueDomain(u.Domain))) {
		return ErrBadDomain
	}

	if newPassword != "" && confirmPassword != "" {
		if newPassword == "" {
			return ErrEmptyPassword
		}

		if newPassword != confirmPassword {
			return ErrPasswordMismatch
		}

		if !IsValidPassword(newPassword) {
			return ErrBadPassword
		}

		h, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)

		if err != nil {
			return err
		}

		u.Hash = string(h)
		shouldPushUpdates = true
	}

	if r.Form.Get("avatar") != "" {
		a := u.GetAvatar()

		if a == nil {
			a = &models.Avatar{UserId: u.Id, Data: r.Form.Get("avatar")}
		} else {
			if r.Form.Get("avatar") == "DELETE" {
				return models.DeleteAvatar(a.Id)
			}

			a.Data = r.Form.Get("avatar")
		}

		return models.PutAvatar(a)
	}

	if os.Getenv("USERSYNC_DISABLE") == "" && shouldPushUpdates {
		buid := models.GetUserBakeryID(u.Id)

		if err := usersync.UpdateUser(
			buid,
			u.Username,
			u.Email.String(),
			newPassword,
			role.Rid,
			models.GetUserBakeryID(u.Partner),
		); err != nil {
			return fmt.Errorf("Could not update user with bakery id %d - %s", buid, err.Error())
		}
	}

	return models.PutUser(&u)
}

func ChangeLogo(r *http.Request) error {
	u := ctx.Get(r, "user").(models.User)
	r.ParseForm()
	logo := r.Form.Get("logo")

	if logo == "" {
		return nil
	}

	l := u.GetLogo()

	if l == nil {
		l = &models.Logo{UserId: u.Id, Data: logo}
	} else {
		if logo == "DELETE" {
			return models.DeleteLogo(l.Id)
		}

		l.Data = logo
	}

	return models.PutLogo(l)
}

func UpdateSettingsByAdmin(r *http.Request) error {
	currentUser := ctx.Get(r, "user").(models.User)

	type Usersdata struct {
		Id                   int64     `json:"id"`
		Username             string    `json:"username"`
		FullName             string    `json:"full_name"`
		Email                string    `json:"email"`
		Domain               string    `json:"domain"`
		TimeZone             string    `json:"time_zone"`
		NumOfUsers           int64     `json:"num_of_users"`
		AdminEmail           string    `json:"admin_email"`
		New_password         string    `json:"new_password"`
		Confirm_new_password string    `json:"confirm_new_password"`
		Role                 int64     `json:"role"`
		Hash                 string    `json:"-"`
		ApiKey               string    `json:"api_key"`
		Partner              int64     `json:"partner"`
		PlanId               int64     `json:"plan_id"`
		ExpirationDate       time.Time `json:"expiration_date"`
	}

	var ud = new(Usersdata)
	err := json.NewDecoder(r.Body).Decode(&ud)

	newPassword := ud.New_password
	confirmPassword := ud.Confirm_new_password

	u, err := models.GetUser(ud.Id)

	if err != nil {
		return err
	}

	shouldPushUpdates := false
	domainChanged := false

	if u.Email.String() != ud.Email ||
		u.Username != ud.Username ||
		u.Partner != ud.Partner {
		shouldPushUpdates = true
	}

	if !util.IsEmail(ud.Email) {
		return ErrBadEmail
	}

	if u.Domain != ud.Domain && ud.Domain != "" {
		domainChanged = true
	}

	if ud.Domain != "" &&
		(!util.IsValidDomain(ud.Domain) || (domainChanged && !models.IsUniqueDomain(ud.Domain))) {
		return ErrBadDomain
	}

	u.Id = ud.Id
	u.Email = encryption.EncryptedString{ud.Email}
	u.Domain = ud.Domain
	u.TimeZone = ud.TimeZone
	u.NumOfUsers = ud.NumOfUsers
	u.AdminEmail = encryption.EncryptedString{ud.AdminEmail}
	u.Username = ud.Username
	u.FullName = ud.FullName
	u.ApiKey = ud.ApiKey
	u.Partner = ud.Partner
	u.UpdatedAt = time.Now().UTC()

	// Check the current password

	// Check that new passwords match  //since this is going to do by admin no longer need to check
	if newPassword != "" && confirmPassword != "" {

		// Check that the new password isn't blank
		if newPassword == "" {
			return ErrEmptyPassword
		}

		if newPassword != confirmPassword {
			return ErrPasswordMismatch
		}

		if !IsValidPassword(newPassword) {
			return ErrBadPassword
		}

		// Generate the new hash
		h, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Hash = string(h)
		shouldPushUpdates = true
	}

	// Unset partner for non-customers
	if ud.Role != models.Customer && ud.Role != models.ChildUser && ud.Role != models.LMSUser {
		u.Partner = 0

		if ud.Partner != 0 {
			shouldPushUpdates = true
		}
	}

	if role, err := models.GetUserRole(u.Id); err == nil {
		if role.Rid != ud.Role {
			shouldPushUpdates = true
		}
	}

	if os.Getenv("USERSYNC_DISABLE") == "" && shouldPushUpdates {
		buid := models.GetUserBakeryID(u.Id)

		if err := usersync.UpdateUser(
			buid,
			u.Username,
			u.Email.String(),
			newPassword,
			ud.Role,
			models.GetUserBakeryID(u.Partner),
		); err != nil {
			return fmt.Errorf("Could not update user with bakery id %d - %s", buid, err.Error())
		}
	}

	if err = models.PutUser(&u); err != nil {
		return err
	}

	ur := models.UserRole{}
	ur.Uid = ud.Id
	ur.Rid = ud.Role

	//first delete the users roles in update
	if err = models.DeleteUserRoles(ur.Uid); err != nil {
		return err
	}

	//Second save the user roles again
	err = models.PutUserRole(&ur)

	if currentUser.CanManageSubscriptions() {
		wasSubscribed := u.IsSubscribed()
		s := u.GetSubscription()

		if s != nil {
			if ud.PlanId != s.PlanId {
				if ud.PlanId != 0 {
					err = s.ChangePlan(ud.PlanId)

					if err != nil {
						return err
					}
				} else {
					err = models.DeleteSubscription(s.Id)

					if err != nil {
						return err
					}
				}
			}

			if ud.ExpirationDate != s.ExpirationDate && ud.PlanId != 0 {
				err = s.ChangeExpirationDate(ud.ExpirationDate)

				if err != nil {
					return err
				}
			}
		} else {
			if ud.PlanId != 0 {
				uid := u.Id

				if u.IsChildUser() {
					uid = u.Partner
				}

				subscription := &models.Subscription{
					UserId:         uid,
					PlanId:         ud.PlanId,
					ExpirationDate: ud.ExpirationDate,
				}

				err = models.PostSubscription(subscription)

				if err != nil {
					return err
				}
			}
		}

		if !wasSubscribed && u.IsSubscribed() {
			if os.Getenv("DONT_NOTIFY_USERS") == "" {
				notifier.SendAccountUpgradeEmail(u.Email.String(), u.FullName)
			}
		}
	}

	return nil
}

// IsValidPassword tells is the given password conforms to our password policy
func IsValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	if regexp.MustCompile(`\s`).MatchString(password) {
		return false
	}

	alphaMatches := regexp.MustCompile(`([a-zA-Z])`).FindStringSubmatch(password)
	numMatches := regexp.MustCompile(`([0-9])`).FindStringSubmatch(password)
	specialMatches := regexp.MustCompile(`([^a-zA-Z0-9\s])`).FindStringSubmatch(password)

	if len(alphaMatches) < 2 || len(numMatches) < 2 || len(specialMatches) < 2 {
		return false
	}

	return true
}

// GenerateLimitedAccessKey generates an encrypted access key limited to the given route prefix and IP.
// Will return empty string in case of any errors.
func GenerateLimitedAccessKey(id int64, ip, route string) string {
	keyBytes, err := json.Marshal(LimitedAccessKey{
		ID:    id,
		IP:    ip,
		Route: route,
	})

	if err != nil {
		log.Errorf("Could not serialize limited access key to JSON - %s", err.Error())
		return ""
	}

	key, err := bakery.Encrypt(string(keyBytes))

	if err != nil {
		log.Errorf("Could not encrypt limited access key - %s", err.Error())
		return ""
	}

	return key
}

// ParseLimitedAccessKey decrypts the given limited access key
func ParseLimitedAccessKey(key string) (*LimitedAccessKey, error) {
	keyJSON, err := bakery.Decrypt(key)

	if err != nil {
		return nil, fmt.Errorf("Could not decrypt limited access key - %s", err.Error())
	}

	var lak LimitedAccessKey

	if err := json.Unmarshal([]byte(keyJSON), &lak); err != nil {
		return nil, fmt.Errorf("Could not unserialize limited access key - %s", err.Error())
	}

	return &lak, nil
}

// IsValidForRequest tells if this limited access key is valid for the given request
func (lak *LimitedAccessKey) IsValidForRequest(r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, lak.Route) {
		log.Error("Invalid limited access key - route mismatch")
		return false
	}

	ip := r.RemoteAddr

	if ipAddr, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = ipAddr
	}

	if ip != lak.IP {
		log.Error("Invalid limited access key - IP mismatch")
		return false
	}

	return true
}
