package models

import (
	"net/http"
	"strconv"

	log "github.com/everycloud-technologies/phishing-simulation/logger"
	"github.com/vincent-petithory/dataurl"
)

// Avatar is a custom avatar image
type Avatar struct {
	Id     int64 `gorm:"column:id; primary_key:yes"`
	UserId int64
	Data   string
}

// PutAvatar craetes or updates the given avatar
func PutAvatar(l *Avatar) error {
	return db.Save(l).Error
}

// GetAvatar returns avatar by id
func GetAvatar(id int64) (*Avatar, error) {
	a := Avatar{}

	if err := db.Where("id = ?", id).First(&a).Error; err != nil {
		return &a, err
	}

	return &a, nil
}

// DeleteAvatar deletes avatar specified by the given id
func DeleteAvatar(id int64) error {
	err = db.Delete(&Avatar{Id: id}).Error

	if err != nil {
		log.Error(err)
	}

	return err
}

// Serve writes proper headers and content of this avatar image to the given ResponseWriter
func (l *Avatar) Serve(w http.ResponseWriter) {
	dataURL, err := dataurl.DecodeString(l.Data)

	if err != nil {
		log.Error(err)
		return
	}

	w.Header().Set("Content-Type", dataURL.MediaType.ContentType())
	w.Header().Set("Content-Length", strconv.Itoa(len(dataURL.Data)))
	w.Write(dataURL.Data)
}
