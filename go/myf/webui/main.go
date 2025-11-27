package main

import (
	"time"

	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8bus/go/overlay/protocol"
	"github.com/saichler/l8bus/go/overlay/vnet"
	"github.com/saichler/l8bus/go/overlay/vnic"
	"github.com/saichler/l8myfamiliy/go/myf/device_service"
	"github.com/saichler/l8myfamiliy/go/myf/location_service"
	"github.com/saichler/l8myfamiliy/go/types/l8myfamily"
	"github.com/saichler/l8reflect/go/reflect/introspecting"
	"github.com/saichler/l8services/go/services/manager"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8health"
	"github.com/saichler/l8types/go/types/l8sysconfig"
	"github.com/saichler/l8types/go/types/l8web"
	"github.com/saichler/l8utils/go/utils/logger"
	"github.com/saichler/l8utils/go/utils/registry"
	"github.com/saichler/l8utils/go/utils/resources"
	"github.com/saichler/l8web/go/web/server"
	"github.com/saichler/probler/go/prob/common"
)

const (
	VNET = 12345
)

func main() {
	resources := CreateResources("vnetfamily")
	resources.Logger().SetLogLevel(ifs.Info_Level)
	net := vnet.NewVNet(resources)
	net.Start()
	resources.Logger().Info("vnet started!")
	startWebServer(9093, "/data/probler")
}

func startWebServer(port int, cert string) {
	serverConfig := &server.RestServerConfig{
		Host:           protocol.MachineIP,
		Port:           port,
		Authentication: false,
		CertName:       cert,
		Prefix:         common.PREFIX,
	}
	svr, err := server.NewRestServer(serverConfig)
	if err != nil {
		panic(err)
	}

	nic := CreateVnic(VNET, "web")

	hs, ok := nic.Resources().Services().ServiceHandler(health.ServiceName, 0)
	if ok {
		ws := hs.WebService()
		svr.RegisterWebService(ws, nic)
	}

	location_service.Activate(nic)
	device_service.Activate(nic)
	time.Sleep(time.Second)

	//Activate the webpoints topo_service
	sla := ifs.NewServiceLevelAgreement(&server.WebService{}, ifs.WebService, 0, false, nil)
	sla.SetArgs(svr)
	nic.Resources().Services().Activate(sla, nic)

	nic.Resources().Logger().Info("Web Server Started!")

	svr.Start()
}

func CreateVnic(vnet uint32, name string) ifs.IVNic {
	resources := CreateResources(name)

	node, _ := resources.Introspector().Inspect(&l8myfamily.Device{})
	introspecting.AddPrimaryKeyDecorator(node, "Id")

	node, _ = resources.Introspector().Inspect(&l8myfamily.Location{})
	introspecting.AddPrimaryKeyDecorator(node, "DeviceId")

	nic := vnic.NewVirtualNetworkInterface(resources, nil)
	nic.Resources().SysConfig().KeepAliveIntervalSeconds = 60
	nic.Start()
	nic.WaitForConnection()

	nic.Resources().Registry().Register(&l8myfamily.Device{})
	nic.Resources().Registry().Register(&l8myfamily.Location{})
	nic.Resources().Registry().Register(&l8myfamily.DeviceList{})
	nic.Resources().Registry().Register(&l8api.L8Query{})
	nic.Resources().Registry().Register(&l8web.L8Empty{})
	nic.Resources().Registry().Register(&l8health.L8Health{})
	nic.Resources().Registry().Register(&l8health.L8HealthList{})

	return nic
}

func CreateResources(alias string) ifs.IResources {
	log := logger.NewLoggerImpl(&logger.FmtLogMethod{})
	log.SetLogLevel(ifs.Error_Level)
	res := resources.NewResources(log)

	res.Set(registry.NewRegistry())

	sec, err := ifs.LoadSecurityProvider(res)
	if err != nil {
		time.Sleep(time.Second * 10)
		panic(err.Error())
	}
	res.Set(sec)

	conf := &l8sysconfig.L8SysConfig{MaxDataSize: resources.DEFAULT_MAX_DATA_SIZE,
		RxQueueSize:              resources.DEFAULT_QUEUE_SIZE,
		TxQueueSize:              resources.DEFAULT_QUEUE_SIZE,
		LocalAlias:               alias,
		VnetPort:                 uint32(VNET),
		KeepAliveIntervalSeconds: 30}
	res.Set(conf)

	res.Set(introspecting.NewIntrospect(res.Registry()))
	res.Set(manager.NewServices(res))

	return res
}
