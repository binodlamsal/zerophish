package models

import (
	"fmt"
	"time"
)

// LMSCampaign struct represents an LMS campaign
type LMSCampaign struct {
	Id                int64     `json:"id"`
	Title             string    `json:"title"`
	AuthorId          int64     `json:"author_id"`
	PubliclyAvailable bool      `json:"publicly_available"`
	QAPassPercentage  int64     `json:"qa_pass_percentage"`
	PhishingCampaign  int64     `json:"phishing_campaign"`
	EnableQA          bool      `json:"enable_qa"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// LMSCampaignUser stuct represents a relation between a gophish "lms_user" and an LMS campaign
type LMSCampaignUser struct {
	Id         int64     `json:"id"`
	UserId     int64     `json:"user_id"`
	CampaignId int64     `json:"campaign_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// GetLinkedLMSCampaigns finds and returns LMS campaigns linked to a given campaign id
func GetLinkedLMSCampaigns(cid int64) ([]LMSCampaign, error) {
	lmsCampaigns := []LMSCampaign{}

	if err := db.Table("lms_campaigns").Where("phishing_campaign = ?", cid).Scan(&lmsCampaigns).Error; err != nil {
		return lmsCampaigns, err
	}

	return lmsCampaigns, nil
}

// AddUser creates a record in lms_campaign_users table linking the given user id to this LMS campaign
func (c *LMSCampaign) AddUser(uid int64) error {
	if c.HasUser(uid) {
		return nil
	}

	user := LMSCampaignUser{
		UserId:     uid,
		CampaignId: c.Id,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := db.Table("lms_campaign_users").Save(&user).Error

	if err != nil {
		return fmt.Errorf("Could not add LMS user to LMS campaign %d - %s", c.Id, err.Error())
	}

	return nil
}

// HasUser tells if this LMS campaign has user with the given id linked to it
func (c *LMSCampaign) HasUser(uid int64) bool {
	var count int64

	db.
		Table("lms_campaign_users").
		Where("user_id = ? AND campaign_id = ?", uid, c.Id).
		Count(&count)

	return count > 0
}
