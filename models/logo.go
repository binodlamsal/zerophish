package models

import (
	"fmt"
	"net/http"
	"strconv"

	log "github.com/everycloud-technologies/phishing-simulation/logger"
	"github.com/vincent-petithory/dataurl"
)

// Logo is a custom logo image
type Logo struct {
	Id     int64 `gorm:"column:id; primary_key:yes"`
	UserId int64
	Data   string
}

// PutLogo craetes or updates the given logo
func PutLogo(l *Logo) error {
	return db.Save(l).Error
}

// DeleteLogo deletes logo specified by the given id
func DeleteLogo(id int64) error {
	err = db.Delete(&Logo{Id: id}).Error

	if err != nil {
		log.Error(err)
	}

	return err
}

// DeleteUserLogo deletes logo created by user with the given uid
func DeleteUserLogo(uid int64) error {
	err := db.Where("user_id=?", uid).Delete(&Logo{}).Error

	if err != nil {
		return fmt.Errorf(
			"Couldn't delete logo created by user with id %d - %s",
			uid, err.Error(),
		)
	}

	return nil
}

// Serve writes proper headers and content of this logo image to the given ResponseWriter
func (l *Logo) Serve(w http.ResponseWriter) {
	dataURL, err := dataurl.DecodeString(l.Data)

	if err != nil {
		log.Error(err)
		return
	}

	w.Header().Set("Content-Type", dataURL.MediaType.ContentType())
	w.Header().Set("Content-Length", strconv.Itoa(len(dataURL.Data)))
	w.Write(dataURL.Data)
}
