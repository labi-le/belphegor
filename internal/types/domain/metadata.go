package domain

import (
	"fmt"
	"os"
	"os/user"
	"runtime"

	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/id"
)

type Device struct {
	ID   id.Unique
	Name string
	Arch string
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
		ID:   id.New(),
	}
}

func SelfMetaData() Device {
	return defaultMetadata
}

func (meta Device) UniqueID() id.Unique {
	return meta.ID
}

func (meta Device) String() string {
	return meta.Name
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
