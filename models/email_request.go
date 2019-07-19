package models

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/mail"
	"strings"

	"github.com/everycloud-technologies/phishing-simulation/config"
	"github.com/everycloud-technologies/phishing-simulation/encryption"
	log "github.com/everycloud-technologies/phishing-simulation/logger"
	"github.com/everycloud-technologies/phishing-simulation/mailer"
	"github.com/gophish/gomail"
)

// PreviewPrefix is the standard prefix added to the rid parameter when sending
// test emails.
const PreviewPrefix = "preview-"

// EmailRequest is the structure of a request
// to send a test email to test an SMTP connection.
// This type implements the mailer.Mail interface.
type EmailRequest struct {
	Id          int64        `json:"-"`
	Template    Template     `json:"template"`
	TemplateId  int64        `json:"-"`
	Page        Page         `json:"page"`
	PageId      int64        `json:"-"`
	SMTP        SMTP         `json:"smtp"`
	URL         string       `json:"url"`
	Tracker     string       `json:"tracker" gorm:"-"`
	TrackingURL string       `json:"tracking_url" gorm:"-"`
	UserId      int64        `json:"-"`
	ErrorChan   chan (error) `json:"-" gorm:"-"`
	RId         string       `json:"id"`
	FromAddress string       `json:"from_address"`
	BaseRecipient
}

func (s *EmailRequest) getBaseURL() string {
	return s.URL
}

func (s *EmailRequest) getFromAddress() string {
	return s.FromAddress
}

// Validate ensures the SendTestEmailRequest structure
// is valid.
func (s *EmailRequest) Validate() error {
	switch {
	case s.Email.String() == "":
		return ErrEmailNotSpecified
	case s.FromAddress == "" && s.SMTP.FromAddress == "":
		return ErrFromAddressNotSpecified
	}
	return nil
}

// Backoff treats temporary errors as permanent since this is expected to be a
// synchronous operation. It returns any errors given back to the ErrorChan
func (s *EmailRequest) Backoff(reason error) error {
	s.ErrorChan <- reason
	return nil
}

// Error returns an error on the ErrorChan.
func (s *EmailRequest) Error(err error) error {
	s.ErrorChan <- err
	return nil
}

// Success returns nil on the ErrorChan to indicate that the email was sent
// successfully.
func (s *EmailRequest) Success() error {
	s.ErrorChan <- nil
	return nil
}

// PostEmailRequest stores a SendTestEmailRequest in the database.
func PostEmailRequest(s *EmailRequest) error {
	// Generate an ID to be used in the underlying Result object
	rid, err := generateResultId()
	if err != nil {
		return err
	}
	s.RId = fmt.Sprintf("%s%s", PreviewPrefix, rid)
	return db.Save(&s).Error
}

// GetEmailRequestByResultId retrieves the EmailRequest by the underlying rid
// parameter.
func GetEmailRequestByResultId(id string) (EmailRequest, error) {
	s := EmailRequest{}
	err := db.Table("email_requests").Where("r_id=?", id).First(&s).Error
	return s, err
}

// Generate fills in the details of a gomail.Message with the contents
// from the SendTestEmailRequest.
func (s *EmailRequest) Generate(msg *gomail.Message) error {
	f, err := mail.ParseAddress(s.FromAddress)
	if err != nil {
		return err
	}
	fn := f.Name
	if fn == "" {
		fn = f.Address
	}
	msg.SetAddressHeader("From", f.Address, f.Name)

	ptx, err := NewPhishingTemplateContext(s, s.BaseRecipient, s.RId)
	if err != nil {
		return err
	}

	url, err := ExecuteTemplate(s.URL, ptx)
	if err != nil {
		return err
	}
	s.URL = url

	// Add the transparency headers
	msg.SetHeader("X-Sender", "X-PHISHTEST")
	msg.SetHeader("X-Mailer", "EveryCloud")
	if config.Conf.ContactAddress != "" {
		msg.SetHeader("X-Gophish-Contact", config.Conf.ContactAddress)
	}

	// Parse the customHeader templates
	for _, header := range s.SMTP.Headers {
		key, err := ExecuteTemplate(header.Key, ptx)
		if err != nil {
			log.Error(err)
		}

		value, err := ExecuteTemplate(header.Value, ptx)
		if err != nil {
			log.Error(err)
		}

		// Add our header immediately
		msg.SetHeader(key, value)
	}

	// Parse remaining templates
	subject, err := ExecuteTemplate(s.Template.Subject, ptx)
	if err != nil {
		log.Error(err)
	}
	// don't set the Subject header if it is blank
	if len(subject) != 0 {
		msg.SetHeader("Subject", subject)
	}

	msg.SetHeader("To", s.FormatAddress())
	if s.Template.Text != "" {
		text, err := ExecuteTemplate(s.Template.Text, ptx)
		if err != nil {
			log.Error(err)
		}
		msg.SetBody("text/plain", text)
	}
	if s.Template.HTML != "" {
		html, err := ExecuteTemplate(s.Template.HTML, ptx)
		if err != nil {
			log.Error(err)
		}
		if s.Template.Text == "" {
			msg.SetBody("text/html", html)
		} else {
			msg.AddAlternative("text/html", html)
		}
	}
	// Attach the files
	for _, a := range s.Template.Attachments {
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

// GetDialer returns the mailer.Dialer for the underlying SMTP object
func (s *EmailRequest) GetDialer() (mailer.Dialer, error) {
	return s.SMTP.GetDialer()
}

// DeleteUserEmailRequests deletes email requests created by user with the given uid
func DeleteUserEmailRequests(uid int64) error {
	err := db.Where("user_id=?", uid).Delete(&EmailRequest{}).Error

	if err != nil {
		return fmt.Errorf(
			"Couldn't delete one or more email requests related to user with id %d - %s",
			uid, err.Error(),
		)
	}

	return nil
}

// EncryptRequestEmails encrypts email column in email_requests table
func EncryptRequestEmails() {
	log.Info("Encrypting emails in email_requests table...")

	type req struct {
		ID    int64  `json:"id"`
		Email string `json:"email" sql:"not null;unique"`
	}

	reqs := []req{}

	err := db.
		Table("email_requests").
		Where(`email LIKE "%@%"`).
		Find(&reqs).
		Error

	if err != nil {
		log.Error(err)
		return
	}

	for _, r := range reqs {
		email, err := encryption.Encrypt(r.Email)

		if err != nil {
			log.Error(err)
			return
		}

		err = db.
			Table("email_requests").
			Where("id = ?", r.ID).
			UpdateColumns(req{Email: email}).
			Error

		if err != nil {
			log.Error(err)
			return
		}

		log.Infof("Encrypted email of email_request with id %d", r.ID)
	}

	log.Info("Done.")
}
