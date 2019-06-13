package models

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"net/mail"
	"strings"
	"time"

	"github.com/everycloud-technologies/phishing-simulation/config"
	log "github.com/everycloud-technologies/phishing-simulation/logger"
	"github.com/everycloud-technologies/phishing-simulation/mailer"
	"github.com/everycloud-technologies/phishing-simulation/util"
	"github.com/gophish/gomail"
)

// MaxSendAttempts set to 8 since we exponentially backoff after each failed send
// attempt. This will give us a maximum send delay of 256 minutes, or about 4.2 hours.
var MaxSendAttempts = 8

// ErrMaxSendAttempts is thrown when the maximum number of sending attemps for a given
// MailLog is exceeded.
var ErrMaxSendAttempts = errors.New("max send attempts exceeded")

// MailLog is a struct that holds information about an email that is to be
// sent out.
type MailLog struct {
	Id          int64     `json:"-"`
	UserId      int64     `json:"-"`
	CampaignId  int64     `json:"campaign_id"`
	RId         string    `json:"id"`
	SendDate    time.Time `json:"send_date"`
	SendAttempt int       `json:"send_attempt"`
	Processing  bool      `json:"-"`
}

// MailLogs is a list of MailLog's
type MailLogs []*MailLog

// GenerateMailLog creates a new maillog for the given campaign and
// result. It sets the initial send date to match the campaign's launch date.
func GenerateMailLog(c *Campaign, r *Result, sendDate time.Time) error {
	m := &MailLog{
		UserId:     c.UserId,
		CampaignId: c.Id,
		RId:        r.RId,
		SendDate:   sendDate,
	}
	err = db.Save(m).Error
	return err
}

// Backoff sets the MailLog SendDate to be the next entry in an exponential
// backoff. ErrMaxRetriesExceeded is thrown if this maillog has been retried
// too many times. Backoff also unlocks the maillog so that it can be processed
// again in the future.
func (m *MailLog) Backoff(reason error) error {
	r, err := GetResult(m.RId)
	if err != nil {
		return err
	}
	if m.SendAttempt == MaxSendAttempts {
		r.HandleEmailError(ErrMaxSendAttempts)
		return ErrMaxSendAttempts
	}
	// Add an error, since we had to backoff because of a
	// temporary error of some sort during the SMTP transaction
	m.SendAttempt++
	backoffDuration := math.Pow(2, float64(m.SendAttempt))
	m.SendDate = m.SendDate.Add(time.Minute * time.Duration(backoffDuration))
	err = db.Save(m).Error
	if err != nil {
		return err
	}
	err = r.HandleEmailBackoff(reason, m.SendDate)
	if err != nil {
		return err
	}
	err = m.Unlock()
	return err
}

// Unlock removes the processing flag so the maillog can be processed again
func (m *MailLog) Unlock() error {
	m.Processing = false
	return db.Save(&m).Error
}

// Lock sets the processing flag so that other processes cannot modify the maillog
func (m *MailLog) Lock() error {
	m.Processing = true
	return db.Save(&m).Error
}

// Error sets the error status on the models.Result that the
// maillog refers to. Since MailLog errors are permanent,
// this action also deletes the maillog.
func (m *MailLog) Error(e error) error {
	r, err := GetResult(m.RId)
	if err != nil {
		log.Warn(err)
		return err
	}
	err = r.HandleEmailError(e)
	if err != nil {
		log.Warn(err)
		return err
	}
	err = db.Delete(m).Error
	return err
}

// Success deletes the maillog from the database and updates the underlying
// campaign result.
func (m *MailLog) Success() error {
	r, err := GetResult(m.RId)
	if err != nil {
		return err
	}
	err = r.HandleEmailSent()
	if err != nil {
		return err
	}
	err = db.Delete(m).Error
	return nil
}

// GetDialer returns a dialer based on the maillog campaign's SMTP configuration
func (m *MailLog) GetDialer() (mailer.Dialer, error) {
	c, err := GetCampaign(m.CampaignId)
	if err != nil {
		return nil, err
	}
	return c.SMTP.GetDialer()
}

// Generate fills in the details of a gomail.Message instance with
// the correct headers and body from the campaign and recipient listed in
// the maillog. We accept the gomail.Message as an argument so that the caller
// can choose to re-use the message across recipients.
func (m *MailLog) Generate(msg *gomail.Message) error {
	r, err := GetResult(m.RId)
	if err != nil {
		return err
	}
	c, err := GetCampaign(m.CampaignId)
	if err != nil {
		return err
	}

	var from *mail.Address

	if from, err = mail.ParseAddress(c.FromAddress); err != nil {
		if from, err = mail.ParseAddress(c.Template.FromAddress); err != nil {
			if from, err = mail.ParseAddress(c.SMTP.FromAddress); err != nil {
				return err
			}
		}
	}

	msg.SetAddressHeader("From", from.Address, from.Name)

	ptx, err := NewPhishingTemplateContext(&c, r.BaseRecipient, r.RId)
	if err != nil {
		return err
	}

	// Add the transparency headers
	msg.SetHeader("X-Sender", "X-PHISHTEST")
	msg.SetHeader("X-Mailer", config.ServerName)
	if config.Conf.ContactAddress != "" {
		msg.SetHeader("X-Gophish-Contact", config.Conf.ContactAddress)
	}
	// Parse the customHeader templates
	for _, header := range c.SMTP.Headers {
		key, err := ExecuteTemplate(header.Key, ptx)
		if err != nil {
			log.Warn(err)
		}

		value, err := ExecuteTemplate(header.Value, ptx)
		if err != nil {
			log.Warn(err)
		}

		// Add our header immediately
		msg.SetHeader(key, value)
	}

	// Parse remaining templates
	subject, err := ExecuteTemplate(c.Template.Subject, ptx)
	if err != nil {
		log.Warn(err)
	}
	// don't set Subject header if the subject is empty
	if len(subject) != 0 {
		msg.SetHeader("Subject", subject)
	}

	msg.SetHeader("To", r.FormatAddress())
	if c.Template.Text != "" {
		text, err := ExecuteTemplate(c.Template.Text, ptx)
		if err != nil {
			log.Warn(err)
		}
		msg.SetBody("text/plain", text)
	}
	if c.Template.HTML != "" {
		html, err := ExecuteTemplate(c.Template.HTML, ptx)
		if err != nil {
			log.Warn(err)
		}
		if c.Template.Text == "" {
			msg.SetBody("text/html", html)
		} else {
			msg.AddAlternative("text/html", html)
		}
	}
	// Attach the files
	for _, a := range c.Template.Attachments {
		msg.Attach(func(a Attachment) (string, gomail.FileSetting, gomail.FileSetting) {
			h := map[string][]string{"Content-ID": {fmt.Sprintf("<%s>", a.Name)}}
			return a.Name, gomail.SetCopyFunc(func(w io.Writer) error {
				decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(a.Content))
				_, err = io.Copy(w, decoder)
				return err
			}), gomail.SetHeader(h)
		}(a))
	}

	return nil
}

// IsTimeToSend tells if this e-mail should be sent right now,
// current UTC time (t) and campaign business hours (if any) are taken into account
func (m *MailLog) IsTimeToSend(t time.Time) bool {
	c := Campaign{}

	if db.Where("id = ?", m.CampaignId).First(&c).Error != nil {
		log.Errorf("%s: campaign not found", err)
		return false
	}

	if m.SendDate.Before(t) {
		if c.StartTime != "" && c.EndTime != "" && c.TimeZone != "" {
			if !util.IsLocalBusinessTime(t, c.StartTime, c.EndTime, c.TimeZone) {
				return false
			}
		}

		return true
	}

	return false
}

// WithinBusinessHours returns mail logs with "send_date" within respective campaign's business hours
func (mls MailLogs) WithinBusinessHours(t time.Time) MailLogs {
	selectedMailLogs := MailLogs{}

	for _, ml := range mls {
		c := Campaign{}

		if db.Where("id = ?", ml.CampaignId).First(&c).Error != nil {
			log.Errorf("%s: campaign not found", err)
			continue
		}

		if c.StartTime != "" && c.EndTime != "" && c.TimeZone != "" {
			if !util.IsLocalBusinessTime(t, c.StartTime, c.EndTime, c.TimeZone) {
				continue
			}
		}

		selectedMailLogs = append(selectedMailLogs, ml)
	}

	return selectedMailLogs
}

// GetQueuedMailLogs returns the mail logs that are queued up for the given minute.
func GetQueuedMailLogs(t time.Time) (MailLogs, error) {
	ms := MailLogs{}
	err := db.Where("send_date <= ? AND processing = ?", t, false).
		Find(&ms).Error
	if err != nil {
		log.Warn(err)
	}
	return ms.WithinBusinessHours(t), err
}

// GetMailLogsByCampaign returns all of the mail logs for a given campaign.
func GetMailLogsByCampaign(cid int64) ([]*MailLog, error) {
	ms := []*MailLog{}
	err := db.Where("campaign_id = ?", cid).Find(&ms).Error
	return ms, err
}

// LockMailLogs locks or unlocks a slice of maillogs for processing.
func LockMailLogs(ms []*MailLog, lock bool) error {
	tx := db.Begin()
	for i := range ms {
		ms[i].Processing = lock
		err := tx.Save(ms[i]).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

// UnlockAllMailLogs removes the processing lock for all maillogs
// in the database. This is intended to be called when Gophish is started
// so that any previously locked maillogs can resume processing.
func UnlockAllMailLogs() error {
	err = db.Model(&MailLog{}).Update("processing", false).Error
	return err
}
