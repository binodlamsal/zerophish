package models

import (
	"net/http"
	"strconv"

	log "github.com/binodlamsal/zerophish/logger"
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

// DeleteAvatar deletes given avatar
func DeleteAvatar(a *Avatar) error {
	GetCache().DeleteUserAvatar(a)
	err = db.Delete(a).Error

	if err != nil {
		log.Error(err)
	}

	return err
}

// Serve writes proper headers and content of this avatar image to the given ResponseWriter
func (a *Avatar) Serve(w http.ResponseWriter) {
	dataURL, err := dataurl.DecodeString(a.Data)

	if err != nil {
		log.Error(err)
		return
	}

	w.Header().Set("Content-Type", dataURL.MediaType.ContentType())
	w.Header().Set("Content-Length", strconv.Itoa(len(dataURL.Data)))
	w.Write(dataURL.Data)
}

func (a *Avatar) BeforeSave() (err error) {
	GetCache().DeleteEntry("user", a.UserId, "avatar")
	return
}
