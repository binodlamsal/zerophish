package models

import log "github.com/binodlamsal/gophish/logger"

// Plan is a subscription plan
type Plan struct {
	Id   int64  `json:"id" gorm:"column:id; primary_key:yes"`
	Name string `json:"name"`
}

// PostPlan creates a new plan
func PostPlan(p *Plan) error {
	err := db.Save(p).Error

	if err != nil {
		log.Error(err)
	}

	return err
}

// GetPlans returns all plans
func GetPlans() ([]Plan, error) {
	plans := []Plan{}

	if err := db.Find(&plans).Error; err != nil {
		log.Error(err)
		return plans, err
	}

	return plans, err
}

// GetPlanByName returns the plan, if it exists, specified by the given name
func GetPlanByName(name string) (Plan, error) {
	plan := Plan{}
	err := db.Where("name=?", name).First(&plan).Error

	if err != nil {
		log.Error(err)
	}

	return plan, err
}
