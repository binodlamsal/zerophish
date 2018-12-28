package models

import (
	"time"

	log "github.com/binodlamsal/gophish/logger"
)

// Subscription is a paid subscription for a certain plan
type Subscription struct {
	Id             int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId         int64     `json:"user_id"`
	PlanId         int64     `json:"plan_id"`
	ExpirationDate time.Time `json:"expiration_date"`
}

// PostSubscription creates a new subscription
func PostSubscription(s *Subscription) error {
	err := db.Save(s).Error

	if err != nil {
		log.Error(err)
	}

	return err
}

// GetSubscriptions returns all subscriptions
func GetSubscriptions() ([]Subscription, error) {
	subscriptions := []Subscription{}

	if err := db.Find(&subscriptions).Error; err != nil {
		log.Error(err)
		return subscriptions, err
	}

	return subscriptions, err
}
