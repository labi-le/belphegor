package domain

import (
	"fmt"
	"github.com/labi-le/belphegor/internal/types/proto"
	"os"
	"os/user"
	"runtime"
)

type MetaData struct {
	Name string
	Arch string
	ID   UniqueID
}

var defaultMetadata = initDefaultMetadata()

func initDefaultMetadata() MetaData {
	hostname, _ := os.Hostname()
	usr, _ := user.Current()
	name := "unknown@unknown"
	if hostname != "" && usr != nil {
		name = fmt.Sprintf("%s@%s", usr.Username, hostname)
	}

	return MetaData{
		Name: name,
		Arch: runtime.GOARCH,
		ID:   NewID(),
	}
}

func SelfMetaData() MetaData {
	return defaultMetadata
}

func (meta MetaData) UniqueID() UniqueID {
	return meta.ID
}

func (meta MetaData) String() string {
	return fmt.Sprintf("%s (%d)", meta.Name, meta.ID)
}

func MetaDataFromProto(device *proto.Device) MetaData {
	return MetaData{
		Name: device.Name,
		Arch: device.Arch,
		ID:   device.ID,
	}
}

func (meta MetaData) Proto() *proto.Device {
	return &proto.Device{
		Name: meta.Name,
		Arch: meta.Arch,
		ID:   meta.ID,
	}
}
