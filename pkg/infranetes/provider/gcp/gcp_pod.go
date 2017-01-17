package gcp

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/golang/glog"

	gcpvm "github.com/apcera/libretto/virtualmachine/gcp"
	kubeapi "k8s.io/kubernetes/pkg/kubelet/api/v1alpha1/runtime"

	"github.com/sjpotter/infranetes/cmd/infranetes/flags"
	"github.com/sjpotter/infranetes/pkg/infranetes/provider"
	"github.com/sjpotter/infranetes/pkg/infranetes/provider/common"
	"github.com/sjpotter/infranetes/pkg/utils"
)

func init() {
	provider.PodProviders.RegisterProvider("gcp", NewGCPPodProvider)
}

type gcpPodProvider struct {
	config *gceConfig
	ipList *utils.Deque
}

type gcePodData struct{}

type gceConfig struct {
	Zone        string
	SourceImage string
	Project     string
	Scope       string
	AuthFile    string
	Subnet      string
}

func NewGCPPodProvider() (provider.PodProvider, error) {
	var conf gceConfig

	file, err := ioutil.ReadFile("gce.json")
	if err != nil {
		return nil, fmt.Errorf("File error: %v\n", err)
	}

	json.Unmarshal(file, &conf)

	if conf.SourceImage == "" || conf.Zone == "" || conf.Project == "" || conf.Scope == "" || conf.AuthFile == "" || conf.Subnet == "" {
		msg := fmt.Sprintf("Failed to read in complete config file: conf = %+v", conf)
		glog.Info(msg)
		return nil, fmt.Errorf(msg)
	}

	// FIXME: add autodetection like AWS
	if *flags.MasterIP == "" || *flags.IPBase == "" {
		return nil, fmt.Errorf("GCP doesn't have autodetection yet: MasterIP = %v, IPBase = %V", *flags.MasterIP, *flags.IPBase)
	}

	ipList := utils.NewDeque()
	for i := 1; i <= 255; i++ {
		ipList.Append(fmt.Sprint(*flags.IPBase + "." + strconv.Itoa(i)))
	}

	return &gcpPodProvider{
		ipList: ipList,
	}, nil
}

func (*gcpPodProvider) UpdatePodState(data *common.PodData) {
	if data.Booted {
		data.UpdatePodState()
	}
}

func (p *gcpPodProvider) bootSandbox(vm *gcpvm.VM, config *kubeapi.PodSandboxConfig, name string) (*common.PodData, error) {
	cAnno := common.ParseCommonAnnotations(config.Annotations)

	if err := vm.Provision(); err != nil {
		return nil, fmt.Errorf("failed to provision vm: %v\n", err)
	}

	ips, err := vm.GetIPs()
	if err != nil {
		return nil, fmt.Errorf("CreatePodSandbox: error in GetIPs(): %v", err)
	}

	glog.Infof("CreatePodSandbox: ips = %v", ips)

	// FIXME: Perhaps better way to choose public vs private ip
	index := 1
	podIp := ips[index].String()

	client, err := common.CreateRealClient(podIp)
	if err != nil {
		return nil, fmt.Errorf("CreatePodSandbox: error in createClient(): %v", err)
	}

	err = client.SetSandboxConfig(config)
	if err != nil {
		glog.Warningf("CreatePodSandbox: Failed to save sandbox config: %v", err)
	}

	err = client.SetPodIP(podIp)
	if err != nil {
		glog.Warningf("CreatePodSandbox: Failed to configure inteface: %v", err)
	}

	if cAnno.StartProxy {
		err = client.StartProxy()
		if err != nil {
			client.Close()
			glog.Warningf("CreatePodSandbox: Couldn't start kube-proxy: %v", err)
		}
	}

	if cAnno.SetHostname {
		err = client.SetHostname(config.GetHostname())
		if err != nil {
			glog.Warningf("CreatePodSandbox: couldn't set hostname to %v: %v", config.GetHostname(), err)
		}
	}

	booted := true

	locaData := &gcePodData{}

	podData := common.NewPodData(vm, &name, config.Metadata, config.Annotations, config.Labels, podIp, config.Linux, client, booted, &locaData)

	return podData, nil
}

// FIXME: add image support
func (v *gcpPodProvider) RunPodSandbox(req *kubeapi.RunPodSandboxRequest) (*common.PodData, error) {
	name := "infranetes-" + req.GetConfig().GetMetadata().GetUid()
	disk := []gcpvm.Disk{{DiskType: "pd-standard", DiskSizeGb: 10, AutoDelete: true}}

	vm := &gcpvm.VM{
		//Scopes:        []string{"https://www.googleapis.com/auth/cloud-platform"},
		//AccountFile: "/root/gcp.json",
		Name:          name,
		Zone:          v.config.Zone,
		MachineType:   "g1-small",
		SourceImage:   v.config.SourceImage,
		Disks:         disk,
		Preemptible:   false,
		Network:       "default",
		Subnetwork:    v.config.Subnet,
		UseInternalIP: false,
		ImageProjects: []string{"engineering-lab"},
		Project:       "engineering-lab",
		Scopes:        []string{v.config.Scope},
		AccountFile:   v.config.AuthFile,
		Tags:          []string{"infranetes:true"},
	}

	podIp := v.ipList.Shift().(string)

	ret, err := v.bootSandbox(vm, req.Config, podIp)
	if err == nil {
		// FIXME: Google's version of elastic IP handling goes here
	}

	return ret, err
}

func (v *gcpPodProvider) PreCreateContainer(data *common.PodData, req *kubeapi.CreateContainerRequest, imageStatus func(req *kubeapi.ImageStatusRequest) (*kubeapi.ImageStatusResponse, error)) error {
	//FIXME: will when image support is added
	return nil
}

func (v *gcpPodProvider) StopPodSandbox(podData *common.PodData) {}

func (v *gcpPodProvider) RemovePodSandbox(data *common.PodData) {
	glog.Infof("RemovePodSandbox: release IP: %v", data.Ip)

	v.ipList.Append(data.Ip)
}

func (v *gcpPodProvider) PodSandboxStatus(podData *common.PodData) {}

func (v *gcpPodProvider) ListInstances() ([]*common.PodData, error) {
	//FIXME: Implement - Needs tagging
	return nil, nil
}