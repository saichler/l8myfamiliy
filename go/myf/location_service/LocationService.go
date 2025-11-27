package location_service

import (
	"github.com/saichler/l8myfamiliy/go/myf/device_service"
	"github.com/saichler/l8myfamiliy/go/types/l8myfamily"
	"github.com/saichler/l8services/go/services/base"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8services"
	"github.com/saichler/l8types/go/types/l8web"
	"github.com/saichler/l8utils/go/utils/web"
)

const (
	ServiceName = "Location"
	ServiceArea = byte(53)
)

func Activate(vnic ifs.IVNic) {
	serviceConfig := ifs.NewServiceLevelAgreement(&base.BaseService{}, ServiceName, ServiceArea, true, &LocationCallback{})

	services := &l8services.L8Services{}
	services.ServiceToAreas = make(map[string]*l8services.L8ServiceAreas)
	services.ServiceToAreas[ServiceName] = &l8services.L8ServiceAreas{}
	services.ServiceToAreas[ServiceName].Areas = make(map[int32]bool)
	services.ServiceToAreas[ServiceName].Areas[int32(ServiceArea)] = true

	serviceConfig.SetServiceItem(&l8myfamily.Location{})

	serviceConfig.SetVoter(true)
	serviceConfig.SetTransactional(false)
	serviceConfig.SetPrimaryKeys("DeviceId")
	serviceConfig.SetWebService(web.New(ServiceName, ServiceArea,
		&l8myfamily.Location{}, &l8web.L8Empty{},
		nil, nil,
		nil, nil,
		nil, nil,
		nil, nil))
	base.Activate(serviceConfig, vnic)
}

type LocationCallback struct{}

func (lc *LocationCallback) BeforePost(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *LocationCallback) AfterPost(elem interface{}, vnic ifs.IVNic) interface{} {
	l := elem.(*l8myfamily.Location)
	device_service.UpdateDevice(l.DeviceId, l.Longitude, l.Latitude, vnic)
	return nil
}

func (lc *LocationCallback) BeforePut(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *LocationCallback) AfterPut(elem interface{}, vnic ifs.IVNic) interface{} {
	l := elem.(*l8myfamily.Location)
	device_service.UpdateDevice(l.DeviceId, l.Longitude, l.Latitude, vnic)
	return nil
}

func (lc *LocationCallback) BeforePatch(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *LocationCallback) AfterPatch(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *LocationCallback) BeforeDelete(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *LocationCallback) AfterDelete(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *LocationCallback) BeforeGet(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}

func (lc *LocationCallback) AfterGet(elem interface{}, vnic ifs.IVNic) interface{} {
	return nil
}
