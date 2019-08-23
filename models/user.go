package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/everycloud-technologies/phishing-simulation/encryption"

	"github.com/everycloud-technologies/phishing-simulation/bakery"
	"github.com/everycloud-technologies/phishing-simulation/util"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"

	log "github.com/everycloud-technologies/phishing-simulation/logger"
)

// Roles
const (
	Administrator = 1
	Partner       = 2
	Customer      = 3
	ChildUser     = 4
	LMSUser       = 5
)

// User represents the user model for gophish.
type User struct {
	Id              int64                      `json:"id"`
	Username        string                     `json:"username" sql:"not null;unique"`
	Email           encryption.EncryptedString `json:"email" sql:"not null;unique"`
	Partner         int64                      `json:"partner" sql:"not null"`
	Hash            string                     `json:"-"`
	ApiKey          encryption.EncryptedString `json:"api_key" sql:"not null;unique"`
	PlainApiKey     string                     `json:"plain_api_key" gorm:"-"`
	FullName        string                     `json:"full_name" sql:"not null"`
	Domain          string                     `json:"domain"`
	TimeZone        string                     `json:"time_zone"`
	NumOfUsers      int64                      `json:"num_of_users"`
	AdminEmail      encryption.EncryptedString `json:"admin_email" sql:"not null"`
	EmailVerifiedAt time.Time                  `json:"email_verified_at"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
	LastLoginAt     time.Time                  `json:"last_login_at"`
	LastLoginIp     string                     `json:"last_login_ip" sql:"not null"`
	LastUserAgent   string                     `json:"last_user_agent"`
	ToBeDeleted     bool                       `json:"to_be_deleted"`
}

// BakeryUser stores relations between local user ids and bakery master user ids
type BakeryUser struct {
	UID       int64
	MasterUID int64
}

// Role represents the role model for gophish.
type Role struct {
	Rid         int64  `json:"rid"`
	Name        string `json:"name" sql:"not null;unique"`
	DisplayName string `json:"display_name" sql:"not null"`
	Weight      string `json:"weight" sql:"not null;unique"`
}

// UserRole represents the user role model for gophish.
type UserRole struct {
	Uid int64 `json:"uid"`
	Rid int64 `json:"rid" sql:"not"`
}

// Roles is a list of roles
type Roles []Role

// UserRoles is a list of user roles
type UserRoles []UserRole

func (bu BakeryUser) TableName() string {
	return "bakery_user"
}

// AvailableFor returns roles which a user with the given role can create users with
func (roles Roles) AvailableFor(role UserRole) Roles {
	if role.Is(Administrator) {
		return roles
	}

	if role.Is(Partner) {
		return Roles{
			Role{Rid: 3, Name: "Customer", Weight: "2"},
			Role{Rid: 4, Name: "Child User", Weight: "3"},
			Role{Rid: 5, Name: "LMS User", Weight: "4"},
		}
	}

	if role.Is(ChildUser) {
		return Roles{
			Role{Rid: 3, Name: "Customer", Weight: "2"},
			Role{Rid: 5, Name: "LMS User", Weight: "4"},
		}
	}

	return Roles{}
}

// TableName specifies the database tablename for Gorm to use
func (ur UserRole) TableName() string {
	return "users_role"
}

// Name returns this role's name
func (ur UserRole) Name() string {
	role := Role{}
	err := db.Where("rid = ?", ur.Rid).First(&role).Error

	if err != nil {
		return "Unknown"
	}

	return role.Name
}

// DisplayName returns this role's display name
func (ur UserRole) DisplayName() string {
	role := Role{}
	err := db.Where("rid = ?", ur.Rid).First(&role).Error

	if err != nil {
		return "Unknown"
	}

	return role.DisplayName
}

// Is tells if this role id matches the given one
func (r Role) Is(rid int64) bool {
	return r.Rid == rid
}

// Is tells if this user role id matches the given one
func (ur UserRole) Is(rid int64) bool {
	return ur.Rid == rid
}

// IsOneOf tells if this user role id is among the given role ids
func (ur UserRole) IsOneOf(rids []int64) bool {
	for _, rid := range rids {
		if ur.Rid == rid {
			return true
		}
	}

	return false
}

// GetUser returns the user that the given id corresponds to. If no user is found, an
// error is thrown.
func GetUser(id int64) (User, error) {
	u := User{}
	err := db.Where("id=?", id).First(&u).Error
	return u, err
}

// GetUserByBakeryID returns the user that the given bakery user id corresponds to.
// If no user is found, an error is returned.
func GetUserByBakeryID(buid int64) (User, error) {
	u := User{}
	bu := BakeryUser{}
	err := db.Where("master_uid = ?", buid).First(&bu).Error

	if err != nil {
		return u, err
	}

	err = db.Where("id=?", bu.UID).First(&u).Error
	return u, err
}

// GetLMSUser returns a user with LMS role and given email
func GetLMSUser(email string) (User, error) {
	u := User{}
	u, err := GetUserByUsername(email)

	if err != nil {
		return u, err
	}

	ur, err := GetUserRole(u.Id)

	if err != nil {
		return u, err
	}

	if ur.Is(LMSUser) {
		return u, nil
	}

	return u, errors.New("Not LMS user")
}

// GetUserByAPIKey returns the user that the given API Key corresponds to. If no user is found, an
// error is thrown.
func GetUserByAPIKey(key string) (User, error) {
	u := User{}
	err := db.Where("api_key = ?", encryption.EncryptedString{key}).First(&u).Error
	return u, err
}

// GetUserByUsername returns the user that the given username corresponds to. If no user is found, an
// error is thrown.
func GetUserByUsername(username string) (User, error) {
	u := User{}

	err := db.
		Where("username = ?", username).
		Or("email = ?", encryption.EncryptedString{username}).
		First(&u).
		Error

	return u, err
}

// GetUserByDomain returns the user with the given domain
func GetUserByDomain(domain string) (User, error) {
	u := User{}
	err := db.Where("domain = ?", domain).First(&u).Error
	return u, err
}

// PutUser updates the given user
func PutUser(u *User) error {
	err := db.Save(u).Error
	return err
}

// UpdateUser update the given user (only non-empty fields will be updated)
func UpdateUser(u *User) error {
	return db.Model(u).UpdateColumns(u).Error
}

// PutRole updates role
func PutRole(r *Role) error {
	err := db.Save(r).Error
	return err
}

// PutUserRole updates role of the given user
func PutUserRole(ur *UserRole) error {
	err := db.Save(ur).Error
	return err
}

// SetUserRole sets role of the given uid to the given rid
func SetUserRole(uid, rid int64) error {
	return db.Model(UserRole{}).Where("uid = ?", uid).Update("rid", rid).Error
}

// CreateUser creates a new user with the given props and returns it
func CreateUser(username, fullName, email, password string, rid int64, partner int64) (*User, error) {
	if username == "" {
		return nil, errors.New("Username must not be empty")
	}

	if email == "" {
		return nil, errors.New("E-mail must not be empty")
	}

	if !util.IsEmail(email) {
		return nil, errors.New("E-mail must be valid")
	}

	if password == "" {
		return nil, errors.New("Password must not be empty")
	}

	if rid < 1 || rid > 5 {
		return nil, errors.New("Please select the role from the dropdown")
	}

	_, err1 := GetUserByUsername(username)
	_, err2 := GetUserByUsername(email)

	if err1 == nil || err2 == nil {
		return nil, fmt.Errorf("Username (%s) or e-mail (%s) is already taken", username, email)
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

	u := User{
		Username:  username,
		FullName:  fullName,
		Email:     encryption.EncryptedString{email},
		Hash:      string(h),
		ApiKey:    encryption.EncryptedString{util.GenerateSecureKey()},
		Partner:   partner,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err = PutUser(&u); err != nil {
		return nil, err
	}

	err = PutUserRole(&UserRole{
		Uid: u.Id,
		Rid: rid,
	})

	if err != nil {
		return nil, err
	}

	return &u, nil
}

// GetUsers returns the users owned by the given user.
func GetUsers(uid int64) ([]User, error) {
	users := []User{}
	role, err := GetUserRole(uid)

	if err != nil {
		return users, err
	}

	if role.Is(Administrator) {
		err = db.Order("id asc").Find(&users).Error
	} else if role.Is(Partner) {
		cuids, err := GetDirectCustomerIds(uid)

		if err != nil {
			return users, err
		}

		err = db.
			Joins("LEFT JOIN targets ON targets.email=users.email").
			Joins("LEFT JOIN users_role ON users_role.uid=users.id").
			Joins("LEFT JOIN group_targets ON group_targets.target_id=targets.id").
			Joins("LEFT JOIN groups ON groups.id=group_targets.group_id").
			Where("(partner = ? OR partner IN (?) OR groups.user_id = ?) AND users_role.rid IN (?)", uid, cuids, uid, []int{ChildUser, Customer, LMSUser}).
			Group("users.id").Order("id asc").Find(&users).Error
	} else if role.Is(Customer) {
		err = db.
			Joins("LEFT JOIN targets ON targets.email=users.email").
			Joins("LEFT JOIN users_role ON users_role.uid=users.id").
			Joins("LEFT JOIN group_targets ON group_targets.target_id=targets.id").
			Joins("LEFT JOIN groups ON groups.id=group_targets.group_id").
			Where("(partner = ? OR groups.user_id = ?) AND users_role.rid = ?", uid, uid, LMSUser).
			Group("users.id").Order("id asc").Find(&users).Error
	} else if role.Is(ChildUser) {
		user, err := GetUser(uid)

		if err != nil {
			return users, err
		}

		cuids, err := GetDirectCustomerIds(user.Partner)

		if err != nil {
			return users, err
		}

		err = db.
			Joins("LEFT JOIN targets ON targets.email=users.email").
			Joins("LEFT JOIN users_role ON users_role.uid=users.id").
			Joins("LEFT JOIN group_targets ON group_targets.target_id=targets.id").
			Joins("LEFT JOIN groups ON groups.id=group_targets.group_id").
			Where(
				"(users.id = ? OR users.partner = ? OR users.partner IN (?) OR groups.user_id = ?) AND users.id <> ? AND users_role.rid IN (?)",
				user.Partner, user.Partner, cuids, user.Partner, uid, []int{Partner, ChildUser, Customer, LMSUser},
			).
			Group("users.id").Order("users.id asc").Find(&users).Error
	}

	return users, err
}

// GetUserIds returns the user ids owned by the given user.
func GetUserIds(uid int64) ([]int64, error) {
	uids := []int64{}
	users, err := GetUsers(uid)

	if err != nil {
		return uids, err
	}

	for _, u := range users {
		uids = append(uids, u.Id)
	}

	return uids, err
}

// GetChildUserIds returns user ids of all child users bound to the given uid
func GetChildUserIds(uid int64) ([]int64, error) {
	var ids []int64

	if uid == 0 {
		return ids, nil
	}

	err := db.
		Model(&User{}).
		Joins("LEFT JOIN users_role ON users_role.uid=users.id").
		Where("users_role.rid=? AND users.partner=?", ChildUser, uid).
		Pluck("id", &ids).Error

	return ids, err
}

// GetDirectCustomerIds returns user ids of all customers directly bound to the given uid
func GetDirectCustomerIds(uid int64) ([]int64, error) {
	var ids []int64

	if uid == 0 {
		return ids, nil
	}

	err = db.
		Model(&User{}).
		Joins("LEFT JOIN users_role ON users_role.uid=users.id").
		Where("users_role.rid = ? AND users.partner = ?", Customer, uid).
		Pluck("id", &ids).Error

	return ids, err
}

// GetCustomerIds returns user ids of customers directly or indirectly related to the given uid
func GetCustomerIds(uid int64) ([]int64, error) {
	var ids []int64
	role, err := GetUserRole(uid)

	if err != nil {
		return ids, err
	}

	if role.Is(Administrator) {
		err = db.
			Model(&User{}).
			Where("id <> ?", uid).
			Pluck("id", &ids).Error
	} else if role.Is(Partner) {
		err = db.
			Model(&User{}).
			Joins("LEFT JOIN users_role ON users_role.uid=users.id").
			Where("users_role.rid = ? AND users.partner = ?", Customer, uid).
			Pluck("id", &ids).Error
	} else if role.Is(ChildUser) {
		u, err := GetUser(uid)

		if err != nil {
			return ids, err
		}

		err = db.
			Model(&User{}).
			Joins("LEFT JOIN users_role ON users_role.uid=users.id").
			Where("users_role.rid = ? AND users.partner = ?", Customer, u.Partner).
			Pluck("id", &ids).Error
	}

	return ids, err
}

// IsAdministrator tells if this user is administrator
func (u User) IsAdministrator() bool {
	role, err := GetUserRole(u.Id)

	if err != nil {
		return false
	}

	return role.Is(Administrator)
}

// IsPartner tells if this user is partner
func (u User) IsPartner() bool {
	role, err := GetUserRole(u.Id)

	if err != nil {
		return false
	}

	return role.Is(Partner)
}

// IsCustomer tells if this user is customer
func (u User) IsCustomer() bool {
	role, err := GetUserRole(u.Id)

	if err != nil {
		return false
	}

	return role.Is(Customer)
}

// IsChildUser tells if this user is child user
func (u User) IsChildUser() bool {
	role, err := GetUserRole(u.Id)

	if err != nil {
		return false
	}

	return role.Is(ChildUser)
}

// IsLMSUser tells if this user is LMS user
func (u User) IsLMSUser() bool {
	role, err := GetUserRole(u.Id)

	if err != nil {
		return false
	}

	return role.Is(LMSUser)
}

// GetLogo returns logo which was assigned to this partner/customer account or nil
func (u User) GetLogo() *Logo {
	l := Logo{}

	if u.IsPartner() {
		if db.Where("user_id = ?", u.Id).First(&l).Error == nil {
			return &l
		}
	} else if u.IsChildUser() || u.IsCustomer() {
		if db.Where("user_id = ?", u.Partner).First(&l).Error == nil {
			return &l
		}
	}

	return nil
}

// GetAvatar returns Avatar which was assigned to this user or nil
func (u User) GetAvatar() *Avatar {
	if a, found := GetCache().GetUserAvatar(u.Id); found {
		return a
	}

	a := Avatar{}

	if db.Table("avatars").Where("user_id = ?", u.Id).First(&a).Error == nil {
		GetCache().AddUserAvatar(&a)
		return &a
	}

	GetCache().AddEntry("user", u.Id, "avatar", nil)
	return nil
}

// GetSubscription returns user subscription or nil if there is none
func (u User) GetSubscription() *Subscription {
	s := Subscription{}
	uid := u.Id

	if u.IsChildUser() {
		uid = u.Partner
	}

	if s, found := GetCache().GetUserSubscription(uid); found {
		return s
	}

	if db.Where("user_id = ?", uid).First(&s).Error == nil {
		GetCache().AddUserSubscription(&s)
		return &s
	}

	GetCache().AddEntry("user", uid, "subscription", nil)
	return nil
}

// IsSubscribed tells if this user is subscribed to a plan and the subscription is not expired
func (u User) IsSubscribed() bool {
	s := u.GetSubscription()

	if s != nil {
		if u.IsCustomer() {
			partner, err := GetUser(u.Partner)

			if err != nil {
				return s.IsActive()
			}

			if partner.IsAdministrator() {
				return s.IsActive()
			}

			if ps := partner.GetSubscription(); ps != nil {
				return ps.IsActive() && s.IsActive()
			}

			return false
		}

		return s.IsActive()
	}

	return false
}

// CanCreateCampaign tells if this user is allowed to create a campaign,
// the decision is made based on user's subscription status and plan
func (u User) CanCreateCampaign() bool {
	if u.IsAdministrator() || u.IsSubscribed() {
		return true
	}

	uids, err := GetUserIds(u.Id)

	if err != nil {
		log.Errorf("Could not find users related to uid %d - %s", u.Id, err.Error())
		return false
	}

	uids = append(uids, u.Id)
	var count int64
	db.Model(&Campaign{}).Where("user_id IN (?)", uids).Count(&count)
	return count < 1
}

// CanCreateGroup tells if this user is allowed to create a group (of target users),
// the decision is made based on user's subscription status and plan
func (u User) CanCreateGroup() bool {
	if u.IsAdministrator() || u.IsSubscribed() {
		return true
	}

	uids, err := GetUserIds(u.Id)

	if err != nil {
		log.Errorf("Could not find users related to uid %d - %s", u.Id, err.Error())
		return false
	}

	uids = append(uids, u.Id)
	var count int64
	db.Model(&Group{}).Where("user_id IN (?)", uids).Count(&count)
	return count < 1
}

// CanManageSubscriptions tells if this user is allowed to manage customers' subscriptions,
// the decision is made based on user's subscription status and plan
func (u User) CanManageSubscriptions() bool {
	if u.IsAdministrator() || ((u.IsPartner() || u.IsChildUser()) && u.IsSubscribed()) {
		return true
	}

	return false
}

// CanManageUserWithId tells if this user can update/delete/impersonate a user with the given uid
func (u User) CanManageUserWithId(uid int64) bool {
	uids, err := GetUserIds(u.Id)

	if err != nil {
		log.Errorf("Could not find user ids owned by user %s: %s", u.Username, err.Error())
		return false
	}

	for _, id := range uids {
		if uid == id {
			return true
		}
	}

	return false
}

// HasTemplates tells if this user owns any email templates
func (u User) HasTemplates() bool {
	var count int64
	if u.IsChildUser() {
		db.Model(&Template{}).Where("user_id = ? OR user_id = ?", u.Id, u.Partner).Count(&count)
	} else {
		db.Model(&Template{}).Where("user_id = ?", u.Id).Count(&count)
	}

	return count > 0
}

// HasPages tells if this user owns any landing pages
func (u User) HasPages() bool {
	var count int64

	if u.IsChildUser() {
		db.Model(&Page{}).Where("user_id = ? OR user_id = ?", u.Id, u.Partner).Count(&count)
	} else {
		db.Model(&Page{}).Where("user_id = ?", u.Id).Count(&count)
	}

	return count > 0
}

// SetBakeryUserID assigns bakery master user id to this user
func (u User) SetBakeryUserID(id int64) error {
	bu := BakeryUser{}

	if err := db.Where("uid = ?", u.Id).First(&bu).Error; err == nil {
		bu.MasterUID = id

		if err = db.Save(&bu).Error; err != nil {
			return fmt.Errorf("Could not update bakery master user id for user with id %d - %s", u.Id, err.Error())
		}

		return nil
	}

	bu.UID = u.Id
	bu.MasterUID = id

	if err = db.Save(&bu).Error; err != nil {
		return fmt.Errorf("Could not set bakery master user id for user with id %d - %s", u.Id, err.Error())
	}

	return nil
}

// DecryptApiKey decrypts encrypted ApiKey field and puts the result into PlainApiKey
func (u *User) DecryptApiKey() {
	u.PlainApiKey = u.ApiKey.String()
}

// GetUserBakeryID returns bakery user id associated with the given uid or 0 if not found
func GetUserBakeryID(uid int64) int64 {
	bu := BakeryUser{}

	if err := db.Where("uid = ?", uid).First(&bu).Error; err == nil {
		return bu.MasterUID
	}

	return 0
}

// GetRoles returns all available roles
func GetRoles() (Roles, error) {
	r := Roles{}
	err := db.Order("rid asc").Find(&r).Error
	return r, err
}

// GetUserRole returns a role assigned to the given uid
func GetUserRole(uid int64) (UserRole, error) {
	if r, found := GetCache().GetUserRole(uid); found {
		return *r, nil
	}

	r := UserRole{}
	err := db.Where("uid = ?", uid).First(&r).Error

	if err == nil {
		GetCache().AddUserRole(&r)
	}

	return r, err
}

// DeleteUserRoles deletes all roles of a given uid
func DeleteUserRoles(uid int64) error {
	GetCache().DeleteEntry("user", uid, "role")
	err = db.Delete(UserRole{}, "uid = ?", uid).Error
	return err
}

// DeleteUserSubscriptions deletes all subscriptions of a given uid
func DeleteUserSubscriptions(uid int64) error {
	GetCache().DeleteEntry("user", uid, "subscription")
	err = db.Delete(Subscription{}, "user_id = ?", uid).Error
	return err
}

// DeleteUserAvatar deletes avatar of a given uid
func DeleteUserAvatar(uid int64) error {
	GetCache().DeleteEntry("user", uid, "avatar")
	err = db.Delete(Avatar{}, "user_id = ?", uid).Error
	return err
}

// DeleteUserBakeryID deletes bakery master user id for the given uid
func DeleteUserBakeryID(uid int64) error {
	err = db.Delete(BakeryUser{}, "uid = ?", uid).Error
	return err
}

// DeleteUser deletes the specified user and in case of success returns a
// bakery user id associated with the given uid or 0 if there's no such id.
func DeleteUser(uid int64) (int64, error) {
	if err := db.Delete(&User{Id: uid}).Error; err != nil {
		return 0, err
	}

	buid := GetUserBakeryID(uid)

	_ = DeleteUserSubscriptions(uid)
	_ = DeleteUserAvatar(uid)
	_ = DeleteUserRoles(uid)
	_ = DeleteUserBakeryID(uid)

	if err = DeleteUserCampaigns(uid); err != nil {
		log.Error(err)
	}

	if err = DeleteUserEmailRequests(uid); err != nil {
		log.Error(err)
	}

	if err = DeleteUserGroups(uid); err != nil {
		log.Error(err)
	}

	if err = DeleteUserLogo(uid); err != nil {
		log.Error(err)
	}

	if err = DeleteUserPages(uid); err != nil {
		log.Error(err)
	}

	if err = DeleteUserTemplates(uid); err != nil {
		log.Error(err)
	}

	if err = DeleteChildUsers(uid); err != nil {
		log.Error(err)
	}

	if err = DeleteCustomers(uid); err != nil {
		log.Error(err)
	}

	return buid, nil
}

// DeleteChildUsers deletes all child users of a user with the given uid
func DeleteChildUsers(uid int64) error {
	cuids, err := GetChildUserIds(uid)

	if err != nil {
		return fmt.Errorf("Couldn't find ids of child users of user with id %d - %s", uid, err.Error())
	}

	var errs []error

	for _, cuid := range cuids {
		_, err = DeleteUser(cuid)

		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf(
			"Couldn't delete %d child user(s) of user with id %d",
			len(errs), uid,
		)
	}

	return nil
}

// DeleteCustomers deletes all customers of a user with the given uid
func DeleteCustomers(uid int64) error {
	cids, err := GetDirectCustomerIds(uid)

	if err != nil {
		return fmt.Errorf("Couldn't find ids of customers of user with id %d - %s", uid, err.Error())
	}

	var errs []error

	for _, cid := range cids {
		_, err = DeleteUser(cid)

		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf(
			"Couldn't delete %d customer(s) of user with id %d",
			len(errs), uid,
		)
	}

	return nil
}

// GetUsersByRole returns all users of the given role (rid)
func GetUsersByRoleID(rid int64) ([]User, error) {
	u := []User{}
	err = db.Raw("SELECT * FROM users u LEFT JOIN users_role ur ON (u.id = ur.uid) where ur.rid = ?", rid).Scan(&u).Error
	return u, err
}

func (u *User) BeforeSave() (err error) {
	if u.EmailVerifiedAt.IsZero() {
		u.EmailVerifiedAt = time.Now().UTC()
	}

	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now().UTC()
	}

	return
}

func (u *User) BeforeUpdate(scope *gorm.Scope) error {
	if u.Domain == "DELETE" {
		err := scope.SetColumn("Domain", "")

		if err != nil {
			return err
		}
	}

	return nil
}

// IsUniqueDomain tells if the given domain is unique among all other domains stored in the db
func IsUniqueDomain(domain string) bool {
	var unique bool
	var count int64
	_ = db.Table("users").Where("domain = ?", domain).Count(&count).Error

	if count == 0 {
		unique = true
	}

	return unique
}

func EncryptApiKeys() {
	log.Info("Encrypting API keys...")
	users := []User{}
	err := db.Find(&users).Error

	if err != nil {
		log.Error(err)
		return
	}

	for _, u := range users {
		apikey := u.ApiKey.String()

		if apikey == "" {
			continue
		}

		if len(apikey) > 64 {
			apikey, err := bakery.Decrypt(apikey)

			if err != nil {
				log.Error(err)
				continue
			}

			u.ApiKey = encryption.EncryptedString{apikey}
		}

		err := db.Save(&u).Error

		if err != nil {
			log.Error(err)
		} else {
			log.Infof("Encrypted API Key (%s) of user %d (%s)", u.ApiKey, u.Id, u.Username)
		}
	}

	log.Info("Done.")
}

// DecryptApiKeys decrypts api_key column in users table
func DecryptApiKeys() {
	log.Info("Decrypting api_key column in users table...")

	type user struct {
		ID     int64                      `json:"id"`
		ApiKey encryption.EncryptedString `json:"api_key" sql:"not null;unique"`
	}

	users := []user{}

	err := db.
		Table("users").
		Find(&users).
		Error

	if err != nil {
		log.Error(err)
		return
	}

	encryption.Disabled = true

	for _, u := range users {
		err = db.
			Table("users").
			Where("id = ?", u.ID).
			UpdateColumns(u).
			Error

		if err != nil {
			log.Error(err)
			return
		}

		log.Infof("Decrypted api_key column of user with id %d", u.ID)
	}

	encryption.Disabled = false
	log.Info("Done.")
}

// EncryptUserEmails encrypts email and admin_email columns in users table
func EncryptUserEmails() {
	log.Info("Encrypting emails in users table...")

	type user struct {
		ID         int64  `json:"id"`
		Email      string `json:"email" sql:"not null;unique"`
		AdminEmail string `json:"admin_email" sql:"not null;unique"`
	}

	users := []user{}

	err := db.
		Table("users").
		Where(`email LIKE "%@%" OR admin_email LIKE "%@%"`).
		Find(&users).
		Error

	if err != nil {
		log.Error(err)
		return
	}

	for _, u := range users {
		email, err := encryption.Encrypt(u.Email)

		if err != nil {
			log.Error(err)
			return
		}

		adminEmail, err := encryption.Encrypt(u.AdminEmail)

		if err != nil {
			log.Error(err)
			return
		}

		err = db.
			Table("users").
			Where("id = ?", u.ID).
			UpdateColumns(user{Email: email, AdminEmail: adminEmail}).
			Error

		if err != nil {
			log.Error(err)
			return
		}

		log.Infof("Encrypted emails of user with id %d", u.ID)
	}

	log.Info("Done.")
}

// DecryptUserEmails decrypts email and admin_email columns in users table
func DecryptUserEmails() {
	log.Info("Decrypting emails in users table...")

	type user struct {
		ID         int64                      `json:"id"`
		Email      encryption.EncryptedString `json:"email" sql:"not null;unique"`
		AdminEmail encryption.EncryptedString `json:"admin_email" sql:"not null;unique"`
	}

	users := []user{}

	err := db.
		Table("users").
		Find(&users).
		Error

	if err != nil {
		log.Error(err)
		return
	}

	encryption.Disabled = true

	for _, u := range users {
		err = db.
			Table("users").
			Where("id = ?", u.ID).
			UpdateColumns(u).
			Error

		if err != nil {
			log.Error(err)
			return
		}

		log.Infof("Decrypted emails of user with id %d", u.ID)
	}

	encryption.Disabled = false
	log.Info("Done.")
}
