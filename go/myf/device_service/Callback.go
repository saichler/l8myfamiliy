package device_service

import (
	"fmt"

	"github.com/saichler/l8myfamiliy/go/types/l8myfamily"
	"github.com/saichler/l8types/go/ifs"
)

type DeviceCallback struct{}

func (lc *DeviceCallback) BeforePost(elem interface{}, vnic ifs.IVNic) interface{} {
	device := elem.(*l8myfamily.Device)
	fmt.Println("[Device] ", device.Id, "-", device.FamilyId, "-", device.Name)
	return nil
}

func (lc *DeviceCallback) AfterPost(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *DeviceCallback) BeforePut(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *DeviceCallback) AfterPut(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *DeviceCallback) BeforePatch(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *DeviceCallback) AfterPatch(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *DeviceCallback) BeforeDelete(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *DeviceCallback) AfterDelete(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *DeviceCallback) BeforeGet(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *DeviceCallback) AfterGet(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}
