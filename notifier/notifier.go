package notifier

import (
	"os"

	log "github.com/everycloud-technologies/phishing-simulation/logger"
	m "github.com/keighl/mandrill"
)

var client *m.Client

func init() {
	client = m.ClientWithKey(os.Getenv("MANDRILL_KEY"))
}

// SendWelcomeEmail sends a welcome email to the given address.
// If partner flag is true then the partner email template will be used,
// or the customer template otherwise.
func SendWelcomeEmail(email, name, username string, partner bool) {
	message := &m.Message{}
	message.AddRecipient(email, name, "to")
	message.FromEmail = "donotreply@everycloudtech.com"
	message.FromName = "EveryCloud Technologies LLC"
	message.Subject = "Welcome from EveryCloud"

	message.MergeVars = []*m.RcptMergeVars{
		m.MapToRecipientVars(email, map[string]interface{}{
			"FIRST_NAME": name,
			"USERNAME":   username,
		}),
	}

	template := "sat-free-phish-welcome-eu"

	if partner {
		template = "sat-free-phish-welcome-partner"
	}

	_, err := client.MessagesSendTemplate(message, template, nil)

	if err != nil {
		log.
			WithFields(map[string]interface{}{"tag": "notifier"}).
			Errorf("Could not send welcome email to %s - %s", email, err.Error())
	}
}

// SendAccountUpgradeEmail sends an account upgrade notification email to the given address.
func SendAccountUpgradeEmail(email, name string) {
	message := &m.Message{}
	message.AddRecipient(email, name, "to")
	message.FromEmail = "donotreply@everycloudtech.com"
	message.FromName = "EveryCloud Technologies LLC"
	message.Subject = "Account upgraded"

	message.MergeVars = []*m.RcptMergeVars{
		m.MapToRecipientVars(email, map[string]interface{}{
			"FIRST_NAME": name,
		}),
	}

	_, err := client.MessagesSendTemplate(message, "sat-account-upgraded", nil)

	if err != nil {
		log.
			WithFields(map[string]interface{}{"tag": "notifier"}).
			Errorf("Could not send account upgrade notification email to %s - %s", email, err.Error())
	}
}
