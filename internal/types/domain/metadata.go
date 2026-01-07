package domain

import (
	"fmt"
	"os"
	"os/user"
	"runtime"

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
		ID:   id.MyID,
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
