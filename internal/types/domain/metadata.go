package domain

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/labi-le/belphegor/internal/types/proto"
	"os"
	"os/user"
	"runtime"
)

type UniqueID = uuid.UUID

type MetaData struct {
	Name     string
	Arch     string
	uniqueID UniqueID
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
		Name:     name,
		Arch:     runtime.GOARCH,
		uniqueID: uuid.New(),
	}
}

func SelfMetaData() MetaData {
	return defaultMetadata
}

func (meta MetaData) UniqueID() UniqueID {
	return meta.uniqueID
}

func (meta MetaData) String() string {
	return fmt.Sprintf("%s (%s)", meta.Name, meta.uniqueID)
}

func MetaDataFromProto(device *proto.Device) MetaData {
	return MetaData{
		Name:     device.Name,
		Arch:     device.Arch,
		uniqueID: uuid.MustParse(device.UniqueID),
	}
}

func (meta MetaData) Proto() *proto.Device {
	return &proto.Device{
		Name:     meta.Name,
		Arch:     meta.Arch,
		UniqueID: meta.uniqueID.String(),
	}
}
