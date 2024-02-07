package modem

import (
	"fmt"
	"os"
)

var registration = map[string]map[string]RegisterCallback{}

var EnvURL = os.Getenv("MODEM_URL")
var EnvUser = os.Getenv("MODEM_USER")
var EnvPassword = os.Getenv("MODEM_PASSWORD")

type RegisterCallback func() Modem

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
	return cb(), nil
}
