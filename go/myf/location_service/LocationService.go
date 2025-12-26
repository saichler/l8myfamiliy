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
	webs := web.New(ServiceName, ServiceArea, 0)
	webs.AddEndpoint(&l8myfamily.Location{}, ifs.POST, &l8web.L8Empty{})
	base.Activate(serviceConfig, vnic)
}

type LocationCallback struct{}

func (lc *LocationCallback) Before(elem interface{}, action ifs.Action, notify bool, vnic ifs.IVNic) (interface{}, bool, error) {
	return nil, true, nil
}

func (lc *LocationCallback) After(elem interface{}, action ifs.Action, notify bool, vnic ifs.IVNic) (interface{}, bool, error) {
	if action == ifs.POST || action == ifs.PUT {
		l := elem.(*l8myfamily.Location)
		device_service.UpdateDevice(l.DeviceId, l.Longitude, l.Latitude, vnic)
	}
	return nil, true, nil
}
