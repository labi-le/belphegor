package data

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/rs/zerolog/log"
	"os"
	"os/user"
	"runtime"
)

type UniqueID = uuid.UUID

type MetaData struct {
	proto    *types.Device
	cachedID UniqueID
}

func (meta *MetaData) UniqueID() UniqueID {
	if meta.cachedID == uuid.Nil {
		meta.cachedID = uuid.MustParse(meta.proto.UniqueID)
	}
	return meta.cachedID
}

func SelfMetaData() *MetaData {
	return &MetaData{proto: &types.Device{
		Name:     DeviceName(),
		Arch:     runtime.GOARCH,
		UniqueID: uuid.New().String(),
	}}
}

func (meta *MetaData) Kind() *types.Device {
	return meta.proto
}

func MetaDataFromKind(device *types.Device) *MetaData {
	return &MetaData{
		proto: device,
	}
}

func (meta *MetaData) String() string {
	return fmt.Sprintf(
		"%s (%s)",
		meta.proto.Name,
		meta.proto.UniqueID,
	)
}

func (meta *MetaData) Name() string {
	return meta.proto.Name
}

func DeviceName() string {
	hostname, hostErr := os.Hostname()
	if hostErr != nil {
		log.Error().AnErr("deviceName:hostname", hostErr)
		return "unknown@unknown"
	}

	current, userErr := user.Current()
	if userErr != nil {
		log.Error().AnErr("deviceName:username", userErr)

		return "unknown@unknown"
	}

	return fmt.Sprintf("%s@%s", current.Username, hostname)
}
