package database

import (
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Setting struct {
	SettingKey   string `json:"settingKey" gorm:"primaryKey;" extensions:"!x-nullable"`
	SettingValue string `json:"settingValue" gorm:"not null;" extensions:"!x-nullable"`
	SettingType  string `json:"-" gorm:"not null;" extensions:"!x-nullable"`
}

const (
	ReqInterval = "req_interval"
)

func InitSettings() error {
	if err := DB.FirstOrCreate(
		&Setting{SettingKey: ReqInterval, SettingValue: "15", SettingType: "int"}).Error; err != nil {
		return err
	}

	return nil
}

func GetValue(settingKey string) (interface{}, error) {
	sett := Setting{}

	if err := DB.Table("settings").First(&sett, &Setting{SettingKey: settingKey}).Error; err != nil {
		log.Errorf("[GetValue] Error retreiving setting: %s", err)
		return nil, err
	}

	switch sett.SettingType {
	case "int":
		i, err := strconv.Atoi(sett.SettingValue)
		return i, err
	case "string":
		return sett.SettingValue, nil
	case "bool":
		return sett.SettingValue == "true", nil
	}

	return nil, fmt.Errorf("unknown settings type '%s'", sett.SettingType)
}

func (setting *Setting) Save() error {
	// Use SaveOrCreate to either update existing setting or create new one
	result := DB.Save(setting)
	if result.Error != nil {
		log.Errorf("[Save] Error saving setting: %s", result.Error)
		return result.Error
	}

	// Check if a new record was created
	if result.RowsAffected > 0 {
		log.Infof("[Save] Setting saved: %s = %s", setting.SettingKey, setting.SettingValue)
	}

	return nil
}
