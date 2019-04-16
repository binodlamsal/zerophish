package models

import (
	"errors"
	"strings"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/PuerkitoBio/goquery"
	log "github.com/everycloud-technologies/phishing-simulation/logger"
)

// Page contains the fields used for a Page model
type Page struct {
	Id                 int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId             int64     `json:"user_id" gorm:"column:user_id"`
	Username           string    `json:"username" gorm:"-"`
	Name               string    `json:"name"`
	TagsId             int64     `json:"tag" gorm:"column:tag"`
	Tags               Tags      `json:"tags"`
	HTML               string    `json:"html" gorm:"column:html"`
	CaptureCredentials bool      `json:"capture_credentials" gorm:"column:capture_credentials"`
	CapturePasswords   bool      `json:"capture_passwords" gorm:"column:capture_passwords"`
	Public             bool      `json:"public" gorm:"column:public"`
	RedirectURL        string    `json:"redirect_url" gorm:"column:redirect_url"`
	ModifiedDate       time.Time `json:"modified_date"`
	Writable           bool      `json:"writable" gorm:"-"`
}

// ErrPageNameNotSpecified is thrown if the name of the landing page is blank.
var ErrPageNameNotSpecified = errors.New("Page Name not specified")

// ErrPageCategoryNotSpecified is thrown if the category of the landing page is blank.
var ErrPageCategoryNotSpecified = errors.New("Page category not specified")

// parseHTML parses the page HTML on save to handle the
// capturing (or lack thereof!) of credentials and passwords
func (p *Page) parseHTML() error {
	d, err := goquery.NewDocumentFromReader(strings.NewReader(p.HTML))
	if err != nil {
		return err
	}
	forms := d.Find("form")
	forms.Each(func(i int, f *goquery.Selection) {
		// We always want the submitted events to be
		// sent to our server
		f.SetAttr("action", "")
		if p.CaptureCredentials {
			// If we don't want to capture passwords,
			// find all the password fields and remove the "name" attribute.
			if !p.CapturePasswords {
				inputs := f.Find("input")
				inputs.Each(func(j int, input *goquery.Selection) {
					if t, _ := input.Attr("type"); strings.EqualFold(t, "password") {
						input.RemoveAttr("name")
					}
				})
			}
		} else {
			// Otherwise, remove the name from all
			// inputs.
			inputFields := f.Find("input")
			inputFields.Each(func(j int, input *goquery.Selection) {
				input.RemoveAttr("name")
			})
		}
	})
	p.HTML, err = d.Html()
	return err
}

// Validate ensures that a page contains the appropriate details
func (p *Page) Validate() error {
	if p.Name == "" {
		return ErrPageNameNotSpecified
	}

	if p.TagsId == 0 {
		return ErrPageCategoryNotSpecified
	}

	// If the user specifies to capture passwords,
	// we automatically capture credentials
	if p.CapturePasswords && !p.CaptureCredentials {
		p.CaptureCredentials = true
	}
	if err := ValidateTemplate(p.HTML); err != nil {
		return err
	}
	if err := ValidateTemplate(p.RedirectURL); err != nil {
		return err
	}
	return p.parseHTML()
}

// IsWritableByUser tells if this page can be modified by a user with the given uid
func (p *Page) IsWritableByUser(uid int64) bool {
	role, err := GetUserRole(uid)

	if err != nil {
		log.Error(err)
		return false
	}

	if p.UserId == 0 {
		oid, err := GetPageOwnerId(p.Id)

		if err != nil {
			log.Error(err)
			return false
		}

		p.UserId = oid
	}

	if p.UserId == uid || role.Is(Administrator) {
		return true
	}

	if p.Public {
		return false
	}

	uids, err := GetUserIds(uid)

	if err != nil {
		log.Error(err)
		return false
	}

	if role.Is(ChildUser) {
		u, err := GetUser(uid)

		if err != nil {
			log.Error(err)
			return false
		}

		uids = append(uids, u.Partner)
	}

	for _, u := range uids {
		if u == p.UserId {
			return true
		}
	}

	return false
}

// IsPageWritableByUser tell if a page (identified by pid)
// can be modified by a user (identified by uid)
func IsPageWritableByUser(pid, uid int64) bool {
	role, err := GetUserRole(uid)

	if err != nil {
		log.Error(err)
		return false
	}

	if role.Is(Administrator) {
		return true
	}

	oid, err := GetPageOwnerId(pid)

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

	for _, u := range uids {
		if u == oid {
			return true
		}
	}

	return false
}

// IsPageAccessibleByUser tells if a page (identified by pid)
// is accessible by a user (identified by uid)
func IsPageAccessibleByUser(pid, uid int64) bool {
	oid, err := GetPageOwnerId(pid)

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

// GetPageOwnerId returns user id of creator of the page identified by id
func GetPageOwnerId(id int64) (int64, error) {
	p := Page{}
	err := db.Where("id = ?", id).Select("user_id").First(&p).Error

	if err != nil {
		return 0, errors.New("page not found: " + err.Error())
	}

	return p.UserId, nil
}

// GetPages returns the pages owned by the given user.
// Optionally "filter" can be one of: own, public ,customers.
// Where:
// own - return items that belong to this user,
// public - public items
// customers - customers' items
// Note: empty "filter" will be treated as "own"
func GetPages(uid int64, filter string) ([]Page, error) {
	ps := []Page{}
	role, err := GetUserRole(uid)

	if err != nil {
		return ps, err
	}

	user, err := GetUser(uid)

	if err != nil {
		return ps, err
	}

	if filter != "own" && filter != "customers" && filter != "public" && filter != "own-and-public" {
		if !role.Is(Administrator) {
			filter = "own"
		} else if filter != "all" {
			filter = "own"
		}
	}

	query := db.Table("pages")

	if filter == "own" || filter == "own-and-public" {
		if role.IsOneOf([]int64{Partner, ChildUser}) {
			u, err := GetUser(uid)

			if err != nil {
				return ps, err
			}

			partner := u.Partner

			if role.Is(Partner) {
				partner = u.Id
			}

			cuids, err := GetChildUserIds(partner)

			if err != nil {
				return ps, err
			}

			if filter == "own" {
				query = query.Where("user_id = ? OR user_id = ? OR user_id IN (?)", uid, user.Partner, cuids)
			} else { // own-and-public
				query = query.Where("user_id = ? OR user_id = ? OR user_id IN (?) OR public = ?", uid, user.Partner, cuids, 1)
			}
		} else { // admins and customers
			if filter == "own" {
				query = query.Where("user_id = ?", uid)
			} else { // own-and-public
				query = query.Where("user_id = ? OR public = ?", uid, 1)
			}
		}
	} else if filter == "public" {
		query = query.Where("public = ?", 1)
	} else if filter == "customers" {
		cuids, err := GetCustomerIds(uid)

		if err != nil {
			return ps, err
		}

		query = query.Where("user_id IN (?)", cuids)
	} else { // all
		if !role.Is(Administrator) {
			return ps, err
		}
	}

	err = query.
		Select("pages.*, pages.id AS id, users.username AS username").
		Joins("LEFT JOIN users ON users.id = pages.user_id").
		Scan(&ps).Error

	if err != nil {
		log.Error(err)
		return ps, err
	}

	for i := range ps {
		// Set Writable flag
		ps[i].Writable = ps[i].IsWritableByUser(uid)
	}

	return ps, err
}

// GetPage returns the page, if it exists, specified by the given id
func GetPage(id int64) (Page, error) {
	p := Page{}
	err := db.Where("id=?", id).Find(&p).Error

	if err != nil {
		log.Error(err)
	}

	return p, err
}

// GetPageByName returns the page, if it exists, specified by the given name and user_id.
func GetPageByName(n string, uid int64) (Page, error) {
	p := Page{}
	role, err := GetUserRole(uid)

	if err != nil {
		return p, err
	}

	if role.Is(ChildUser) {
		u, err := GetUser(uid)

		if err != nil {
			return p, err
		}

		if db.
			Where("user_id=? and name=?", uid, n).
			Or("user_id=? and name=?", u.Partner, n).
			Or("public = ? and name=?", 1, n).
			First(&p).RecordNotFound() {
			return p, gorm.ErrRecordNotFound
		}
	} else {
		if db.
			Where("user_id=? and name=?", uid, n).
			Or("public = ? and name=?", 1, n).
			First(&p).RecordNotFound() {
			return p, gorm.ErrRecordNotFound
		}
	}

	return p, err
}

// PostPage creates a new page in the database.
func PostPage(p *Page) error {
	err := p.Validate()
	if err != nil {
		log.Error(err)
		return err
	}

	tg, err := GetTagById(p.TagsId)

	if err != nil {
		log.Error(err)
		return err
	}

	p.Tags = tg

	// Insert into the DB
	err = db.Save(p).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// PutPage edits an existing Page in the database.
// Per the PUT Method RFC, it presumes all data for a page is provided.
func PutPage(p *Page) error {
	err := p.Validate()

	if err != nil {
		log.Error(err)
		return err
	}

	err = db.Model(&p).Updates(p).Error

	if err != nil {
		log.Error(err)
	}

	return err
}

// DeletePage deletes an existing page in the database.
// An error is returned if a page with the given page id is not found.
func DeletePage(id int64) error {
	err = db.Delete(Page{Id: id}).Error

	if err != nil {
		log.Error(err)
	}

	return err
}
