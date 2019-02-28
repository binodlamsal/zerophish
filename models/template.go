package models

import (
	"errors"
	"time"

	log "github.com/binodlamsal/gophish/logger"
	"github.com/jinzhu/gorm"
)

// Template models hold the attributes for an email template to be sent to targets
type Template struct {
	Id           int64        `json:"id" gorm:"column:id; primary_key:yes"`
	UserId       int64        `json:"user_id" gorm:"column:user_id"`
	Username     string       `json:"username" gorm:"-"`
	Name         string       `json:"name"`
	Subject      string       `json:"subject"`
	Text         string       `json:"text"`
	HTML         string       `json:"html" gorm:"column:html"`
	RATING       int64        `json:"rating" gorm:"column:rating"`
	TagsId       int64        `json:"tag" gorm:"column:tag"`
	Tags         Tags         `json:"tags"`
	Public       bool         `json:"public" gorm:"column:public"`
	ModifiedDate time.Time    `json:"modified_date"`
	Attachments  []Attachment `json:"attachments"`
	Writable     bool         `json:"writable" gorm:"-"`
}

// Tags models hold the attributes for the categories of templates and landing pages
type Tags struct {
	Id     int64  `json:"id" gorm:"column:id; primary_key:yes"`
	Name   string `json:"name"`
	Weight int64  `json:"weight"`
}

// ErrTemplateNameNotSpecified is thrown when a template name is not specified
var ErrTemplateNameNotSpecified = errors.New("Template name not specified")

// ErrTemplateMissingParameter is thrown when a needed parameter is not provided
var ErrTemplateMissingParameter = errors.New("Need to specify at least plaintext or HTML content")

// Validate checks the given template to make sure values are appropriate and complete
func (t *Template) Validate() error {
	switch {
	case t.Name == "":
		return ErrTemplateNameNotSpecified
	case t.Text == "" && t.HTML == "":
		return ErrTemplateMissingParameter
	}
	if err = ValidateTemplate(t.HTML); err != nil {
		return err
	}
	if err = ValidateTemplate(t.Text); err != nil {
		return err
	}
	return nil
}

// IsWritableByUser tells if this template can be modified by a user with the given uid
func (t *Template) IsWritableByUser(uid int64) bool {
	role, err := GetUserRole(uid)

	if err != nil {
		log.Error(err)
		return false
	}

	if t.UserId == 0 {
		oid, err := GetTemplateOwnerId(t.Id)

		if err != nil {
			log.Error(err)
			return false
		}

		t.UserId = oid
	}

	if t.UserId == uid || role.Is(Administrator) {
		return true
	}

	if t.Public {
		return false
	}

	uids, err := GetUserIds(uid)

	if err != nil {
		log.Error(err)
		return false
	}

	for _, u := range uids {
		if u == t.UserId {
			return true
		}
	}

	return false
}

//Get tags by tag name
func GetTagById(id int64) (Tags, error) {
	t := Tags{}
	err := db.Where("id=?", id).Find(&t).Error
	if err != nil {
		log.Error(err)
	}
	return t, err
}

// GetTemplateOwnerId returns user id of creator of the template identified by id
func GetTemplateOwnerId(id int64) (int64, error) {
	t := Template{}
	err := db.Where("id = ?", id).Select("user_id").First(&t).Error

	if err != nil {
		return 0, errors.New("template not found: " + err.Error())
	}

	return t.UserId, nil
}

// IsTemplateAccessibleByUser tells if a template (identified by tid)
// is accessible by a user (identified by uid)
func IsTemplateAccessibleByUser(tid, uid int64) bool {
	oid, err := GetTemplateOwnerId(tid)

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

	for _, id := range uids {
		if oid == id {
			return true
		}
	}

	return false
}

// IsTemplateWritableByUser tell if a template (identified by tid)
// can be modified by a user (identified by uid)
func IsTemplateWritableByUser(tid, uid int64) bool {
	role, err := GetUserRole(uid)

	if err != nil {
		log.Error(err)
		return false
	}

	if role.Is(Administrator) {
		return true
	}

	oid, err := GetTemplateOwnerId(tid)

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

	for _, u := range uids {
		if u == oid {
			return true
		}
	}

	return false
}

// GetTemplates returns the templates owned by the given user.
func GetTemplates(uid int64) ([]Template, error) {
	ts := []Template{}
	role, err := GetUserRole(uid)

	if err != nil {
		return ts, err
	}

	query := db.Table("templates")

	if role.Is(Administrator) {
		// just grab all templates (see above)
	} else if role.IsOneOf([]int64{Partner, ChildUser}) {
		uids, err := GetUserIds(uid)

		if err != nil {
			return ts, err
		}

		uids = append(uids, uid)
		query = db.Table("templates").Where("user_id IN (?) OR public=?", uids, 1)
	} else {
		query = db.Table("templates").Where("user_id=? OR public=?", uid, 1)
	}

	err = query.
		Select("templates.*, templates.id AS id, users.username AS username").
		Joins("LEFT JOIN users ON users.id = templates.user_id").
		Scan(&ts).Error

	if err != nil {
		log.Error(err)
		return ts, err
	}

	for i := range ts {
		// Get Attachments
		err = db.Where("template_id=?", ts[i].Id).Find(&ts[i].Attachments).Error

		if err == nil && len(ts[i].Attachments) == 0 {
			ts[i].Attachments = make([]Attachment, 0)
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			log.Error(err)
			return ts, err
		}

		// Set Writable flag
		ts[i].Writable = ts[i].IsWritableByUser(uid)
	}

	return ts, err
}

// GetTags returns the all the tags from the database
func GetTags(uid int64) ([]Tags, error) {
	tg := []Tags{}
	err := db.Order("id asc").Find(&tg).Error
	return tg, err
}

// PostTemplate creates a new template in the database.
func PostTags(t *Tags) error {
	// Insert into the DB
	if t.Name == "" {
		return errors.New("Tag name is not specified")
	}
	if t.Weight == 0 {
		return errors.New("Weight is not specified")
	}

	err = db.Save(t).Error
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// GetTemplate returns the template, if it exists, specified by the given id
func GetTemplate(id int64) (Template, error) {
	t := Template{}
	err := db.Where("id=?", id).Find(&t).Error
	if err != nil {
		log.Error(err)
		return t, err
	}

	// Get Attachments
	err = db.Where("template_id=?", t.Id).Find(&t.Attachments).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err)
		return t, err
	}
	if err == nil && len(t.Attachments) == 0 {
		t.Attachments = make([]Attachment, 0)
	}
	return t, err
}

// GetTemplateByName returns the template, if it exists, specified by the given name and user_id.
func GetTemplateByName(n string, uid int64) (Template, error) {
	t := Template{}
	err := db.Where("user_id=? and name=?", uid, n).Or("public = ? and name=?", 1, n).Find(&t).Error
	if err != nil {
		log.Error(err)
		return t, err
	}

	// Get Attachments
	err = db.Where("template_id=?", t.Id).Find(&t.Attachments).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err)
		return t, err
	}
	if err == nil && len(t.Attachments) == 0 {
		t.Attachments = make([]Attachment, 0)
	}
	return t, err
}

// PostTemplate creates a new template in the database.
func PostTemplate(t *Template) error {
	// Insert into the DB
	if err := t.Validate(); err != nil {
		return err
	}

	tg, err := GetTagById(t.TagsId)

	if err != nil {
		log.Error(err)
		return err
	}

	t.Tags = tg

	err = db.Save(t).Error
	if err != nil {
		log.Error(err)
		return err
	}

	// Save every attachment
	for i := range t.Attachments {
		t.Attachments[i].TemplateId = t.Id
		err := db.Save(&t.Attachments[i]).Error
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

// PutTemplate edits an existing template in the database.
// Per the PUT Method RFC, it presumes all data for a template is provided.
func PutTemplate(t *Template) error {
	if err := t.Validate(); err != nil {
		return err
	}
	// Delete all attachments, and replace with new ones
	err = db.Where("template_id=?", t.Id).Delete(&Attachment{}).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err)
		return err
	}
	if err == gorm.ErrRecordNotFound {
		err = nil
	}
	for i := range t.Attachments {
		t.Attachments[i].TemplateId = t.Id
		err := db.Save(&t.Attachments[i]).Error
		if err != nil {
			log.Error(err)
			return err
		}
	}

	// Save final template
	err = db.Model(&t).Updates(t).Error

	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// PutTags edits an existing tag in the database.
// Per the PUT Method RFC, it presumes all data for tag is provided.
func PutTags(t *Tags) error {
	// Save final template
	err = db.Where("id=?", t.Id).Save(t).Error
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// DeleteTemplate deletes an existing template in the database.
// An error is returned if a template with the given template id is not found.
func DeleteTemplate(id int64) error {
	// Delete attachments
	err := db.Where("template_id=?", id).Delete(&Attachment{}).Error

	if err != nil {
		log.Error(err)
		return err
	}

	// Finally, delete the template itself
	err = db.Delete(Template{Id: id}).Error

	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// DeleteTags deletes an existing tag in the database.
// An error is returned if a template with the given user id and tag id is not found.
func DeleteTags(id int64) error {
	// Finally, delete the template itself
	err := db.Delete(Tags{Id: id}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
