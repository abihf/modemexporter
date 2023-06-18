package modem

import (
	"fmt"
	"os"
)

var registration = map[string]map[string]RegisterCallback{}

type RegisterCallback func(url, user, pass string) Modem

func Register(info *Info, cb RegisterCallback) {
	modelMap := registration[info.Vendor]
	if modelMap == nil {
		modelMap = map[string]RegisterCallback{}
		registration[info.Vendor] = modelMap
	}
	modelMap[info.Model] = cb
}

func FromEnv() (Modem, error) {
	vendor := os.Getenv("MODEM_VENDOR")
	model := os.Getenv("MODEM_MODEL")
	modelMap := registration[vendor]
	if modelMap == nil {
		return nil, fmt.Errorf("Modem vendor %s not found", vendor)
	}
	cb := modelMap[model]
	if cb == nil {
		return nil, fmt.Errorf("Modem vendor %s model %s not found", vendor, model)
	}
	return cb(os.Getenv("MODEM_URL"), os.Getenv("MODEM_USER"), os.Getenv("MODEM_PASSWORD")), nil
}
