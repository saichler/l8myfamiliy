package device_service

import (
	"fmt"
	"os"

	"github.com/saichler/l8myfamiliy/go/types/l8myfamily"
	"github.com/saichler/l8utils/go/utils/strings"
	"google.golang.org/protobuf/proto"
)

const (
	location = "/data/my-family/devices/"
)

type DeviceStorage struct{}

func newDeviceStorage() *DeviceStorage {
	os.MkdirAll(location, 0777)
	return &DeviceStorage{}
}

func buildFilename(k string) string {
	return strings.New(location, k).String()
}

func (this *DeviceStorage) Put(k string, v interface{}) error {
	device := v.(*l8myfamily.Device)
	d, e := proto.Marshal(device)
	if e != nil {
		return e
	}
	filename := buildFilename(k)
	return os.WriteFile(filename, d, 0777)
}

func (this *DeviceStorage) Get(k string) (interface{}, error) {
	filename := buildFilename(k)
	d, e := os.ReadFile(filename)
	if e != nil {
		return nil, e
	}
	device := &l8myfamily.Device{}
	e = proto.Unmarshal(d, device)
	return device, e
}

func (this *DeviceStorage) Delete(k string) (interface{}, error) {
	filename := buildFilename(k)
	d, e := os.ReadFile(filename)
	if e != nil {
		return nil, e
	}
	device := &l8myfamily.Device{}
	e = proto.Unmarshal(d, device)
	return device, os.Remove(filename)
}

func (this *DeviceStorage) Collect(f func(interface{}) (bool, interface{})) map[string]interface{} {
	result := make(map[string]interface{})
	devices, err := os.ReadDir(location)
	if err != nil {
		return nil
	}
	for _, devFile := range devices {
		vClone, e := this.Get(devFile.Name())
		if e != nil {
			fmt.Println(e.Error())
			continue
		}
		ok, elem := f(vClone)
		if ok {
			result[devFile.Name()] = elem
		}
	}
	return result
}

func (this *DeviceStorage) CacheEnabled() bool {
	return true
}
