package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/everycloud-technologies/phishing-simulation/bakery"
	log "github.com/everycloud-technologies/phishing-simulation/logger"
	"github.com/everycloud-technologies/phishing-simulation/usersync"
	"github.com/everycloud-technologies/phishing-simulation/util"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Campaign is a struct representing a created campaign
type Campaign struct {
	Id                int64     `json:"id"`
	UserId            int64     `json:"-"`
	Name              string    `json:"name" sql:"not null"`
	CreatedDate       time.Time `json:"created_date"`
	LaunchDate        time.Time `json:"launch_date"`
	SendByDate        time.Time `json:"send_by_date"`
	StartTime         string    `json:"start_time"`
	EndTime           string    `json:"end_time"`
	TimeZone          string    `json:"time_zone"`
	CompletedDate     time.Time `json:"completed_date"`
	TemplateId        int64     `json:"-"`
	Template          Template  `json:"template"`
	FromAddress       string    `json:"from_address"`
	PageId            int64     `json:"-"`
	Page              Page      `json:"page"`
	Status            string    `json:"status"`
	Results           []Result  `json:"results,omitempty"`
	Groups            []Group   `json:"groups,omitempty"`
	Events            []Event   `json:"timeline,omitemtpy"`
	SMTPId            int64     `json:"-"`
	SMTP              SMTP      `json:"smtp"`
	URL               string    `json:"url"`
	RemoveNonClickers bool      `json:"remove_non_clickers"`
	ClickersGroupId   int64     `json:"clickers_group_id"`
	ClickersGroup     string    `json:"clickers_group,omitemtpy" gorm:"-"`
}

// CampaignResults is a struct representing the results from a campaign
type CampaignResults struct {
	Id           int64     `json:"id"`
	Name         string    `json:"name"`
	LaunchDate   time.Time `json:"launch_date"`
	TemplateID   int64     `json:"template_id"`
	TemplateName string    `json:"template_name"`
	PageID       int64     `json:"page_id"`
	PageName     string    `json:"page_name"`
	Status       string    `json:"status"`
	Results      []Result  `json:"results,omitempty"`
	Events       []Event   `json:"timeline,omitempty"`
}

// CampaignSummaries is a struct representing the overview of campaigns
type CampaignSummaries struct {
	Total     int64             `json:"total"`
	Campaigns []CampaignSummary `json:"campaigns"`
}

// CampaignSummary is a struct representing the overview of a single camaign
type CampaignSummary struct {
	Id            int64         `json:"id"`
	UserId        int64         `json:"user_id"`
	Username      string        `json:"username"`
	CreatedDate   time.Time     `json:"created_date"`
	LaunchDate    time.Time     `json:"launch_date"`
	SendByDate    time.Time     `json:"send_by_date"`
	CompletedDate time.Time     `json:"completed_date"`
	Status        string        `json:"status"`
	Name          string        `json:"name"`
	Stats         CampaignStats `json:"stats"`
	Locked        bool          `json:"locked"`
}

// CampaignStats is a struct representing the statistics for a single campaign
type CampaignStats struct {
	Total         int64 `json:"total"`
	EmailsSent    int64 `json:"sent"`
	OpenedEmail   int64 `json:"opened"`
	ClickedLink   int64 `json:"clicked"`
	SubmittedData int64 `json:"submitted_data"`
	EmailReported int64 `json:"email_reported"`
	Error         int64 `json:"error"`
}

// Event contains the fields for an event
// that occurs during the campaign
type Event struct {
	Id         int64     `json:"-"`
	CampaignId int64     `json:"-"`
	Email      string    `json:"email"`
	Time       time.Time `json:"time"`
	Message    string    `json:"message"`
	Details    string    `json:"details"`
}

// EventDetails is a struct that wraps common attributes we want to store
// in an event
type EventDetails struct {
	Payload url.Values        `json:"payload"`
	Browser map[string]string `json:"browser"`
}

// EventError is a struct that wraps an error that occurs when sending an
// email to a recipient
type EventError struct {
	Error string `json:"error"`
}

// ErrCampaignNameNotSpecified indicates there was no template given by the user
var ErrCampaignNameNotSpecified = errors.New("Campaign name not specified")

// ErrGroupNotSpecified indicates there was no template given by the user
var ErrGroupNotSpecified = errors.New("No groups specified")

// ErrTemplateNotSpecified indicates there was no template given by the user
var ErrTemplateNotSpecified = errors.New("No email template specified")

// ErrPageNotSpecified indicates a landing page was not provided for the campaign
var ErrPageNotSpecified = errors.New("No landing page specified")

// ErrSMTPNotSpecified indicates a sending profile was not provided for the campaign
var ErrSMTPNotSpecified = errors.New("No sending profile specified")

// ErrTemplateNotFound indicates the template specified does not exist in the database
var ErrTemplateNotFound = errors.New("Template not found")

// ErrGroupNotFound indicates a group specified by the user does not exist in the database
var ErrGroupNotFound = errors.New("Group not found")

// ErrPageNotFound indicates a page specified by the user does not exist in the database
var ErrPageNotFound = errors.New("Page not found")

// ErrSMTPNotFound indicates a sending profile specified by the user does not exist in the database
var ErrSMTPNotFound = errors.New("Sending profile not found")

// ErrInvalidSendByDate indicates that the user specified a send by date that occurs before the
// launch date
var ErrInvalidSendByDate = errors.New("The launch date must be before the \"send emails by\" date")

// ErrCampaignFromAddressNotValid is thrown when the "from" address is not valid
var ErrCampaignFromAddressNotValid = errors.New("The sender's address is not valid")

// RecipientParameter is the URL parameter that points to the result ID for a recipient.
const RecipientParameter = "rid"

// Validate checks to make sure there are no invalid fields in a submitted campaign
func (c *Campaign) Validate() error {
	switch {
	case c.Name == "":
		return ErrCampaignNameNotSpecified
	case len(c.Groups) == 0:
		return ErrGroupNotSpecified
	case c.Template.Name == "":
		return ErrTemplateNotSpecified
	case c.Page.Name == "":
		return ErrPageNotSpecified
	case c.SMTP.Name == "":
		return ErrSMTPNotSpecified
	case !c.SendByDate.IsZero() && !c.LaunchDate.IsZero() && c.SendByDate.Before(c.LaunchDate):
		return ErrInvalidSendByDate
	}

	if c.FromAddress != "" {
		if _, err = mail.ParseAddress(c.FromAddress); err != nil {
			return ErrCampaignFromAddressNotValid
		}
	}

	return nil
}

// UpdateStatus changes the campaign status appropriately
func (c *Campaign) UpdateStatus(s string) error {
	// This could be made simpler, but I think there's a bug in gorm
	return db.Table("campaigns").Where("id=?", c.Id).Update("status", s).Error
}

// AddEvent creates a new campaign event in the database
func (c *Campaign) AddEvent(e *Event) error {
	e.CampaignId = c.Id
	e.Time = time.Now().UTC()
	return db.Save(e).Error
}

// getDetails retrieves the related attributes of the campaign
// from the database. If the Events and the Results are not available,
// an error is returned. Otherwise, the attribute name is set to [Deleted],
// indicating the user deleted the attribute (template, smtp, etc.)
func (c *Campaign) getDetails() error {
	err = db.Model(c).Related(&c.Results).Error
	if err != nil {
		log.Warnf("%s: results not found for campaign", err)
		return err
	}
	err = db.Model(c).Related(&c.Events).Error
	if err != nil {
		log.Warnf("%s: events not found for campaign", err)
		return err
	}
	err = db.Table("templates").Where("id=?", c.TemplateId).Find(&c.Template).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		c.Template = Template{Name: "[Deleted]"}
		log.Warnf("%s: template not found for campaign", err)
	}
	err = db.Where("template_id=?", c.Template.Id).Find(&c.Template.Attachments).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Warn(err)
		return err
	}
	err = db.Table("pages").Where("id=?", c.PageId).Find(&c.Page).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		c.Page = Page{Name: "[Deleted]"}
		log.Warnf("%s: page not found for campaign", err)
	}
	err = db.Table("smtp").Where("id=?", c.SMTPId).Find(&c.SMTP).Error
	if err != nil {
		// Check if the SMTP was deleted
		if err != gorm.ErrRecordNotFound {
			return err
		}
		c.SMTP = SMTP{Name: "[Deleted]"}
		log.Warnf("%s: sending profile not found for campaign", err)
	}
	err = db.Where("smtp_id=?", c.SMTP.Id).Find(&c.SMTP.Headers).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Warn(err)
		return err
	}
	return nil
}

// getBaseURL returns the Campaign's configured URL.
// This is used to implement the TemplateContext interface.
func (c *Campaign) getBaseURL() string {
	return c.URL
}

// getFromAddress returns the Campaign's configured SMTP "From" address.
// This is used to implement the TemplateContext interface.
func (c *Campaign) getFromAddress() string {
	return c.SMTP.FromAddress
}

// generateSendDate creates a sendDate
func (c *Campaign) generateSendDate(idx int, totalRecipients int) time.Time {
	// If no send date is specified, just return the launch date
	if c.SendByDate.IsZero() || c.SendByDate.Equal(c.LaunchDate) {
		return c.LaunchDate
	}
	// Otherwise, we can calculate the range of minutes to send emails
	// (since we only poll once per minute)
	totalMinutes := c.SendByDate.Sub(c.LaunchDate).Minutes()

	// Next, we can determine how many minutes should elapse between emails
	minutesPerEmail := totalMinutes / float64(totalRecipients)

	// Then, we can calculate the offset for this particular email
	offset := int(minutesPerEmail * float64(idx))

	// Finally, we can just add this offset to the launch date to determine
	// when the email should be sent
	return c.LaunchDate.Add(time.Duration(offset) * time.Minute)
}

// GetClickers returns target users who clicked at least one phishing link during this campaign
func (c *Campaign) GetClickers() ([]Target, error) {
	clickers := []Target{}
	results := []Result{}

	err = db.
		Where("campaign_id = ? AND status IN (?)", c.Id, []string{EVENT_CLICKED, EVENT_DATA_SUBMIT}).
		Find(&results).Error

	for _, r := range results {
		t, err := GetTargetByEmail(r.Email)

		if err != nil {
			return clickers, err
		}

		clickers = append(clickers, t)
	}

	return clickers, nil
}

// GetNonClickers returns target users who never clicked a phishing link during this campaign
func (c *Campaign) GetNonClickers() ([]Target, error) {
	nonclickers := []Target{}
	results := []Result{}

	err = db.
		Where("campaign_id = ? AND status NOT IN (?)", c.Id, []string{EVENT_CLICKED, EVENT_DATA_SUBMIT}).
		Find(&results).Error

	for _, r := range results {
		t, err := GetTargetByEmail(r.Email)

		if err != nil {
			return nonclickers, err
		}

		nonclickers = append(nonclickers, t)
	}

	return nonclickers, nil
}

// getCampaignStats returns a CampaignStats object for the campaign with the given campaign ID.
// It also backfills numbers as appropriate with a running total, so that the values are aggregated.
func getCampaignStats(cid int64) (CampaignStats, error) {
	s := CampaignStats{}
	query := db.Table("results").Where("campaign_id = ?", cid)
	err := query.Count(&s.Total).Error
	if err != nil {
		return s, err
	}
	query.Where("status=?", EVENT_DATA_SUBMIT).Count(&s.SubmittedData)
	if err != nil {
		return s, err
	}
	query.Where("status=?", EVENT_CLICKED).Count(&s.ClickedLink)
	if err != nil {
		return s, err
	}
	query.Where("reported=?", true).Count(&s.EmailReported)
	if err != nil {
		return s, err
	}
	// Every submitted data event implies they clicked the link
	s.ClickedLink += s.SubmittedData
	err = query.Where("status=?", EVENT_OPENED).Count(&s.OpenedEmail).Error
	if err != nil {
		return s, err
	}
	// Every clicked link event implies they opened the email
	s.OpenedEmail += s.ClickedLink
	err = query.Where("status=?", EVENT_SENT).Count(&s.EmailsSent).Error
	if err != nil {
		return s, err
	}
	// Every opened email event implies the email was sent
	s.EmailsSent += s.OpenedEmail
	err = query.Where("status=?", ERROR).Count(&s.Error).Error
	return s, err
}

// GetCampaigns returns the campaigns owned by the given user.
func GetCampaigns(uid int64) ([]Campaign, error) {
	cs := []Campaign{}
	err := db.Model(&User{Id: uid}).Related(&cs).Error
	if err != nil {
		log.Error(err)
	}
	for i := range cs {
		err = cs[i].getDetails()
		if err != nil {
			log.Error(err)
		}
	}
	return cs, err
}

// GetCampaignOwnerId returns user id of creator of the campaign identified by id
func GetCampaignOwnerId(id int64) (int64, error) {
	c := Campaign{}
	err := db.Where("id = ?", id).Select("user_id").First(&c).Error

	if err != nil {
		return 0, errors.New("campaign not found: " + err.Error())
	}

	return c.UserId, nil
}

// IsCampaignAccessibleByUser tells if a campaign (identified by cid)
// is accessible by a user (identified by uid)
func IsCampaignAccessibleByUser(cid, uid int64) bool {
	oid, err := GetCampaignOwnerId(cid)

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

// IsCampaignLockedForUser tells if a campaign with the given cid is locked for user identified by uid
func IsCampaignLockedForUser(cid, uid int64) bool {
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

	if db.Raw("SELECT id FROM campaigns WHERE user_id IN (?) ORDER BY id ASC LIMIT 1", uids).Scan(&result).Error != nil {
		log.Error(err)
		return true
	}

	return cid > result.ID
}

// GetCampaignSummaries gets the summary objects for all the campaigns
// owned by the current user and optionally (depending on the user role)
// campaigns of all other users (for admins) or the respective child users (for partners).
// Optionally "filter" can be one of: own, customers.
// Where:
// own - return items that belong to this user,
// customers - customers' items
// Note: empty "filter" will be treated as "own"
func GetCampaignSummaries(uid int64, filter string) (CampaignSummaries, error) {
	if filter != "own" && filter != "customers" {
		filter = "own"
	}

	overview := CampaignSummaries{}
	cs := []CampaignSummary{}
	role, err := GetUserRole(uid)

	if err != nil {
		return overview, err
	}

	user, err := GetUser(uid)

	if err != nil {
		return overview, err
	}

	subscribed := user.IsSubscribed() || user.IsAdministrator()

	// Get the basic campaign information
	var query *gorm.DB

	if filter == "own" {
		if role.IsOneOf([]int64{Partner, ChildUser}) {
			partner := user.Partner

			if role.Is(Partner) {
				partner = user.Id
			}

			cuids, err := GetChildUserIds(partner)

			if err != nil {
				return overview, err
			}

			query = db.Table("campaigns").Where("user_id = ? OR user_id = ? OR user_id IN (?)", uid, user.Partner, cuids)
		} else {
			query = db.Table("campaigns").Where("user_id = ?", uid)
		}
	} else { // customers
		cuids, err := GetCustomerIds(uid)

		if err != nil {
			return overview, err
		}

		query = db.Table("campaigns").Where("user_id IN (?)", cuids)
	}

	err = query.
		Select("campaigns.id AS id, user_id, users.username AS username, name, created_date, launch_date, completed_date, status").
		Joins("LEFT JOIN users ON users.id = campaigns.user_id").
		Scan(&cs).Error

	if err != nil {
		return overview, err
	}

	for i := range cs {
		s, err := getCampaignStats(cs[i].Id)

		if err != nil {
			return overview, err
		}

		cs[i].Stats = s

		if filter == "own" { // if no valid subscription then "lock" all except the first campaign
			if !subscribed && i > 0 {
				cs[i].Locked = true
			}
		} else { // for partners viewing customers' campaigns if no valid subscription then "lock" all campaigns
			if !subscribed {
				cs[i].Locked = true
			}
		}
	}

	overview.Total = int64(len(cs))
	overview.Campaigns = cs
	return overview, nil
}

// GetCampaignSummary gets the summary object for a campaign specified by the campaign ID
func GetCampaignSummary(id int64, uid int64) (CampaignSummary, error) {
	cs := CampaignSummary{}
	query := db.Table("campaigns").Where("user_id = ? AND id = ?", uid, id)
	query = query.Select("id, name, created_date, launch_date, completed_date, status")
	err := query.Scan(&cs).Error
	if err != nil {
		log.Error(err)
		return cs, err
	}
	s, err := getCampaignStats(cs.Id)
	if err != nil {
		log.Error(err)
		return cs, err
	}
	cs.Stats = s
	return cs, nil
}

// GetCampaign returns the campaign, if it exists, specified by the given id
func GetCampaign(id int64) (Campaign, error) {
	c := Campaign{}
	err := db.Where("id = ?", id).Find(&c).Error
	if err != nil {
		log.Errorf("%s: campaign not found", err)
		return c, err
	}
	err = c.getDetails()
	return c, err
}

// GetCampaignResults returns just the campaign results for the given campaign
func GetCampaignResults(id int64) (CampaignResults, error) {
	cr := CampaignResults{}
	err := db.Table("campaigns").Where("id=?", id).Find(&cr).Error

	if err != nil {
		log.WithFields(logrus.Fields{
			"campaign_id": id,
			"error":       err,
		}).Error(err)
		return cr, err
	}

	cr.TemplateName = "Unknown"
	cr.PageName = "Unknown"

	if t, err := GetTemplate(cr.TemplateID); err == nil {
		cr.TemplateName = t.Name
	}

	if p, err := GetPage(cr.PageID); err == nil {
		cr.PageName = p.Name
	}

	err = db.Table("results").Where("campaign_id=?", cr.Id).Find(&cr.Results).Error

	if err != nil {
		log.Errorf("%s: results not found for campaign", err)
		return cr, err
	}

	err = db.Table("events").Where("campaign_id=?", cr.Id).Find(&cr.Events).Error

	if err != nil {
		log.Errorf("%s: events not found for campaign", err)
		return cr, err
	}

	return cr, err
}

// GetQueuedCampaigns returns the campaigns that are queued up for this given minute
func GetQueuedCampaigns(t time.Time) ([]Campaign, error) {
	cs := []Campaign{}
	err := db.Where("launch_date <= ?", t).
		Where("status = ?", CAMPAIGN_QUEUED).Find(&cs).Error
	if err != nil {
		log.Error(err)
	}
	log.Infof("Found %d Campaigns to run\n", len(cs))
	for i := range cs {
		err = cs[i].getDetails()
		if err != nil {
			log.Error(err)
		}
	}
	return cs, err
}

// PostCampaign inserts a campaign and all associated records into the database.
func PostCampaign(c *Campaign, uid int64) (err error) {
	if err = c.Validate(); err != nil {
		return
	}

	var clickersGroup Group
	clickersGroupCreated := false

	defer func() {
		if err != nil && clickersGroupCreated {
			err := DeleteGroup(&clickersGroup)

			if err != nil {
				log.Errorf("Could not delete clickers group after failed creation of a campaign")
			}
		}
	}()

	if c.ClickersGroup != "" {
		clickersGroup, err = CreateEmptyGroup(c.ClickersGroup, uid)

		if err != nil {
			log.Error(err)
			return
		}

		clickersGroupCreated = true
		c.ClickersGroupId = clickersGroup.Id
	} else if c.ClickersGroupId != 0 {
		for _, g := range c.Groups {
			if c.ClickersGroupId == g.Id {
				return fmt.Errorf("Clickers cannot be added to this group - %s", g.Name)
			}
		}
	}

	// Fill in the details
	c.UserId = uid
	c.CreatedDate = time.Now().UTC()
	c.CompletedDate = time.Time{}
	c.Status = CAMPAIGN_QUEUED
	if c.LaunchDate.IsZero() {
		c.LaunchDate = c.CreatedDate
	} else {
		c.LaunchDate = c.LaunchDate.UTC()
	}

	if delay, _ := strconv.ParseInt(os.Getenv("DELAY_SENDING_BY_X_MINS"), 10, 0); delay > 0 {
		c.LaunchDate = c.LaunchDate.Add(time.Duration(delay) * time.Minute)
	}

	if !c.SendByDate.IsZero() {
		c.SendByDate = c.SendByDate.UTC()
	}
	if c.LaunchDate.Before(c.CreatedDate) || c.LaunchDate.Equal(c.CreatedDate) {
		c.Status = CAMPAIGN_IN_PROGRESS
	}
	// Check to make sure all the groups already exist
	// Also, later we'll need to know the total number of recipients (counting
	// duplicates is ok for now), so we'll do that here to save a loop.
	totalRecipients := 0
	for i, g := range c.Groups {
		c.Groups[i], err = GetGroupByName(g.Name, uid)
		if err == gorm.ErrRecordNotFound {
			log.WithFields(logrus.Fields{
				"group": g.Name,
			}).Error("Group does not exist")
			return ErrGroupNotFound
		} else if err != nil {
			log.Error(err)
			return err
		}
		totalRecipients += len(c.Groups[i].Targets)
	}
	// Check to make sure the template exists
	t, err := GetTemplateByName(c.Template.Name, uid)
	if err == gorm.ErrRecordNotFound {
		log.WithFields(logrus.Fields{
			"template": t.Name,
		}).Error("Template does not exist")
		return ErrTemplateNotFound
	} else if err != nil {
		log.Error(err)
		return
	}
	c.Template = t
	c.TemplateId = t.Id
	// Check to make sure the page exists
	p, err := GetPageByName(c.Page.Name, uid)
	if err == gorm.ErrRecordNotFound {
		log.WithFields(logrus.Fields{
			"page": p.Name,
		}).Error("Page does not exist")
		return ErrPageNotFound
	} else if err != nil {
		log.Error(err)
		return
	}
	c.Page = p
	c.PageId = p.Id
	// Check to make sure the sending profile exists
	s, err := GetSMTPByName(c.SMTP.Name)
	if err == gorm.ErrRecordNotFound {
		log.WithFields(logrus.Fields{
			"smtp": s.Name,
		}).Error("Sending profile does not exist")
		return ErrSMTPNotFound
	} else if err != nil {
		log.Error(err)
		return
	}
	c.SMTP = s
	c.SMTPId = s.Id
	// Insert into the DB
	err = db.Save(c).Error
	if err != nil {
		log.Error(err)
		return
	}
	err = c.AddEvent(&Event{Message: "Campaign Created"})
	if err != nil {
		log.Error(err)
	}
	// Insert all the results
	resultMap := make(map[string]bool)
	recipientIndex := 0
	for _, g := range c.Groups {
		// Insert a result for each target in the group
		for _, t := range g.Targets {
			// Remove duplicate results - we should only
			// send emails to unique email addresses.
			if _, ok := resultMap[t.Email]; ok {
				continue
			}
			resultMap[t.Email] = true
			sendDate := c.generateSendDate(recipientIndex, totalRecipients)
			r := &Result{
				BaseRecipient: BaseRecipient{
					Email:     t.Email,
					Position:  t.Position,
					FirstName: t.FirstName,
					LastName:  t.LastName,
				},
				Status:       STATUS_SCHEDULED,
				CampaignId:   c.Id,
				UserId:       c.UserId,
				SendDate:     sendDate,
				Reported:     false,
				ModifiedDate: c.CreatedDate,
			}
			if r.SendDate.Before(c.CreatedDate) || r.SendDate.Equal(c.CreatedDate) {
				r.Status = STATUS_SENDING
			}
			err = r.GenerateId()
			if err != nil {
				log.Error(err)
				continue
			}
			err = db.Save(r).Error
			if err != nil {
				log.WithFields(logrus.Fields{
					"email": t.Email,
				}).Error(err)
			}
			c.Results = append(c.Results, *r)
			log.Infof("Creating maillog for %s to send at %s\n", r.Email, sendDate)
			err = GenerateMailLog(c, r, sendDate)
			if err != nil {
				log.Error(err)
				continue
			}
			recipientIndex++
		}
	}
	err = db.Save(c).Error
	return
}

//DeleteCampaign deletes the specified campaign
func DeleteCampaign(id int64) error {
	log.WithFields(logrus.Fields{
		"campaign_id": id,
	}).Info("Deleting campaign")
	// Delete all the campaign results
	err := db.Where("campaign_id=?", id).Delete(&Result{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Where("campaign_id=?", id).Delete(&Event{}).Error
	if err != nil {
		log.Error(err)
		return err
	}

	err = db.Where("campaign_id=?", id).Delete(&MailLog{}).Error
	if err != nil {
		log.Error(err)
		return err
	}

	// Delete the campaign
	err = db.Delete(&Campaign{Id: id}).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// DeleteUserCampaigns deletes campaigns created by user with the given uid
func DeleteUserCampaigns(uid int64) error {
	var ids []int64

	err := db.
		Model(&Campaign{}).
		Where("user_id=?", uid).
		Pluck("id", &ids).Error

	if err != nil {
		return fmt.Errorf(
			"Couldn't find ids of campaigns owned by user with id %d - %s",
			uid, err.Error(),
		)
	}

	var errs []error

	for _, id := range ids {
		err = DeleteCampaign(id)

		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf(
			"Couldn't delete %d campaign(s) owned by user with id %d",
			len(errs), uid,
		)
	}

	return nil
}

// CompleteCampaign effectively "ends" a campaign.
// Any future emails clicked will return a simple "404" page.
func CompleteCampaign(id int64) error {
	log.WithFields(logrus.Fields{
		"campaign_id": id,
	}).Info("Marking campaign as complete")
	c, err := GetCampaign(id)
	if err != nil {
		return err
	}
	// Don't overwrite original completed time
	if c.Status == CAMPAIGN_COMPLETE {
		return nil
	}
	// Mark the campaign as complete
	c.CompletedDate = time.Now().UTC()
	c.Status = CAMPAIGN_COMPLETE
	err = db.Set("gorm:association_autocreate", false).Where("id=?", id).Save(&c).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// ProcessCampaignTargets moves or copies clickers and non-clickers into different groups
// according to settings of a campaign with the given id
func ProcessCampaignTargets(id int64) error {
	c, err := GetCampaign(id)

	if err != nil {
		err = fmt.Errorf("Could not find campaign with id %d: %s", id, err.Error())
		log.Error(err)
		return err
	}

	if !c.RemoveNonClickers && c.ClickersGroupId == 0 {
		return nil
	}

	if c.ClickersGroupId != 0 {
		clickers, err := c.GetClickers()

		if err != nil {
			log.Error(err)
			return err
		}

		g, err := GetGroup(c.ClickersGroupId)

		if err != nil {
			err = fmt.Errorf("Couldn't find group %d: %s", c.ClickersGroupId, err.Error())
			log.Error(err)
			return err
		}

		err = (&g).AddTargets(clickers)

		if err != nil {
			log.Error(err)
			return err
		}
	}

	if c.RemoveNonClickers {
		nonclickers, err := c.GetNonClickers()

		if err != nil {
			log.Error(err)
			return err
		}

		groups := map[int64][]Target{}

		for _, nc := range nonclickers {
			gids, err := nc.GetGroupIds()

			if err != nil {
				err = fmt.Errorf("Could not get group ids of target %d: %s", nc.Id, err.Error())
				log.Error(err)
				return err
			}

			if len(gids) == 0 {
				continue
			}

			gid := gids[0]

			if _, ok := groups[gid]; ok {
				groups[gid] = append(groups[gid], nc)
			} else {
				groups[gid] = []Target{nc}
			}
		}

		for gid, ts := range groups {
			g, err := GetGroup(gid)

			if err != nil {
				err = fmt.Errorf("Could not find group %d: %s", gid, err.Error())
				log.Error(err)
				return err
			}

			err = g.RemoveTargets(ts)

			if err != nil {
				log.Error(err)
				return err
			}
		}
	}

	return nil
}

// AfterFind decrypts encrypted passwords stored in event details
func (e *Event) AfterFind() (err error) {
	var detailsWithPassword struct {
		Payload struct {
			Password []string `json:"password"`
		} `json:"payload"`
	}

	if e.Details != "" {
		err := json.Unmarshal([]byte(e.Details), &detailsWithPassword)

		if err != nil {
			return nil
		}

		if encPwd := detailsWithPassword.Payload.Password; len(encPwd) > 0 {
			pwd, err := bakery.Decrypt(encPwd[0])

			if err != nil {
				return nil
			}

			e.Details = strings.Replace(e.Details, encPwd[0], pwd, 1)
		}
	}

	return
}

// AfterCreate creates a new user with "lms_user" role whenever EVENT_CLICKED event occurs,
// additionally links newly created LMS users to relevant LMS campaigns.
func (e *Event) AfterCreate(tx *gorm.DB) error {
	if e.Message != EVENT_CLICKED {
		return nil
	}

	tx.Commit()
	coid, err := GetCampaignOwnerId(e.CampaignId)

	if err != nil {
		log.Errorf("Could not determine owner id of campaign %d - %s", e.CampaignId, err.Error())
		return nil
	}

	campaignOwner, err := GetUser(coid)

	if err != nil {
		log.Errorf("Could not find owner of campaign %d - %s", e.CampaignId, err.Error())
		return nil
	}

	if !campaignOwner.IsSubscribed() {
		return nil
	}

	partner := campaignOwner.Id

	if campaignOwner.IsChildUser() {
		partner = campaignOwner.Partner
	}

	fullname := GetTargetsFullName(e.Email, coid)
	username := util.GenerateUsername(fullname, e.Email)

	u, err := CreateUser(
		username, fullname,
		e.Email, "qwerty", LMSUser, partner,
	)

	if err != nil {
		log.Errorf("Could not create LMS user - %s", err.Error())
		return nil
	}

	if os.Getenv("USERSYNC_DISABLE") == "" {
		uid, err := usersync.PushUser(
			u.Id,
			u.Username,
			u.Email,
			u.FullName,
			"qwerty",
			LMSUser,
			GetUserBakeryID(partner),
			false,
		)

		if err != nil {
			_, _ = DeleteUser(u.Id)
			log.Errorf("Could not push user to the main server - %s", err.Error())
			return nil
		}

		err = (*u).SetBakeryUserID(uid)

		if err != nil {
			log.Error(err)
		}
	}

	lmsCampaigns, err := GetLinkedLMSCampaigns(e.CampaignId)

	if len(lmsCampaigns) == 0 {
		return nil
	}

	for _, c := range lmsCampaigns {
		if err := c.AddUser(u.Id); err != nil {
			log.Error(err)
		}
	}

	return nil
}
