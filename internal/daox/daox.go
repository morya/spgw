package daox

import (
	"errors"

	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/models"
)

func FindSpInfoBySpid(spid string) (*models.SpInfo, error) {
	if spid == "" {
		return nil, errors.New("spid is empty")
	}
	m := &models.SpInfo{}
	err := db.MainDB.Where("spid = ?", spid).First(m).Error
	if err != nil {
		return nil, err
	}
	return m, nil
}
