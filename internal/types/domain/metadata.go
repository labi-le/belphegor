package domain

import (
	"fmt"
	"os"
	"os/user"
	"runtime"

	"github.com/labi-le/belphegor/internal/types/proto"
)

type Device struct {
	Name string
	Arch string
	ID   UniqueID
}

var defaultMetadata = initDefaultMetadata()

func initDefaultMetadata() Device {
	hostname, _ := os.Hostname()
	usr, _ := user.Current()
	name := "unknown@unknown"
	if hostname != "" && usr != nil {
		name = fmt.Sprintf("%s@%s", usr.Username, hostname)
	}

	return Device{
		Name: name,
		Arch: runtime.GOARCH,
		ID:   NewID(),
	}
}

func SelfMetaData() Device {
	return defaultMetadata
}

func (meta Device) UniqueID() UniqueID {
	return meta.ID
}

func (meta Device) String() string {
	return fmt.Sprintf("%s (%d)", meta.Name, meta.ID)
}

func MetaDataFromProto(device *proto.Device) Device {
	return Device{
		Name: device.Name,
		Arch: device.Arch,
		ID:   device.ID,
	}
}

func (meta Device) Proto() *proto.Device {
	return &proto.Device{
		Name: meta.Name,
		Arch: meta.Arch,
		ID:   meta.ID,
	}
}
