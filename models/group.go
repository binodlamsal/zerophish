package models

import (
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/everycloud-technologies/phishing-simulation/encryption"
	"github.com/everycloud-technologies/phishing-simulation/usersync"

	log "github.com/everycloud-technologies/phishing-simulation/logger"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Group contains the fields needed for a user -> group mapping
// Groups contain 1..* Targets
type Group struct {
	Id           int64     `json:"id"`
	UserId       int64     `json:"-"`
	Name         string    `json:"name"`
	ModifiedDate time.Time `json:"modified_date"`
	Targets      []Target  `json:"targets" sql:"-"`
	CreatorId    int64     `json:"creator,omitemtpy" gorm:"-"`
}

// GroupSummaries is a struct representing the overview of Groups.
type GroupSummaries struct {
	Total  int64          `json:"total"`
	Groups []GroupSummary `json:"groups"`
}

// GroupSummary represents a summary of the Group model. The only
// difference is that, instead of listing the Targets (which could be expensive
// for large groups), it lists the target count.
type GroupSummary struct {
	Id           int64     `json:"id"`
	UserId       int64     `json:"user_id"`
	Username     string    `json:"username"`
	Name         string    `json:"name"`
	ModifiedDate time.Time `json:"modified_date"`
	NumTargets   int64     `json:"num_targets"`
	Locked       bool      `json:"locked"`
}

// GroupTarget is used for a many-to-many relationship between 1..* Groups and 1..* Targets
type GroupTarget struct {
	GroupId  int64 `json:"-"`
	TargetId int64 `json:"-"`
}

// Target contains the fields needed for individual targets specified by the user
// Groups contain 1..* Targets, but 1 Target may belong to 1..* Groups
type Target struct {
	Id int64 `json:"id"`
	BaseRecipient
}

// BaseRecipient contains the fields for a single recipient. This is the base
// struct used in members of groups and campaign results.
type BaseRecipient struct {
	Email     encryption.EncryptedString `json:"email"`
	FirstName string                     `json:"first_name"`
	LastName  string                     `json:"last_name"`
	Position  string                     `json:"position"`
	IsLMSUser bool                       `json:"is_lms_user" gorm:"-"`
}

// FormatAddress returns the email address to use in the "To" header of the email
func (r *BaseRecipient) FormatAddress() string {
	addr := r.Email.String()
	if r.FirstName != "" && r.LastName != "" {
		a := &mail.Address{
			Name:    fmt.Sprintf("%s %s", r.FirstName, r.LastName),
			Address: r.Email.String(),
		}
		addr = a.String()
	}
	return addr
}

// FormatAddress returns the email address to use in the "To" header of the email
func (t *Target) FormatAddress() string {
	addr := t.Email.String()
	if t.FirstName != "" && t.LastName != "" {
		a := &mail.Address{
			Name:    fmt.Sprintf("%s %s", t.FirstName, t.LastName),
			Address: t.Email.String(),
		}
		addr = a.String()
	}
	return addr
}

// GetGroupIds returns group ids which this target belongs to
func (t *Target) GetGroupIds() ([]int64, error) {
	gids := []int64{}
	err = db.Table("group_targets").Where("target_id = ?", t.Id).Pluck("group_id", &gids).Error

	if err != nil {
		return gids, err
	}

	return gids, nil
}

// ErrEmailNotSpecified is thrown when no email is specified for the Target
var ErrEmailNotSpecified = errors.New("No email address specified")

// ErrGroupNameNotSpecified is thrown when a group name is not specified
var ErrGroupNameNotSpecified = errors.New("Group name not specified")

// ErrNoTargetsSpecified is thrown when no targets are specified by the user
var ErrNoTargetsSpecified = errors.New("No targets specified")

// Validate performs validation on a group given by the user
func (g *Group) Validate() error {
	switch {
	case g.Name == "":
		return ErrGroupNameNotSpecified
	case len(g.Targets) == 0:
		return ErrNoTargetsSpecified
	}
	return nil
}

// HasTargets tells if this group contains all given target ids.
// If at least one target id doesn't belong to this group then false will be returned.
func (g *Group) HasTargets(tids []int64) bool {
	var count int64

	err := db.
		Table("group_targets").Select("target_id").
		Where("target_id IN (?) AND group_id = ?", tids, g.Id).
		Count(&count).Error

	if err != nil {
		log.Errorf("Could not get targets of group %d - ", g.Id, err.Error())
		return false
	}

	return len(tids) == int(count)
}

// ContainsTargetsOutsideOfDomain tells if this group's targets contain one or more email addresses
// outside of the given domain. If domain is blank then it always returns false.
func (g *Group) ContainsTargetsOutsideOfDomain(domain string) bool {
	if domain == "" {
		return false
	}

	for _, t := range g.Targets {
		if !strings.HasSuffix(t.Email.String(), domain) {
			return true
		}
	}

	return false
}

// AddTargets adds the given targets to this group
func (g *Group) AddTargets(targets []Target) (err error) {
	for _, t := range targets {
		if g.HasTargets([]int64{t.Id}) {
			continue
		}

		err = insertTargetIntoGroup(t, g.Id)

		if err != nil {
			err = fmt.Errorf("Could not add target %d to group %d: %s", t.Id, g.Id, err.Error())
			return
		}

		log.Infof("Added target user %s to group %s", t.Email, g.Name)
	}

	return
}

// RemoveTargets removes the given targets from this group.
// After removal, targets that won't belong to at least one another group will be deleted.
func (g *Group) RemoveTargets(targets []Target) (err error) {
	for _, t := range targets {
		if !g.HasTargets([]int64{t.Id}) {
			continue
		}

		err = db.Where("group_id=? AND target_id=?", g.Id, t.Id).Delete(&GroupTarget{}).Error

		if err != nil {
			err = fmt.Errorf("Could not remove target %d from group %d: %s", t.Id, g.Id, err.Error())
			return
		}

		var count int64
		err = db.Table("group_targets").Where("target_id=?", t.Id).Count(&count).Error

		if err != nil {
			err = fmt.Errorf("Could not count number of groups target %d belongs to: %s", t.Id, err.Error())
			return
		}

		if count == 0 {
			err = DeleteGroupTarget(t.Id)

			if err != nil {
				return
			}

			log.Infof("Removed target user %s from group %s", t.Email, g.Name)
		}
	}

	return
}

// GetGroups returns the groups owned by the given user.
func GetGroups(uid int64) ([]Group, error) {
	gs := []Group{}
	user, err := GetUser(uid)

	if err != nil {
		return gs, err
	}

	if user.IsAdministrator() {
		err = db.Find(&gs).Error
	} else if user.IsChildUser() || user.IsPartner() {
		partner := user.Partner

		if user.IsPartner() {
			partner = user.Id
		}

		cuids, err := GetChildUserIds(partner)

		if err != nil {
			return gs, err
		}

		err = db.Where("user_id=? OR user_id=? OR user_id IN (?)", uid, user.Partner, cuids).Find(&gs).Error
	} else { // customers
		err = db.Where("user_id=?", uid).Find(&gs).Error
	}

	if err != nil {
		log.Error(err)
		return gs, err
	}
	for i := range gs {
		gs[i].Targets, err = GetTargets(gs[i].Id)
		if err != nil {
			log.Error(err)
		}
	}
	return gs, nil
}

// GetGroupSummaries returns the summaries for the groups
// created and/or accessible (in case of admin and partner) by the given uid.
// Optionally "filter" can be one of: own, customers.
// Where:
// own - return items that belong to this user,
// customers - customers' items
// uid-xxx - items of user with uid xxx
// Note: empty "filter" will be treated as "own"
func GetGroupSummaries(uid int64, filter string) (GroupSummaries, error) {
	if filter != "own" && filter != "customers" && !strings.HasPrefix(filter, "uid-") {
		filter = "own"
	}

	gs := GroupSummaries{}
	role, err := GetUserRole(uid)

	if err != nil {
		return gs, err
	}

	user, err := GetUser(uid)

	if err != nil {
		return gs, err
	}

	subscribed := user.IsSubscribed() || user.IsAdministrator()

	var query *gorm.DB

	if filter == "own" {
		if role.IsOneOf([]int64{Partner, ChildUser}) {
			partner := user.Partner

			if role.Is(Partner) {
				partner = user.Id
			}

			cuids, err := GetChildUserIds(partner)

			if err != nil {
				return gs, err
			}

			query = db.Table("groups").Where("user_id = ? OR user_id = ? OR user_id IN (?)", uid, user.Partner, cuids)
		} else {
			query = db.Table("groups").Where("user_id = ?", uid)
		}
	} else if strings.HasPrefix(filter, "uid-") {
		uid, err := strconv.Atoi(strings.TrimPrefix(filter, "uid-"))

		if err != nil {
			return gs, fmt.Errorf("Couldn't parse uid from filter %s: %s", filter, err.Error())
		}

		if !user.CanManageUserWithId(int64(uid)) {
			return gs, fmt.Errorf("Not enough permissions to access resources of user with id %d", uid)
		}

		query = db.Table("groups").Where("user_id = ?", uid)
	} else { // customers
		cuids, err := GetCustomerIds(uid)

		if err != nil {
			return gs, err
		}

		query = db.Table("groups").Where("user_id IN (?)", cuids)
	}

	err = query.
		Select("groups.id AS id, user_id, users.username AS username, name, modified_date").
		Joins("LEFT JOIN users ON users.id = groups.user_id").
		Scan(&gs.Groups).Error

	if err != nil {
		log.Error(err)
		return gs, err
	}
	for i := range gs.Groups {
		if filter == "own" { // if no valid subscription then "lock" all except the first group
			if !subscribed && i > 0 {
				gs.Groups[i].Locked = true
			}
		} else { // for partners viewing customers' groups if no valid subscription then "lock" all groups
			if !subscribed {
				gs.Groups[i].Locked = true
			}
		}

		query = db.Table("group_targets").Where("group_id=?", gs.Groups[i].Id)
		err = query.Count(&gs.Groups[i].NumTargets).Error
		if err != nil {
			return gs, err
		}
	}
	gs.Total = int64(len(gs.Groups))
	return gs, nil
}

// GetGroup returns the group, if it exists, specified by the given id
func GetGroup(id int64) (Group, error) {
	g := Group{}
	err := db.Where("id=?", id).Find(&g).Error

	if err != nil {
		log.Error(err)
		return g, err
	}

	g.Targets, err = GetTargets(g.Id)

	if err != nil {
		log.Error(err)
	}

	return g, nil
}

// GetGroupSummary returns the summary for the requested group
func GetGroupSummary(id int64, uid int64) (GroupSummary, error) {
	g := GroupSummary{}
	query := db.Table("groups").Where("user_id=? and id=?", uid, id)
	err := query.Select("id, name, modified_date").Scan(&g).Error
	if err != nil {
		log.Error(err)
		return g, err
	}
	query = db.Table("group_targets").Where("group_id=?", id)
	err = query.Count(&g.NumTargets).Error
	if err != nil {
		return g, err
	}
	return g, nil
}

// GetGroupByName returns the group, if it exists, specified by the given name and user_id.
func GetGroupByName(n string, uid int64) (Group, error) {
	g := Group{}
	role, err := GetUserRole(uid)

	if err != nil {
		return g, err
	}

	if role.Is(Administrator) {
		if db.Where("name=?", n).First(&g).RecordNotFound() {
			return g, gorm.ErrRecordNotFound
		}
	} else if role.IsOneOf([]int64{Partner, ChildUser}) {
		u, err := GetUser(uid)

		if err != nil {
			return g, err
		}

		partner := u.Partner

		if role.Is(Partner) {
			partner = u.Id
		}

		cuids, err := GetChildUserIds(partner)

		if err != nil {
			return g, err
		}

		if db.
			Where("(user_id=? OR user_id=? OR user_id IN (?)) and name=?", uid, u.Partner, cuids, n).
			First(&g).RecordNotFound() {
			return g, gorm.ErrRecordNotFound
		}
	} else { // customer
		if db.Where("user_id=? and name=?", uid, n).First(&g).RecordNotFound() {
			return g, gorm.ErrRecordNotFound
		}
	}

	g.Targets, err = GetTargets(g.Id)

	if err != nil {
		log.Error(err)
	}

	return g, err
}

// PostGroup creates a new group in the database.
func PostGroup(g *Group) error {
	if err := g.Validate(); err != nil {
		return err
	}
	// Insert the group into the DB
	err = db.Save(g).Error
	if err != nil {
		log.Error(err)
		return err
	}
	for _, t := range g.Targets {
		insertTargetIntoGroup(t, g.Id)
	}
	return nil
}

// CreateEmptyGroup creates a new empty group with the given name owned by uid.
func CreateEmptyGroup(name string, uid int64) (Group, error) {
	g := Group{}

	if _, err := GetGroupByName(name, uid); err == nil {
		return g, fmt.Errorf("Group %s already exists", name)
	}

	g = Group{
		UserId:       uid,
		Name:         name,
		ModifiedDate: time.Now(),
	}

	if err := db.Save(&g).Error; err != nil {
		return g, fmt.Errorf("Could not create empty group - %s: %s", name, err.Error())
	}

	return g, nil
}

// PutGroup updates the given group if found in the database.
func PutGroup(g *Group) error {
	if err := g.Validate(); err != nil {
		return err
	}
	// Fetch group's existing targets from database.
	ts := []Target{}
	ts, err = GetTargets(g.Id)
	if err != nil {
		log.WithFields(logrus.Fields{
			"group_id": g.Id,
		}).Error("Error getting targets from group")
		return err
	}
	// Check existing targets, removing any that are no longer in the group.
	tExists := false
	for _, t := range ts {
		tExists = false
		// Is the target still in the group?
		for _, nt := range g.Targets {
			if t.Email.Equals(nt.Email) {
				tExists = true
				break
			}
		}
		// If the target does not exist in the group any longer, we delete it
		if !tExists {
			err = db.Where("group_id=? and target_id=?", g.Id, t.Id).Delete(&GroupTarget{}).Error

			if err != nil {
				log.WithFields(logrus.Fields{
					"email": t.Email,
				}).Error("Error deleting target-to-group relationship")
			}

			if DeleteGroupTarget(t.Id) != nil {
				continue
			}

			// Delete related LMS user (if any)
			if u, err := GetLMSUser(t.Email.String()); err == nil {
				if buid, err := DeleteUser(u.Id); err == nil {
					if os.Getenv("USERSYNC_DISABLE") == "" {
						usersync.DeleteTrainingCampaigns(u.Id)

						if err := usersync.DeleteUser(buid); err != nil {
							log.Error(err)
						}
					}

					log.Infof("Deleted related LMS user %s", u.Email)
				}
			}
		}
	}
	// Add any targets that are not in the database yet.
	for _, nt := range g.Targets {
		// Check and see if the target already exists in the db
		tExists = false
		for _, t := range ts {
			if t.Email.Equals(nt.Email) {
				tExists = true
				nt.Id = t.Id
				break
			}
		}
		// Add target if not in database, otherwise update target information.
		if !tExists {
			insertTargetIntoGroup(nt, g.Id)
		} else {
			UpdateTarget(nt)
		}
	}
	err = db.Model(&g).Updates(g).Error
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// DeleteGroup deletes a given group by group ID and user ID
func DeleteGroup(g *Group) error {
	// Delete all the group_targets entries for this group
	err := db.Where("group_id=?", g.Id).Delete(&GroupTarget{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	// Delete the group itself
	err = db.Delete(g).Error
	if err != nil {
		log.Error(err)
		return err
	}

	for _, t := range g.Targets {
		if DeleteGroupTarget(t.Id) != nil {
			continue
		}

		// Delete related LMS user (if any)
		if u, err := GetLMSUser(t.Email.String()); err == nil {
			if buid, err := DeleteUser(u.Id); err == nil {
				if os.Getenv("USERSYNC_DISABLE") == "" {
					usersync.DeleteTrainingCampaigns(u.Id)

					if err := usersync.DeleteUser(buid); err != nil {
						log.Error(err)
					}
				}

				log.Infof("Deleted related LMS user %s", u.Email)
			}
		}
	}

	return nil
}

// DeleteUserGroups deletes groups created by user with the given uid
func DeleteUserGroups(uid int64) error {
	var groups []*Group
	err := db.Table("groups").Where("user_id=?", uid).Scan(&groups).Error

	if err != nil {
		return fmt.Errorf(
			"Couldn't find groups owned by user with id %d - %s",
			uid, err.Error(),
		)
	}

	var errs []error

	for _, g := range groups {
		g.Targets, _ = GetTargets(g.Id)
		err = DeleteGroup(g)

		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf(
			"Couldn't delete %d group(s) owned by user with id %d",
			len(errs), uid,
		)
	}

	return nil
}

func insertTargetIntoGroup(t Target, gid int64) error {
	if _, err = mail.ParseAddress(t.Email.String()); err != nil {
		log.WithFields(logrus.Fields{
			"email": t.Email,
		}).Error("Invalid email")
		return err
	}
	trans := db.Begin()
	err = trans.Where(t).FirstOrCreate(&t).Error
	if err != nil {
		log.WithFields(logrus.Fields{
			"email": t.Email,
		}).Error(err)
		trans.Rollback()
		return err
	}
	err = trans.Where("group_id=? and target_id=?", gid, t.Id).Find(&GroupTarget{}).Error
	if err == gorm.ErrRecordNotFound {
		err = trans.Save(&GroupTarget{GroupId: gid, TargetId: t.Id}).Error
		if err != nil {
			log.Error(err)
			trans.Rollback()
			return err
		}
	}
	if err != nil {
		log.WithFields(logrus.Fields{
			"email": t.Email,
		}).Error("Error adding many-many mapping")
		trans.Rollback()
		return err
	}
	err = trans.Commit().Error
	if err != nil {
		trans.Rollback()
		log.Error("Error committing db changes")
		return err
	}
	return nil
}

// UpdateTarget updates the given target information in the database.
func UpdateTarget(target Target) error {
	targetInfo := map[string]interface{}{
		"first_name": target.FirstName,
		"last_name":  target.LastName,
		"position":   target.Position,
	}
	err := db.Model(&target).Where("id = ?", target.Id).Updates(targetInfo).Error
	if err != nil {
		log.WithFields(logrus.Fields{
			"email": target.Email,
		}).Error("Error updating target information")
	}
	return err
}

// GetTargets performs a many-to-many select to get all the Targets for a Group
func GetTargets(gid int64) ([]Target, error) {
	ts := []Target{}

	err := db.
		Table("targets").
		Select(
			`targets.id, targets.email, targets.first_name, targets.last_name, targets.position,
			(SELECT COUNT(*) FROM users LEFT JOIN users_role ON users_role.uid = users.id
			WHERE users.email = targets.email AND users_role.rid = 5 LIMIT 1) AS is_lms_user`,
		).
		Joins("left join group_targets gt ON targets.id = gt.target_id").
		Where("gt.group_id=?", gid).
		Scan(&ts).Error

	return ts, err
}

// GetTargetsByIds returns group targets identified by the given ids
func GetTargetsByIds(tids []int64) ([]Target, error) {
	ts := []Target{}

	if err := db.Table("targets").Where("id IN (?)", tids).Scan(&ts).Error; err != nil {
		return ts, err
	}

	return ts, nil
}

// GetGroupOwnerId returns user id of creator of the group identified by id
func GetGroupOwnerId(id int64) (int64, error) {
	g := Group{}
	err := db.Where("id = ?", id).Select("user_id").First(&g).Error

	if err != nil {
		return 0, errors.New("group not found: " + err.Error())
	}

	return g.UserId, nil
}

// GetTargetByEmail finds and returns target with the given email
func GetTargetByEmail(email string) (t Target, err error) {
	err = db.Where("email = ?", encryption.EncryptedString{email}).First(&t).Error
	return
}

// IsGroupAccessibleByUser tells if a group (identified by gid)
// is accessible by a user (identified by uid)
func IsGroupAccessibleByUser(gid, uid int64) bool {
	oid, err := GetGroupOwnerId(gid)

	if err != nil {
		log.Error(err)
		return false
	}

	if oid == uid {
		return true
	}

	uids, err := GetUserIds(uid)

	if err != nil {
		log.Error(err)
		return false
	}

	u, err := GetUser(uid)

	if err != nil {
		log.Error(err)
		return false
	}

	if u.IsChildUser() {
		uids = append(uids, u.Partner)
	}

	for _, id := range uids {
		if oid == id {
			return true
		}
	}

	return false
}

// IsGroupLockedForUser tells if a group with the given gid is locked for user identified by uid
func IsGroupLockedForUser(gid, uid int64) bool {
	u, err := GetUser(uid)

	if err != nil {
		log.Error(err)
		return true
	}

	if u.IsSubscribed() || u.IsAdministrator() {
		return false
	}

	result := struct {
		ID int64
	}{}

	var uids []int64

	if u.IsPartner() || u.IsChildUser() {
		uids, err = GetUserIds(uid)

		if err != nil {
			log.Errorf("Could not find users related to uid %d - %s", uid, err.Error())
			return true
		}
	}

	uids = append(uids, uid)

	if db.Raw("SELECT id FROM groups WHERE user_id IN (?) ORDER BY id ASC LIMIT 1", uids).Scan(&result).Error != nil {
		log.Error(err)
		return true
	}

	return gid > result.ID
}

// GetTargetsFullName finds and returns full name of a target user
// with the given email which is in a user group owned by the given user (oid).
// If no matching target user found then an empty string will be returned.
func GetTargetsFullName(email string, oid int64) string {
	groups, err := GetGroups(oid)

	if err != nil {
		return ""
	}

	for _, g := range groups {
		for _, t := range g.Targets {
			if t.Email.String() == email {
				return t.FirstName + " " + t.LastName
			}
		}
	}

	return ""
}

// DeleteGroupTarget deletes a group target identified by the given id if
// such target belongs to not more than one group, otherwise returns an error.
func DeleteGroupTarget(id int64) error {
	var groups int
	db.Table("group_targets").Where("target_id=?", id).Count(&groups)

	if groups > 0 {
		return errors.New("Won't delete this target because it belongs to another group")
	}

	err = db.Where("id=?", id).Delete(&Target{}).Error

	if err != nil {
		return err
	}

	return nil
}

// EncryptTargetEmails encrypts email column in targets table
func EncryptTargetEmails() {
	log.Info("Encrypting emails in targets table...")

	type target struct {
		ID    int64  `json:"id"`
		Email string `json:"email" sql:"not null;unique"`
	}

	targets := []target{}

	err := db.
		Table("targets").
		Where(`email LIKE "%@%"`).
		Find(&targets).
		Error

	if err != nil {
		log.Error(err)
		return
	}

	for _, t := range targets {
		email, err := encryption.Encrypt(t.Email)

		if err != nil {
			log.Error(err)
			return
		}

		err = db.
			Table("targets").
			Where("id = ?", t.ID).
			UpdateColumns(target{Email: email}).
			Error

		if err != nil {
			log.Error(err)
			return
		}

		log.Infof("Encrypted email of target with id %d", t.ID)
	}

	log.Info("Done.")
}
