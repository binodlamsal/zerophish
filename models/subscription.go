package models

import (
	"encoding/json"
	"time"

	log "github.com/binodlamsal/zerophish/logger"
	"github.com/sirupsen/logrus"
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

// DeleteSubscription deletes given subscription
func DeleteSubscription(s *Subscription) error {
	log.WithFields(logrus.Fields{
		"subscription_id": s.Id,
	}).Info("Deleting subscription")

	// Delete the campaign
	err = db.Delete(s).Error

	if err != nil {
		log.Error(err)
	}

	GetCache().DeleteUserSubscription(s)
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

// IsActive tells if this subscription if active (not expired)
func (s *Subscription) IsActive() bool {
	return s.ExpirationDate.After(time.Now().UTC())
}

// ChangePlan changes this subscription's plan
func (s *Subscription) ChangePlan(planId int64) error {
	s.PlanId = planId
	err := db.Save(s).Error

	if err != nil {
		log.Error(err)
	}

	return err
}

// ChangeExpirationDate changes this subscription's expiration date
func (s *Subscription) ChangeExpirationDate(expDate time.Time) error {
	s.ExpirationDate = expDate
	err := db.Save(s).Error

	if err != nil {
		log.Error(err)
	}

	return err
}

func (s *Subscription) BeforeSave() error {
	GetCache().DeleteUserSubscription(s)
	return nil
}

// MarshalJSON is a custom JSON marshaller with support of a few computed props
func (s Subscription) MarshalJSON() ([]byte, error) {
	type jsonSubscription struct {
		Id             int64     `json:"id"`
		UserId         int64     `json:"user_id"`
		PlanId         int64     `json:"plan_id"`
		Plan           string    `json:"plan"`
		ExpirationDate time.Time `json:"expiration_date"`
		Expired        bool      `json:"expired"`
	}

	expired := true
	u, err := GetUser(s.UserId)

	if err == nil {
		expired = !u.IsSubscribed()
	}

	return json.Marshal(jsonSubscription{
		Id:             s.Id,
		UserId:         s.UserId,
		PlanId:         s.PlanId,
		Plan:           GetPlanNameById(s.PlanId),
		ExpirationDate: s.ExpirationDate,
		Expired:        expired,
	})
}
