package parser

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

type Values struct {
	Namespace         *string
	NodeSelector      *NodeSelector
	AntiAffinity      *AntiAffinity
	Uuid              *string
	ImagePullSecret   *string
	Image             *PodRootInfo
	AwsSideCar        *PodInfo
	Logging           *PodInfo
	Nasc              *PodInfo
	Multus            *Multus
	LmgScale          *Scale
	LlbScale          *Scale
	Resources         *PodResources
	GwConfig          *string
	GwRedundancy      *Redundancy
	Storage           *Storage
	RtScheduling      *RtScheduling
	LoamB             *Enable
	PodSecurityPolicy *PodSecurityPolicy
}

type NodeSelector struct{}

type AntiAffinity struct{}

type PodRootInfo struct {
	Repository *string
	Name       *string
	Tag        *string
	PullPolicy *string
}

type PodInfo struct {
	Enable             *int
	ImageRepository    *string
	ImageName          *string
	ImageTag           *string
	ImagePullPolicy    *string
	ConfigReadInterval *string
	ScrapeInterval     *ScrapeInterval
}

type ScrapeInterval struct {
	loam *KcikpiInfo
	lmg  *KcikpiInfo
}

type KcikpiInfo struct {
	kciInfo []KciKpi
}

type KciKpi struct {
	Name     *string
	Interval *int
}

type Storage struct {
	PvCreation      *int
	ParentPath      *string
	PvLogsName      *string
	PvStorageClass  *string
	PvLogsClaimName *string
	PvSize          *string
	CfSize          *string
	CfAInfo         []map[string]string
	CfBInfo         []map[string]string
}

type RtScheduling struct {
	Enable         *int
	CgroupHostPath *string
}

type PodSecurityPolicy struct {
	Create     *bool
	Privileged *bool
}

type Redundancy struct {
	Active *int
}

type Scale struct {
	MinReplicas                    *int
	MaxReplicas                    *int
	TargetCPUUtilizationPercentage *int
}

type PodResources struct {
	Lmg     *Resource
	Llb     *Resource
	Loam    *Resource
	Logging *Resource
	Nasc    *Resource
}

type Resource struct {
	Cpu          *int
	Memory       *string
	Hugepages1Gi *string
	Multus       []*MultusCnfInfo
	NodeSelector *NodeSelector
}

type Enable struct {
	Enable *int
}

// Multus
type Multus struct {
	Lmg       *MultusCnfInfo
	Llb       *MultusCnfInfo
	AttachDef []*AttachDef `yaml:"attachDef"`
	GroFlag   *int
	Dsf       *MultusDataPlaneInfo
	Xdp       *MultusDataPlaneInfo
	Dpdk      *MultusDataPlaneInfo
}

// AttachDef
type AttachDef struct {
	Name         *string
	CniVersion   *string
	ResourceName *string
	Vlan         *int
	Type         *string
	PciBusID     *string
	DeviceID     *string
	Device       *string
}

// MultusCnfInfo
type MultusCnfInfo struct {
	NumDevices   *int
	NetNames     []*string
	ResourceName *string
}

// MultusDataPlaneInfo
type MultusDataPlaneInfo struct {
	Enable        *int
	NumDsfDevices *int
}

func (p *Parser) ParseCnfKustomize(cnfName *string, appc *AppConfig, appIPMap *AppIPMap) {
	log.Infof("Rendering Application Data into Kustomize K8s manifests for %s...", *cnfName)
	dirName := filepath.Join(*p.BaseAppKustomizesDir)
	p.CreateDirectory(dirName, 0777)
	p.CreateDirectory(filepath.Join(dirName, *cnfName), 0777)

	// Parse the application templates
	t := ParseTemplates("./templates/app-kustomize")

	switch *cnfName {
	case "upf", "smf":
		values := &Values{
			Namespace:       cnfName,
			NodeSelector:    &NodeSelector{},
			AntiAffinity:    &AntiAffinity{},
			Uuid:            StringPtr("842887ce-329d-4add-9a1c-e7dd03faa00f"),
			ImagePullSecret: appc.ContainerRepo.ImageSecret,
			Image: &PodRootInfo{
				Repository: appc.ContainerRepo.ImageRepo,
				Name:       StringPtr("lmg"),
				Tag:        appc.Containers["llb"].ImageTag,
				PullPolicy: StringPtr("IfNotPresent"),
			},
			Nasc: &PodInfo{
				Enable:          IntPtr(1),
				ImageRepository: appc.ContainerRepo.ImageRepo,
				ImageName:       appc.Containers["nasc"].ImageName,
				ImageTag:        appc.Containers["nasc"].ImageTag,
				ImagePullPolicy: StringPtr("IfNotPresent"),
			},
			AwsSideCar: &PodInfo{
				Enable:          IntPtr(0),
				ImageRepository: appc.ContainerRepo.ImageRepo,
				ImageName:       appc.Containers["nasc"].ImageName,
				ImageTag:        appc.Containers["nasc"].ImageTag,
				ImagePullPolicy: StringPtr("IfNotPresent"),
			},
			Logging: &PodInfo{
				Enable:          IntPtr(1),
				ImageRepository: appc.ContainerRepo.ImageRepo,
				ImageName:       appc.Containers["nasc"].ImageName,
				ImageTag:        appc.Containers["nasc"].ImageTag,
				ImagePullPolicy: StringPtr("IfNotPresent"),
			},
			Multus: &Multus{
				Lmg: &MultusCnfInfo{
					NumDevices: IntPtr(0),
				},
				Llb: &MultusCnfInfo{
					NumDevices: IntPtr(0),
				},
				GroFlag: IntPtr(1),
				Dsf: &MultusDataPlaneInfo{
					Enable:        IntPtr(0),
					NumDsfDevices: IntPtr(0),
				},
				Xdp: &MultusDataPlaneInfo{
					Enable: IntPtr(0),
				},
				Dpdk: &MultusDataPlaneInfo{
					Enable: IntPtr(0),
				},
			},
			GwConfig: cnfName,
			GwRedundancy: &Redundancy{
				Active: IntPtr(2),
			},
			Storage: &Storage{
				PvCreation:      IntPtr(1),
				ParentPath:      StringPtr("/mnt/glusterfs/"),
				PvLogsName:      StringPtr("logs-volume-smf"),
				PvStorageClass:  StringPtr("manual"),
				PvLogsClaimName: StringPtr("logs-volume-claim"),
				PvSize:          StringPtr("1Gi"),
				CfSize:          StringPtr("1Gi"),
				CfAInfo:         make([]map[string]string, 0),
				CfBInfo:         make([]map[string]string, 0),
			},
			RtScheduling: &RtScheduling{
				Enable:         IntPtr(0),
				CgroupHostPath: StringPtr("/sys/fs/cgroup/cpu,cpuacct/"),
			},
			LoamB: &Enable{
				Enable: IntPtr(1),
			},
			LmgScale: &Scale{
				MinReplicas:                    IntPtr(2),
				MaxReplicas:                    IntPtr(2),
				TargetCPUUtilizationPercentage: IntPtr(90),
			},
			LlbScale: &Scale{
				MinReplicas:                    IntPtr(2),
				MaxReplicas:                    IntPtr(2),
				TargetCPUUtilizationPercentage: IntPtr(90),
			},
			Resources: &PodResources{
				Lmg: &Resource{
					Cpu:          appc.Containers["lmg"].CPU,
					Memory:       appc.Containers["lmg"].Memory,
					Hugepages1Gi: appc.Containers["lmg"].Hugepages1Gi,
					NodeSelector: &NodeSelector{},
				},
				Llb: &Resource{
					Cpu:          appc.Containers["llb"].CPU,
					Memory:       appc.Containers["llb"].Memory,
					Hugepages1Gi: appc.Containers["llb"].Hugepages1Gi,
					NodeSelector: &NodeSelector{},
				},
				Loam: &Resource{
					Cpu:    appc.Containers["loam"].CPU,
					Memory: appc.Containers["loam"].Memory,
				},
				Logging: &Resource{
					Cpu:    appc.Containers["logging"].CPU,
					Memory: appc.Containers["logging"].Memory,
				},
				Nasc: &Resource{
					Cpu:    appc.Containers["nasc"].CPU,
					Memory: appc.Containers["nasc"].Memory,
				},
			},
			PodSecurityPolicy: &PodSecurityPolicy{
				Create:     BoolPtr(false),
				Privileged: BoolPtr(false),
			},
		}

		for n := 0; n <= 1; n++ {
			for i := 1; i <= 2; i++ {
				switch n {
				case 1:
					// case a
					p := make(map[string]string)
					p["PvName"] = fmt.Sprintf("%s-cf%d-a-volume", *cnfName, i)
					p["PvcName"] = fmt.Sprintf("cf%d-a-volume-claim", i)
					values.Storage.CfAInfo = append(values.Storage.CfAInfo, p)
				case 2:
					// case b
					p := make(map[string]string)
					p["PvName"] = fmt.Sprintf("%s-cf%d-b-volume", *cnfName, i)
					p["PvcName"] = fmt.Sprintf("cf%d-b-volume-claim", i)
					values.Storage.CfBInfo = append(values.Storage.CfBInfo, p)
				}

			}
		}
		for _, workloads := range appc.Networks {
			for _, switchGroups := range workloads[0] {
				if networks, ok := switchGroups["itfce"]; ok {
					if networkTypes, ok := networks["intIP"]; ok {
						networkInfo := networkTypes[0]
						var attachDef *AttachDef
						var netName *string
						if *appc.ConnectivityMode == "multiNet" {
							netName = StringPtr(fmt.Sprintf("numa%d-%s-%s-%s-%d", *networkInfo.Numa, *networkInfo.InterfaceName, *networkInfo.Target, *networkInfo.NetworkShortName, *networkInfo.VlanID))
							attachDef = &AttachDef{
								Name:         netName,
								CniVersion:   StringPtr("0.3.1"),
								ResourceName: StringPtr(fmt.Sprintf("gke/sriov_numa%d-%s-%s", *networkInfo.Numa, *networkInfo.InterfaceName, *networkInfo.Target)),
								Vlan:         networkInfo.VlanID,
							}
						} else {
							netName = StringPtr(fmt.Sprintf("numa%d-%s-%s-%s", *networkInfo.Numa, *networkInfo.InterfaceName, *networkInfo.Target, *networkInfo.NetworkShortName))
							attachDef = &AttachDef{
								Name:         netName,
								CniVersion:   StringPtr("0.3.1"),
								ResourceName: StringPtr(fmt.Sprintf("gke/sriov_numa%d-%s-%s", *networkInfo.Numa, *networkInfo.InterfaceName, *networkInfo.Target)),
							}
						}
						if *appc.Containers["llb"].Enabled {
							*values.Multus.Llb.NumDevices++
							values.Multus.Llb.NetNames = append(values.Multus.Llb.NetNames, netName)
						}
						if *appc.Containers["lmg"].Enabled {
							*values.Multus.Lmg.NumDevices++
							values.Multus.Lmg.NetNames = append(values.Multus.Lmg.NetNames, netName)
						}
						values.Multus.AttachDef = append(values.Multus.AttachDef, attachDef)

					}
				}
			}
		}

		// render network attachement definition
		p.RenderNetworkAttachement(t, StringPtr(filepath.Join(dirName, *cnfName)), values)

		// render network attachement definition
		p.RenderToActiveConfigMap(t, StringPtr(filepath.Join(dirName, *cnfName)), values)

		// render network attachement definition
		p.RenderStatefulSet(t, StringPtr(filepath.Join(dirName, *cnfName)), values)

	case "amf":
	}

}
