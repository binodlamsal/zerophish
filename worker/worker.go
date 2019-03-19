package worker

import (
	"time"

	log "github.com/everycloud-technologies/phishing-simulation/logger"
	"github.com/everycloud-technologies/phishing-simulation/mailer"
	"github.com/everycloud-technologies/phishing-simulation/models"
	"github.com/sirupsen/logrus"
)

// Worker is the background worker that handles watching for new campaigns and sending emails appropriately.
type Worker struct{}

// New creates a new worker object to handle the creation of campaigns
func New() *Worker {
	return &Worker{}
}

// Start launches the worker to poll the database every minute for any pending maillogs
// that need to be processed.
func (w *Worker) Start() {
	log.Info("Background Worker Started Successfully - Waiting for Campaigns")
	for t := range time.Tick(1 * time.Minute) {
		ms, err := models.GetQueuedMailLogs(t.UTC())
		if err != nil {
			log.Error(err)
			continue
		}
		// Lock the MailLogs (they will be unlocked after processing)
		err = models.LockMailLogs(ms, true)
		if err != nil {
			log.Error(err)
			continue
		}
		// We'll group the maillogs by campaign ID to (sort of) group
		// them by sending profile. This lets the mailer re-use the Sender
		// instead of having to re-connect to the SMTP server for every
		// email.
		msg := make(map[int64][]mailer.Mail)
		for _, m := range ms {
			msg[m.CampaignId] = append(msg[m.CampaignId], m)
		}

		// Next, we process each group of maillogs in parallel
		for cid, msc := range msg {
			go func(cid int64, msc []mailer.Mail) {
				c, err := models.GetCampaign(cid)
				if err != nil {
					log.Error(err)
					errorMail(err, msc)
					return
				}
				if c.Status == models.CAMPAIGN_QUEUED {
					err := c.UpdateStatus(models.CAMPAIGN_IN_PROGRESS)
					if err != nil {
						log.Error(err)
						return
					}
				}
				log.WithFields(logrus.Fields{
					"num_emails": len(msc),
				}).Info("Sending emails to mailer for processing")
				mailer.Mailer.Queue <- msc
			}(cid, msc)
		}
	}
}

// LaunchCampaign starts a campaign
func (w *Worker) LaunchCampaign(c models.Campaign) {
	ms, err := models.GetMailLogsByCampaign(c.Id)
	if err != nil {
		log.Error(err)
		return
	}
	models.LockMailLogs(ms, true)
	// This is required since you cannot pass a slice of values
	// that implements an interface as a slice of that interface.
	mailEntries := []mailer.Mail{}
	currentTime := time.Now().UTC()
	for _, m := range ms {
		// Only send the emails scheduled to be sent for the past minute to
		// respect the campaign scheduling options
		if !m.IsTimeToSend(currentTime) {
			m.Unlock()
			continue
		}
		mailEntries = append(mailEntries, m)
	}

	if len(mailEntries) > 0 {
		mailer.Mailer.Queue <- mailEntries
	}
}

// SendTestEmail sends a test email
func (w *Worker) SendTestEmail(s *models.EmailRequest) error {
	go func() {
		ms := []mailer.Mail{s}
		mailer.Mailer.Queue <- ms
	}()
	return <-s.ErrorChan
}

// errorMail is a helper to handle erroring out a slice of Mail instances
// in the case that an unrecoverable error occurs.
func errorMail(err error, ms []mailer.Mail) {
	for _, m := range ms {
		m.Error(err)
	}
}
