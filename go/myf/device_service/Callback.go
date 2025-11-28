package device_service

import (
	"fmt"

	"github.com/saichler/l8myfamiliy/go/types/l8myfamily"
	"github.com/saichler/l8types/go/ifs"
)

type DeviceCallback struct{}

func (lc *DeviceCallback) Before(elem interface{}, action ifs.Action, notify bool, vnic ifs.IVNic) (interface{}, error) {
	if action == ifs.POST {
		device := elem.(*l8myfamily.Device)
		fmt.Println("[Device] ", device.Id, "-", device.FamilyId, "-", device.Name)
	}
	return nil, nil
}

func (lc *DeviceCallback) After(elem interface{}, action ifs.Action, notify bool, vnic ifs.IVNic) (interface{}, error) {
	return nil, nil
}
