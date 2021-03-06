package virtualbox

import (
	"github.com/Sirupsen/logrus"
	unikos "github.com/emc-advanced-dev/unik/pkg/os"
	"github.com/emc-advanced-dev/unik/pkg/providers/virtualbox/virtualboxclient"
	"github.com/emc-advanced-dev/pkg/errors"
	"os"
	unikutil "github.com/emc-advanced-dev/unik/pkg/util"
	"github.com/emc-advanced-dev/unik/pkg/config"
	"github.com/emc-advanced-dev/unik/pkg/types"
	"github.com/emc-advanced-dev/unik/pkg/providers/common"
	"time"
	"io/ioutil"
	"github.com/emc-advanced-dev/unik/pkg/compilers/rump"
)

var timeout = time.Second * 10

func (p *VirtualboxProvider) DeployInstanceListener(config config.Virtualbox) error {
	logrus.Infof("checking if instance listener is alive...")
	if instanceListenerIp, err := common.GetInstanceListenerIp(instanceListenerPrefix, timeout); err == nil {
		logrus.Infof("instance listener is alive with IP %s", instanceListenerIp)
		return nil
	}
	logrus.Infof("cannot contact instance listener... cleaning up previous if it exists..")
	virtualboxclient.PowerOffVm(VboxUnikInstanceListener)
	virtualboxclient.DestroyVm(VboxUnikInstanceListener)
	logrus.Infof("compiling new instance listener")
	sourceDir, err := ioutil.TempDir(unikutil.UnikTmpDir(), "")
	if err != nil {
		return errors.New("creating temp dir for instance listener source", err)
	}
	defer os.RemoveAll(sourceDir)
	rawImage, err := common.CompileInstanceListener(sourceDir, instanceListenerPrefix, "projectunik/compilers-rump-go-hw-no-wrapper", rump.CreateImageVirtualBox)
	if err != nil {
		return errors.New("compiling instance listener source to unikernel", err)
	}
	logrus.Infof("staging new instance listener image")
	os.RemoveAll(getImagePath(VboxUnikInstanceListener))
	params := types.StageImageParams{
		Name: VboxUnikInstanceListener,
		RawImage: rawImage,
		Force: true,
	}
	image, err := p.Stage(params)
	if err != nil {
		return errors.New("building bootable vsphere image for instsance listener", err)
	}
	defer func() {
		if err != nil {
			p.DeleteImage(image.Id, true)
		}
	}()

	if err := p.runInstanceListener(image); err != nil {
		return errors.New("launching instance of instance listener", err)
	}
	return nil
}

func (p *VirtualboxProvider) runInstanceListener(image *types.Image) (err error) {
	logrus.WithFields(logrus.Fields{
		"image-id": image.Id,
	}).Infof("running instance of instance listener")

	imagePath, err := unikos.BuildEmptyDataVolume(10)
	if err != nil {
		return errors.New("failed creating raw data volume", err)
	}
	defer os.Remove(imagePath)

	instanceListenerData := "InstanceListenerData"
	createVolumeParams := types.CreateVolumeParams{
		Name: instanceListenerData,
		ImagePath: imagePath,
	}
	instanceListenerVol, err := p.CreateVolume(createVolumeParams)
	if err != nil {
		return errors.New("creating data vol for instance listener", err)
	}

	instanceDir := getInstanceDir(VboxUnikInstanceListener)
	defer func() {
		if err != nil {
			logrus.WithError(err).Warnf("error encountered, ensuring vm and disks are destroyed")
			virtualboxclient.PowerOffVm(VboxUnikInstanceListener)
			virtualboxclient.DestroyVm(VboxUnikInstanceListener)
			os.RemoveAll(instanceDir)
			p.DeleteVolume(instanceListenerVol.Id, true)
			os.RemoveAll(getVolumePath(instanceListenerData))
		}
	}()

	logrus.Debugf("creating vsphere vm")

	if err := virtualboxclient.CreateVm(VboxUnikInstanceListener, virtualboxInstancesDirectory, image.RunSpec.DefaultInstanceMemory, p.config.AdapterName, p.config.VirtualboxAdapterType, image.RunSpec.StorageDriver); err != nil {
		return errors.New("creating vm", err)
	}

	logrus.Debugf("copying base boot vmdk to instance dir")
	logrus.Debugf("copying source boot vmdk")
	instanceBootImage := instanceDir + "/boot.vmdk"
	if err := unikos.CopyFile(getImagePath(image.Name), instanceBootImage); err != nil {
		return errors.New("copying base boot image", err)
	}
	if err := virtualboxclient.RefreshDiskUUID(instanceBootImage); err != nil {
		return errors.New("refreshing disk uuid", err)
	}
	if err := virtualboxclient.AttachDisk(VboxUnikInstanceListener, instanceBootImage, 0, image.RunSpec.StorageDriver); err != nil {
		return errors.New("attaching boot vol to instance", err)
	}

	controllerPort, err := common.GetControllerPortForMnt(image, "/data")
	if err != nil {
		return errors.New("getting controller port for mnt point", err)
	}
	if err := virtualboxclient.AttachDisk(VboxUnikInstanceListener, getVolumePath(instanceListenerVol.Name), controllerPort, image.RunSpec.StorageDriver); err != nil {
		return errors.New("attaching to vm", err)
	}
	if err := p.state.ModifyVolumes(func(volumes map[string]*types.Volume) error {
		volume, ok := volumes[instanceListenerVol.Id]
		if !ok {
			return errors.New("no record of "+volume.Id+" in the state", nil)
		}
		volume.Attachment = instanceListenerVol.Id
		return nil
	}); err != nil {
		return errors.New("modifying volumes in state", err)
	}
	if err := p.state.Save(); err != nil {
		return errors.New("saving instance volume map to state", err)
	}

	logrus.Debugf("powering on vm")
	if err := virtualboxclient.PowerOnVm(VboxUnikInstanceListener); err != nil {
		return errors.New("powering on vm", err)
	}

	instanceListenerIp, err := common.GetInstanceListenerIp(instanceListenerPrefix, time.Second * 30)
	if err != nil {
		return errors.New("failed to retrieve instance listener ip. is unik instance listener running?", err)
	}

	vm, err := virtualboxclient.GetVm(VboxUnikInstanceListener)
	if err != nil {
		return errors.New("getting vm info from vsphere", err)
	}

	instanceId := vm.UUID
	instance := &types.Instance{
		Id:             instanceId,
		Name:           VboxUnikInstanceListener,
		State:          types.InstanceState_Pending,
		IpAddress:      instanceListenerIp,
		Infrastructure: types.Infrastructure_VSPHERE,
		ImageId:        image.Id,
		Created:        time.Now(),
	}

	if err := p.state.ModifyInstances(func(instances map[string]*types.Instance) error {
		instances[instance.Id] = instance
		return nil
	}); err != nil {
		return errors.New("modifying instance map in state", err)
	}
	if err := p.state.Save(); err != nil {
		return errors.New("saving instance volume map to state", err)
	}
	logrus.WithField("instance", instance).Infof("instance created successfully")

	return nil
}
