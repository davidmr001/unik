package types

import (
	"fmt"
	"time"
)

type InstanceState string

const (
	InstanceState_Running    InstanceState = "running"
	InstanceState_Stopped    InstanceState = "stopped"
	InstanceState_Pending    InstanceState = "pending"
	InstanceState_Unknown    InstanceState = "unknown"
	InstanceState_Terminated InstanceState = "terminated"
)

type Infrastructure string

const (
	Infrastructure_AWS        Infrastructure = "AWS"
	Infrastructure_VSPHERE    Infrastructure = "VSPHERE"
	Infrastructure_VIRTUALBOX Infrastructure = "VIRTUALBOX"
)

type Image struct {
	Id             string          `json:"Id"`
	Name           string          `json:"Name"`
	SizeMb         int64           `json:"SizeMb"`
	Infrastructure Infrastructure  `json:"Infrastructure"`
	Created        time.Time       `json:"Created"`
	StageSpec      StageSpec       `json:"StageSpec"`
	RunSpec        RunSpec         `json:"RunSpec"`
}

func (image *Image) Copy() *Image {
	return &Image{
		Id:             image.Id,
		Name:           image.Name,
		SizeMb:         image.SizeMb,
		Infrastructure: image.Infrastructure,
		Created:        image.Created,
		StageSpec:      image.StageSpec,
		RunSpec:        image.RunSpec,
	}
}

func (image *Image) String() string {
	if image == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%-v", *image)
}

type Instance struct {
	Id             string         `json:"Id"`
	Name           string         `json:"Name"`
	State          InstanceState  `json:"State"`
	IpAddress      string         `json:"IpAddress"`
	ImageId        string         `json:"ImageId"`
	Infrastructure Infrastructure `json:"Infrastructure"`
	Created        time.Time      `json:"Created"`
}

func (instance *Instance) Copy() *Instance {
	return &Instance{
		Id:             instance.Id,
		ImageId:        instance.ImageId,
		Infrastructure: instance.Infrastructure,
		Name:           instance.Name,
		State:          instance.State,
		Created:        instance.Created,
	}
}

func (instance *Instance) String() string {
	if instance == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%+v", *instance)
}

type Volume struct {
	Id             string         `json:"Id"`
	Name           string         `json:"Name"`
	SizeMb         int64          `json:"SizeMb"`
	Attachment     string         `json:"Attachment"` //instanceId
	Infrastructure Infrastructure `json:"Infrastructure"`
	Created        time.Time      `json:"Created"`
}

func (volume *Volume) Copy() *Volume{
	return &Volume{
		Id:             volume.Id,
		Name:           volume.Name,
		SizeMb:         volume.SizeMb,
		Attachment:     volume.Attachment,
		Infrastructure: volume.Infrastructure,
		Created:        volume.Created,
	}
}

func (volume *Volume) String() string {
	if volume == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%+v", *volume)
}

type RawImage struct {
	LocalImagePath string          `json:"LocalImagePath"`
	StageSpec      StageSpec       `json:"StageSpec"`
	RunSpec        RunSpec         `json:"RunSpec"`
}

type ImageFormat string
const (
	ImageFormat_RAW ImageFormat = "raw"
	ImageFormat_QCOW2 ImageFormat = "qcow2"
	ImageFormat_VHD ImageFormat = "vhd"
	ImageFormat_VMDK ImageFormat = "vmdk"
)

type XenVirtualizationType string
const (
	XenVirtualizationType_HVM = "hvm"
	XenVirtualizationType_Paravirtual = "paravirtual"
)

type StageSpec struct {
	ImageFormat ImageFormat `json:"ImageFormat"` //required for all compilers
	XenVirtualizationType XenVirtualizationType `json:"XenVirtualizationType,omitempty"`
}

type StorageDriver string
const (
	StorageDriver_SCSI = "SCSI"
	StorageDriver_SATA = "SATA"
	StorageDriver_IDE = "IDE"
)

type VsphereNetworkType string
const (
	VsphereNetworkType_E1000 = "e1000"
	VsphereNetworkType_VMXNET3 = "vmxnet3"
)

type RunSpec struct {
	DeviceMappings []DeviceMapping `json:"DeviceMappings"` //required for all compilers
	DefaultInstanceMemory int `json:"DefaultInstanceMemory"` //required for all compilers
	StorageDriver StorageDriver `json:"StorageDriver,omitempty"`
	VsphereNetworkType VsphereNetworkType `json:"VsphereNetworkType"`
}

type DeviceMapping struct {
	MountPoint string `json:"MountPoint"`
	DeviceName string `json:"DeviceName"`
}