package models

import (
	"errors"
	"fmt"
	"time"

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
	Id              int64     `json:"id"`
	Username        string    `json:"username" sql:"not null;unique"`
	Email           string    `json:"email" sql:"not null;unique"`
	Partner         int64     `json:"partner" sql:"not null"`
	Hash            string    `json:"-"`
	ApiKey          string    `json:"api_key" sql:"not null;unique"`
	PlainApiKey     string    `json:"plain_api_key" gorm:"-"`
	FullName        string    `json:"full_name" sql:"not null"`
	Domain          string    `json:"domain"`
	TimeZone        string    `json:"time_zone"`
	EmailVerifiedAt time.Time `json:"email_verified_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	LastLoginAt     time.Time `json:"last_login_at"`
	LastLoginIp     string    `json:"last_login_ip" sql:"not null"`
	LastUserAgent   string    `json:"last_user_agent"`
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
		}
	}

	if role.Is(ChildUser) {
		return Roles{
			Role{Rid: 3, Name: "Customer", Weight: "2"},
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

	if key, err = bakery.Encrypt(key); err != nil {
		return u, err
	}

	err := db.Where("api_key = ?", key).First(&u).Error
	return u, err
}

// GetUserByUsername returns the user that the given username corresponds to. If no user is found, an
// error is thrown.
func GetUserByUsername(username string) (User, error) {
	u := User{}
	err := db.Where("username = ?", username).Or("email = ?", username).First(&u).Error
	return u, err
}

// PutUser updates the given user
func PutUser(u *User) error {
	err := db.Save(u).Error
	return err
}

// UpdateUser update the given user (only non-empty fields will be updated)
func UpdateUser(u *User) error {
	return db.Model(u).Updates(u).Error
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

	if password == "" {
		return nil, errors.New("Password must not be empty")
	}

	if rid < 1 || rid > 5 {
		return nil, errors.New("Role ID (rid) must be in range: 1-5")
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
		Email:     email,
		Hash:      string(h),
		ApiKey:    util.GenerateSecureKey(),
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
		err = db.
			Joins("LEFT JOIN targets ON targets.email=users.email").
			Joins("LEFT JOIN users_role ON users_role.uid=users.id").
			Joins("LEFT JOIN group_targets ON group_targets.target_id=targets.id").
			Joins("LEFT JOIN groups ON groups.id=group_targets.group_id").
			Where("(partner = ? OR groups.user_id = ?) AND users_role.rid IN (?)", uid, uid, []int{ChildUser, Customer, LMSUser}).
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

		err = db.
			Joins("LEFT JOIN targets ON targets.email=users.email").
			Joins("LEFT JOIN users_role ON users_role.uid=users.id").
			Joins("LEFT JOIN group_targets ON group_targets.target_id=targets.id").
			Joins("LEFT JOIN groups ON groups.id=group_targets.group_id").
			Where(
				"(users.id = ? OR users.partner = ? OR groups.user_id = ?) AND users.id <> ? AND users_role.rid IN (?)",
				user.Partner, user.Partner, user.Partner, uid, []int{Partner, ChildUser, Customer, LMSUser},
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

	// err := db.Raw(
	// 	`SELECT users.id FROM users LEFT JOIN users_role ON users_role.uid=users.id
	// 	WHERE users_role.rid=? AND users.partner=?`,
	// 	ChildUser, uid).Scan(&ids).Error

	err := db.
		Model(&User{}).
		Joins("LEFT JOIN users_role ON users_role.uid=users.id").
		Where("users_role.rid=? AND users.partner=?", ChildUser, uid).
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
	a := Avatar{}

	if db.Table("avatars").Where("user_id = ?", u.Id).First(&a).Error == nil {
		return &a
	}

	return nil
}

// GetSubscription returns user subscription or nil if there is none
func (u User) GetSubscription() *Subscription {
	s := Subscription{}
	uid := u.Id

	if u.IsChildUser() {
		uid = u.Partner
	}

	if db.Where("user_id = ?", uid).First(&s).Error == nil {
		return &s
	}

	return nil
}

// IsSubscribed tells if this user is subscribed to a plan and the subscription is not expired
func (u User) IsSubscribed() bool {
	s := u.GetSubscription()

	if s != nil {
		if u.IsCustomer() {
			partner, err := GetUser(u.Partner)

			if err != nil {
				log.Errorf("Could not determine partner account of customer with id %d", u.Id)
				return false
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
	var count int64
	db.Model(&Campaign{}).Where("user_id = ?", u.Id).Count(&count)

	if u.IsAdministrator() || u.IsSubscribed() || count < 1 {
		return true
	}

	return false
}

// CanCreateGroup tells if this user is allowed to create a group (of target users),
// the decision is made based on user's subscription status and plan
func (u User) CanCreateGroup() bool {
	var count int64
	db.Model(&Group{}).Where("user_id = ?", u.Id).Count(&count)

	if u.IsAdministrator() || u.IsSubscribed() || count < 1 {
		return true
	}

	return false
}

// CanHaveXTargetsInAGroup tells if this user is allowed to have X targets in a group,
// the decision is made based on user's subscription status and plan
func (u User) CanHaveXTargetsInAGroup(targets int) bool {
	if u.IsAdministrator() || u.IsSubscribed() || targets <= 150 {
		return true
	}

	return false
}

// CanManageSubscriptions tells if this user is allowed to manage customers' subscriptions,
// the decision is made based on user's subscription status and plan
func (u User) CanManageSubscriptions() bool {
	if u.IsAdministrator() || ((u.IsPartner() || u.IsChildUser()) && u.IsSubscribed()) {
		return true
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
	defer func() {
		if err := recover(); err != nil {
			log.Error("Recovered from panic in DecryptApiKey()")
			u.PlainApiKey = u.ApiKey
		}
	}()

	apiKey, err := bakery.Decrypt(u.ApiKey)

	if err != nil {
		log.Error(err)
		u.PlainApiKey = u.ApiKey
		return
	}

	u.PlainApiKey = apiKey
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
	r := UserRole{}
	err := db.Where("uid = ?", uid).First(&r).Error
	return r, err
}

// DeleteUserRoles deletes all roles of a given uid
func DeleteUserRoles(uid int64) error {
	err = db.Delete(UserRole{}, "uid = ?", uid).Error
	return err
}

// DeleteUserSubscriptions deletes all subscriptions of a given uid
func DeleteUserSubscriptions(uid int64) error {
	err = db.Delete(Subscription{}, "user_id = ?", uid).Error
	return err
}

// DeleteUserAvatar deletes avatar of a given uid
func DeleteUserAvatar(uid int64) error {
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

	return buid, nil
}

// GetUserPartners returns all the partners from the database
func GetUserPartners() ([]User, error) {
	u := []User{}
	err = db.Raw("SELECT * FROM users u LEFT JOIN users_role ur ON (u.id = ur.uid) where ur.rid in (?, ?)", 1, 2).Scan(&u).Error
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

func (u *User) BeforeCreate(scope *gorm.Scope) error {
	encKey, err := bakery.Encrypt(u.ApiKey)

	if err != nil {
		return err
	}

	return scope.SetColumn("ApiKey", encKey)
}

func (u *User) BeforeUpdate(scope *gorm.Scope) error {
	if len(u.ApiKey) > 64 {
		return nil
	}

	encKey, err := bakery.Encrypt(u.ApiKey)

	if err != nil {
		return err
	}

	return scope.SetColumn("ApiKey", encKey)
}

func (u *User) AfterUpdate(tx *gorm.DB) error {
	if u.Domain == "DELETE" {
		tx.Model(u).UpdateColumn("Domain", "")
	}

	return nil
}

func EncryptApiKeys() {
	log.Info("Encrypting API keys...")
	users := []User{}
	err := db.Where("LENGTH(api_key) <= 64").Find(&users).Error

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for _, u := range users {
		if encKey, err := bakery.Encrypt(u.ApiKey); err == nil {
			if db.Model(u).UpdateColumn("api_key", encKey).Error == nil {
				log.Infof("Encrypted API Key (%s) of user %d (%s)", u.ApiKey, u.Id, u.Username)
			}
		} else {
			log.Error(err)
		}
	}

	log.Info("Done.")
}
