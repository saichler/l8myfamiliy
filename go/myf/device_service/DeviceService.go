package device_service

import (
	"fmt"

	"github.com/saichler/l8myfamiliy/go/types/l8myfamily"
	"github.com/saichler/l8services/go/services/base"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8services"
	"github.com/saichler/l8types/go/types/l8web"
	"github.com/saichler/l8utils/go/utils/web"
)

const (
	ServiceName = "Family"
	ServiceArea = byte(53)
)

func Activate(vnic ifs.IVNic) {
	serviceConfig := ifs.NewServiceLevelAgreement(&base.BaseService{}, ServiceName, ServiceArea, true, &DeviceCallback{})

	services := &l8services.L8Services{}
	services.ServiceToAreas = make(map[string]*l8services.L8ServiceAreas)
	services.ServiceToAreas[ServiceName] = &l8services.L8ServiceAreas{}
	services.ServiceToAreas[ServiceName].Areas = make(map[int32]bool)
	services.ServiceToAreas[ServiceName].Areas[int32(ServiceArea)] = true

	serviceConfig.SetServiceItem(&l8myfamily.Device{})
	serviceConfig.SetServiceItemList(l8myfamily.DeviceList{})

	serviceConfig.SetVoter(true)
	serviceConfig.SetTransactional(false)
	serviceConfig.SetPrimaryKeys("Id")
	serviceConfig.SetStore(newDeviceStorage())
	webs := web.New(ServiceName, ServiceArea, 0)
	webs.AddEndpoint(&l8myfamily.Device{}, ifs.POST, &l8web.L8Empty{})
	webs.AddEndpoint(&l8api.L8Query{}, ifs.GET, &l8myfamily.DeviceList{})
	base.Activate(serviceConfig, vnic)
}

func UpdateDevice(id string, lg, lt float32, vnic ifs.IVNic) {
	sv, ok := vnic.Resources().Services().ServiceHandler(ServiceName, ServiceArea)
	if ok {
		device := &l8myfamily.Device{Id: id, Longitude: lg, Latitude: lt}
		exist := sv.Get(object.New(nil, device), vnic)
		if exist != nil && exist.Error() != nil {
			fmt.Println("Error for ", id, ": ", exist.Error())
			return
		}
		if exist == nil || exist.Element() == nil {
			fmt.Println("No Device exist for ", id)
			return
		}
		existDevice := exist.Element().(*l8myfamily.Device)
		sv.Patch(object.New(nil, device), vnic)
		fmt.Println("Device ", id, "-", existDevice.FamilyId, "-", existDevice.Name, " updated")
	}
}
