package usersync

import (
	"errors"
	"time"

	"github.com/binodlamsal/gophish/auth"
	"github.com/binodlamsal/gophish/models"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

// CreateUser creates a new user with the given props and returns it
func CreateUser(username, email, password string, rid int64) (*models.User, error) {
	if username == "" {
		return nil, errors.New("Username must not be empty")
	}

	if email == "" {
		return nil, errors.New("E-mail must not be empty")
	}

	if password == "" {
		return nil, auth.ErrEmptyPassword
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
		ApiKey:    auth.GenerateSecureKey(),
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
