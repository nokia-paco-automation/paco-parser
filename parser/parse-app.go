package parser

import (
	"fmt"
	"net"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/nokia-paco-automation/paco-parser/types"
	log "github.com/sirupsen/logrus"
)

// DeploymentIPAM is a map where
// first string key = multusNetworkName
// 2nd Key string = ipvlan, sriov1, sriov2
// 3rd key string = IP subnet, could be v4 or v6
//var DeploymentIPAM map[string]map[string]map[string]*IpamApp

type IpamApp struct {
	Gateway      *string
	VlanID       *int
	AllocatedIPs []*AllocatedIPInfo
}

type AllocatedIPInfo struct {
	IPAddress   *string // allocated IP address
	Application *string // allocated application
	Usage       *string // allocated usage
}

// AppIPMap per workload list of IP information that is relevant for the deployment and application
type AppIPMap struct {
	IPinfo map[string]*IPinfo
}

type AppConfig struct {
	ConnectivityMode *string
	Amms             bool
	Dbs              bool
	Emms             bool
	Ipds             bool
	Ipps             bool
	Necc             bool
	Paps             bool
	Aws              bool
	NetworkName      string
	NetworkShortName string
	Mcc              *int
	Mnc              *int
	Supi             [][]*string
	Dnn              []*string
	Slices           map[string]*SliceInfo // key string = slice type; element is slice differentiator
	TrackingArea     []*string
	K8sApiServer     string
	K8sDns           string
	StorageClass     string
	ContainerRepo    *ContainerRepo
	Containers       map[string]*ContainerInfo
	// key1 is wlGenericName, key2 is group, key3 is swithIndex key4 is loopback or interfce, key6 is kind/type
	// group: 0 is used for control plane; group 1, etc for LMG user plane
	Networks          map[string]map[int]map[int]map[string]map[string][]*RenderedNetworkInfo
	SwitchesPerServer *int
	UePoolCidr        *string
	Apn               *string
	// key1: sriov or ipvlan,
	// key2: ServerLogicalInterfacename (bondx),
	// key3: numa,
	// key4: switch-name,
	// value list of pfNames
	UniqueClientServer2NetworkLinks map[string]map[string]map[int]map[string][]*string
	//Switches           []string
	//ClientLinks        map[string]map[string]*ClientLinkInfo
	//UniqueClientLinks  map[string][]*ClientLinkInfo
	WorkloadShortNames map[string]*string
	ConnType           *string
	Llbs               *int
	Lmgs               *int
	K                  *int
}

type ClientLinkInfo struct {
	InterfaceName *string
	Numa          *int
}

type ContainerRepo struct {
	ImageRepo   *string
	ImageSecret *string
}

type ContainerInfo struct {
	ImageName           *string
	ImageTag            *string
	CPU                 *int
	Memory              *string
	Hugepages1Gi        *string
	NodeSelector        *string
	Antiaffinity        []string
	InitialDelaySeconds *int
	PeriodSeconds       *int
	Enabled             *bool
}

type RenderedNetworkInfo struct {
	Ipv4Cidr              *string
	Ipv6Cidr              *string
	Ipv4PrefixLength      *int
	Ipv6PrefixLength      *int
	VlanID                *int
	Target                *string
	NetworkIndex          *int
	SwitchIndex           *int
	Ipv4Addresses         []*AllocatedIPInfo
	Ipv6Addresses         []*AllocatedIPInfo
	Ipv4FloatingIP        *string
	Ipv6FloatingIP        *string
	Ipv4Gw                *string
	Ipv6Gw                *string
	Ipv4GwPerWl           map[int][]string
	Ipv6GwPerWl           map[int][]string
	InterfaceName         *string
	InterfaceEthernetName *string
	Numa                  *int
	Cni                   *string
	NetworkShortName      *string
	ResShortName          *string
	IPv4BGPPeers          map[int]*BGPPeerInfo
	IPv6BGPPeers          map[int]*BGPPeerInfo
	IPv4BGPAddress        *string
	IPv6BGPAddress        *string
	VrfCpId               *int
	VrfUpId               *int
	AS                    *uint32
}

type IPinfo struct {
	N2Ipv4            *string
	N2Ipv6            *string
	N8Ipv4            *string
	N8Ipv6            *string
	N11Ipv4           *string
	N11Ipv6           *string
	N12Ipv4           *string
	N12Ipv6           *string
	N14Ipv4           *string
	N14Ipv6           *string
	N15Ipv4           *string
	N15Ipv6           *string
	N17Ipv4           *string
	N17Ipv6           *string
	N22Ipv4           *string
	N22Ipv6           *string
	N20Ipv4           *string
	N20Ipv6           *string
	N26Ipv4           *string
	N26Ipv6           *string
	NnrfIpv4          *string
	NnrfIpv6          *string
	NsmsIPv4          *string
	NsmsIPv6          *string
	AmfSvcDefaultIPv4 *string
	AmfSvcDefaultIPv6 *string
	AmfSvcLocIPv4     *string
	AmfSvcLocIPv6     *string
	AmfSvcComIPv4     *string
	AmfSvcComIPv6     *string
	AmfSvcEeIPv4      *string
	AmfSvcEeIPv6      *string
	AmfSvcMtIPv4      *string
	AmfSvcMtIPv6      *string
	NfyEirIPv4        *string
	NfyEirIPv6        *string
	NfyAmfIPv4        *string
	NfyAmfIPv6        *string
	NfyAusfIPv4       *string
	NfyAusfIPv6       *string
	NfyNrfIPv4        *string
	NfyNrfIPv6        *string
	NfyNssfIPv4       *string
	NfyNssfIPv6       *string
	NfyPcfIPv4        *string
	NfyPcfIPv6        *string
	NfySmfIPv4        *string
	NfySmfIPv6        *string
	NfyUdmIPv4        *string
	NfyUdmIPv6        *string
	DnsIpdsIPv41      *string
	DnsIpdsIPv61      *string
	DnsIpdsIPv42      *string
	DnsIpdsIPv62      *string
	InternetDNS       *string
	PrimaryDnsIP      *string
	AusfIPv4          *string
	AusfIPv6          *string
	AusfPort          *string
	UdmIPv4           *string
	UdmIPv6           *string
	UdmPort           *string
	SmfIPv4           *string
	SmfIPv6           *string
	SmfPort           *string
	UpfCpIPv4         *string
	UpfCpIPv6         *string
	UpfUpIPv4         *string
	UpfUpIPv6         *string
	UpfPort           *string
	PrometheusIP      *string
}

type switchInfo struct {
	switchesPerServer  *int
	switchBgpPeersIPv4 map[string]map[int]*BGPPeerInfo              // key1 = wlName, key2 = switch id, value is bgp peer info (ip/as)
	switchBgpPeersIPv6 map[string]map[int]*BGPPeerInfo              // key1 = wlName, key2 = switch id, value is bgp peer info (ip/as)
	switchGwsIPv4      map[string]map[string]map[int]map[int]string // key1 = wlName, key2 = networktype, key3 = switch id, key4 is cidr index, value is gw
	switchGwsIPv6      map[string]map[string]map[int]map[int]string // key1 = wlName, key2 = networktype, key3 = switch id, key4 is cidr index, value is gw
	// ipvlan =0
	switchGwsPerWlNameIpv4 map[string]map[int][]string // key1 = wlName, key2 = switch, value is list of GwIPs
	switchGwsPerWlNameIpv6 map[string]map[int][]string // key1 = wlName, key2 = switch, value is list of GwIPs
}

func (p *Parser) ParseApplicationData() *types.AppLbIpResult {

	appLbResult := types.NewAppLbIpResult()

	log.Infof("Rendering Application Data into Helm values.yaml...")
	dirName := filepath.Join(*p.BaseAppValuesDir)
	p.CreateDirectory(dirName, 0777)

	// Parse the application templates
	t := ParseTemplates("./templates/app-helm")

	// get IP allocations from the cluster IP(s)
	var apiServer net.IP
	var dns net.IP
	if svc, ok := p.Config.Cluster.Networks["svc"]; ok {
		for _, ipv4Cidr := range svc.Ipv4Cidr {
			_, ipNet, err := net.ParseCIDR(*ipv4Cidr)
			if err != nil {
				log.WithError(err).Error("Cidr Parsing error")
			}
			apiServer, err = cidr.Host(ipNet, 1)
			if err != nil {
				log.WithError(err).Error("Host Parsing error")
			}
			dns, err = cidr.Host(ipNet, 10)
			if err != nil {
				log.WithError(err).Error("Host Parsing error")
			}
		}
	}

	// get gw and bgp indexes
	gwidx := p.GetApplicationIndex(StringPtr("itfce"), StringPtr("switch"), StringPtr("gw"))
	bgpidx := p.GetApplicationIndex(StringPtr("loopback"), StringPtr("switch"), StringPtr("bgp"))

	for wlName, clients := range p.Config.Workloads {
		p.DeploymentIPAM[wlName] = make(map[string]map[string]*IpamApp)
		// initilaize switchBgpPeers
		// key1 = wlName, key2 = switchid -> value is bgp peer; 1 per switch
		p.SwitchInfo.switchBgpPeersIPv4[wlName] = make(map[int]*BGPPeerInfo)
		p.SwitchInfo.switchBgpPeersIPv6[wlName] = make(map[int]*BGPPeerInfo)
		// initialize switchGws
		// key1 = wlName, key2 = networktype, key3 = switch id, key4 is cidr idx value is gw
		p.SwitchInfo.switchGwsIPv4[wlName] = make(map[string]map[int]map[int]string)
		p.SwitchInfo.switchGwsIPv6[wlName] = make(map[string]map[int]map[int]string)
		// initialize switchGwsPerWlName
		// key1 = wlName, key2 = switch, value is list of GwIPs
		p.SwitchInfo.switchGwsPerWlNameIpv4[wlName] = make(map[int][]string)
		p.SwitchInfo.switchGwsPerWlNameIpv6[wlName] = make(map[int][]string)

		for _, wlInfo := range clients {

			// loopback subnets
			for netwType, netwInfo := range wlInfo.Loopbacks {
				if strings.Contains(netwType, "loopback") {
					for _, ipv4Cidr := range netwInfo.Ipv4Cidr {
						p.AssignSwitchBgpLoopback(StringPtr("ipv4"), ipv4Cidr, StringPtr(wlName), StringPtr(netwType), *bgpidx)
					}

					for _, ipv6Cidr := range netwInfo.Ipv6Cidr {
						p.AssignSwitchBgpLoopback(StringPtr("ipv6"), ipv6Cidr, StringPtr(wlName), StringPtr(netwType), *bgpidx)
					}
				}
			}

			// SRIOV, IPVLAN subnets

			for netwType, netwInfo := range wlInfo.Itfces {
				if strings.Contains(netwType, "ipvlan") || strings.Contains(netwType, "sriov") {

					// initialize sriov or ipvlan
					p.DeploymentIPAM[wlName][netwType] = make(map[string]*IpamApp)

					for idx, ipv4Cidr := range netwInfo.Ipv4Cidr {
						p.AssignSwitchGWs(StringPtr("ipv4"), ipv4Cidr, StringPtr(wlName), StringPtr(netwType), idx, *gwidx, netwInfo)
					}

					for idx, ipv6Cidr := range netwInfo.Ipv6Cidr {
						p.AssignSwitchGWs(StringPtr("ipv6"), ipv6Cidr, StringPtr(wlName), StringPtr(netwType), idx, *gwidx, netwInfo)
					}
				}
			}

			// assign the switches per server
			*p.SwitchInfo.switchesPerServer = len(p.SwitchInfo.switchBgpPeersIPv4[wlName])

		}
	}
	log.Debugf("BGP Peers Ipv4: %v", p.SwitchInfo.switchBgpPeersIPv4)
	log.Debugf("BGP Peers Ipv6: %v", p.SwitchInfo.switchBgpPeersIPv6)
	log.Debugf("Gateways Ipv4: %v", p.SwitchInfo.switchGwsIPv4)
	log.Debugf("Gateways Ipv4: %v", p.SwitchInfo.switchGwsPerWlNameIpv4)
	log.Debugf("Gateways Ipv6: %v", p.SwitchInfo.switchGwsPerWlNameIpv6)

	// get the application information
	for app, pacoInfo := range p.Config.Application {
		if app == "paco" {
			// holds the global IP configuration that applications use to communicate to eachother
			appIPMap := new(AppIPMap)
			appIPMap.IPinfo = make(map[string]*IPinfo)

			appc := make(map[string]*AppConfig)
			// identifies the multus networks on which the apps are connected
			// this holds only the relevant information for the app
			connectedMultusNetworks := make(map[string]*MultusInfo)
			// find all the multus networks on which the cnfs are connected
			for _, cnfInfo := range pacoInfo.Cnfs {
				if *cnfInfo.Enabled {
					for multusGenericWlName, multusInfo := range cnfInfo.Networking.Multus {
						connectedMultusNetworks[multusGenericWlName] = multusInfo

						//initilaize the appIPMap
						appIPMap.IPinfo[multusGenericWlName] = new(IPinfo)
					}
				}
			}
			log.Debugf("Connected multus networks : %v", connectedMultusNetworks)
			// get the cnf related pod/connectivity info
			appIPMap.IPinfo["oam"].InternetDNS = p.Config.Infrastructure.InternetDns
			for cnfName, cnfInfo := range pacoInfo.Cnfs {
				if *cnfInfo.Enabled {
					// holds the relevant information per CNF
					appc[cnfName] = new(AppConfig)
					// provides the connectivity mode for the application
					// Options: multiNet, vlanAwareApp
					appc[cnfName].ConnectivityMode = pacoInfo.Deployment.ConnectivityMode
					// get the K from the Config file; K is K in NtoK deployment model
					appc[cnfName].K = cnfInfo.K

					// initializes the POD/Contaianer information per CNF
					appc[cnfName].InitializeCnfContainerData(p.Config.ContainerRegistry, cnfInfo.Pods)
					//
					//appc[cnfName].InitializeCnfNetworkData(cnfName, cnfInfo, p, appIPMap, p.ClientLinks, p.ClientSriovInfo, p.SwitchInfo)
					appc[cnfName].InitializeCnfNetworkData(cnfName, cnfInfo, p, appIPMap, p.ClientServer2NetworkLinks, p.SwitchInfo, appLbResult)

					appc[cnfName].K8sApiServer = apiServer.String()
					appc[cnfName].K8sDns = dns.String()
					appc[cnfName].NetworkName = *pacoInfo.Deployment.NetworkName
					appc[cnfName].NetworkShortName = *pacoInfo.Deployment.NetworkShortName
					if mcc, ok := pacoInfo.Deployment.Plmn["mcc"]; ok {
						appc[cnfName].Mcc = mcc
					}
					if mnc, ok := pacoInfo.Deployment.Plmn["mnc"]; ok {
						appc[cnfName].Mnc = mnc
					}
					appc[cnfName].Supi = pacoInfo.Deployment.Supi
					appc[cnfName].Dnn = pacoInfo.Deployment.Dnn
					appc[cnfName].TrackingArea = pacoInfo.Deployment.TrackingArea
					appc[cnfName].Slices = pacoInfo.Deployment.Slices
					appc[cnfName].UePoolCidr = pacoInfo.Deployment.UePoolCidr
					appc[cnfName].Apn = pacoInfo.Deployment.Apn
				}

				switch cnfName {
				case "amf":
					if *cnfInfo.Enabled {
						appIPMap.IPinfo["oam"].PrometheusIP = cnfInfo.PrometheusIP
						appc[cnfName].StorageClass = *cnfInfo.StorageClass
					}
				case "smf":
					// nothing special required
				case "upf":
					// nothing special required
				}
			}

			// write the related values.yaml file for the cnfs that are enabled

			for cnfName := range pacoInfo.Cnfs {
				log.Infof("CnfName: %s", cnfName)

				p.WriteCnfValues(t, &dirName,
					StringPtr(cnfName),
					appc[cnfName],
					appIPMap)

				// Show the application IPAM
				//dirName := filepath.Join(*p.BaseAppIpamDir)
				//p.WriteApplicationDeploymentIPAM(&dirName)

				p.ParseCnfKustomize(StringPtr(cnfName), appc[cnfName], appIPMap)
			}

		}
	}
	return appLbResult
}

func (p *Parser) AssignSwitchGWs(version, ipcidr, wlName, netwType *string, idx, gwidx int, netwInfo *NetworkInfo) {
	// initialize ipv4 or ipv6 subnet
	p.DeploymentIPAM[*wlName][*netwType][*ipcidr] = new(IpamApp)

	// get switch index from the netwType
	switchIndex := 0
	networkIndex := 0
	split := strings.Split(*netwType, ".")
	if len(split) > 1 {
		// sriov
		var err error
		networkIndex, err = strconv.Atoi(split[1])
		if err != nil {
			log.Fatalf("error in sriov definition: -> sriov1.1 or sriov2.1, first integer represents the switch, 2nd integer represents the subnet")
		}
		switchIndex, err = strconv.Atoi(strings.TrimPrefix(split[0], "sriov"))
		if err != nil {
			log.Fatalf("error in sriov definition: -> sriov1.1 or sriov2.1")
		}
		log.Debugf("Sriov Name %s, NetworkIndex %d, SwitchIndex %d", *netwType, networkIndex, switchIndex)

	} else {
		// ipvlan, do nothing since ipvlan is connected via a bond and multi-homing
		// network Index = 0
		// switchIndex = 0
	}

	// initialize the gw with netwType
	switch *version {
	case "ipv4":
		if len(p.SwitchInfo.switchGwsIPv4[*wlName][*netwType]) == 0 {
			p.SwitchInfo.switchGwsIPv4[*wlName][*netwType] = make(map[int]map[int]string)
		}
		if len(p.SwitchInfo.switchGwsIPv4[*wlName][*netwType][switchIndex]) == 0 {
			p.SwitchInfo.switchGwsIPv4[*wlName][*netwType][switchIndex] = make(map[int]string)
		}
		if len(p.SwitchInfo.switchGwsPerWlNameIpv4[*wlName][switchIndex]) == 0 {
			p.SwitchInfo.switchGwsPerWlNameIpv4[*wlName][switchIndex] = make([]string, 0)
		}
	case "ipv6":
		if len(p.SwitchInfo.switchGwsIPv6[*wlName][*netwType]) == 0 {
			p.SwitchInfo.switchGwsIPv6[*wlName][*netwType] = make(map[int]map[int]string)
		}
		if len(p.SwitchInfo.switchGwsPerWlNameIpv6[*wlName][switchIndex]) == 0 {
			p.SwitchInfo.switchGwsPerWlNameIpv6[*wlName][switchIndex] = make([]string, 0)
		}

		if len(p.SwitchInfo.switchGwsIPv6[*wlName][*netwType][switchIndex]) == 0 {
			p.SwitchInfo.switchGwsIPv6[*wlName][*netwType][switchIndex] = make(map[int]string)
		}
	}

	// assign Ipv4 or IPv6 Gateway
	_, ipNet, err := net.ParseCIDR(*ipcidr)
	if err != nil {
		log.WithError(err).Error("Cidr Parsing error")
	} else {
		gw, err := cidr.Host(ipNet, gwidx)
		if err != nil {
			log.WithError(err).Error("Host Parsing error")
		} else {
			p.DeploymentIPAM[*wlName][*netwType][*ipcidr].Gateway = StringPtr(gw.String())
			p.DeploymentIPAM[*wlName][*netwType][*ipcidr].VlanID = netwInfo.VlanID

			log.Debugf("gatways: %s", *netwType)
			switch *version {
			case "ipv4":
				p.SwitchInfo.switchGwsIPv4[*wlName][*netwType][switchIndex][idx] = gw.String()
				p.SwitchInfo.switchGwsPerWlNameIpv4[*wlName][switchIndex] = append(p.SwitchInfo.switchGwsPerWlNameIpv4[*wlName][switchIndex], gw.String())
			case "ipv6":
				p.SwitchInfo.switchGwsIPv6[*wlName][*netwType][switchIndex][idx] = gw.String()
				p.SwitchInfo.switchGwsPerWlNameIpv6[*wlName][switchIndex] = append(p.SwitchInfo.switchGwsPerWlNameIpv6[*wlName][switchIndex], gw.String())
			}
		}
	}
}

func (p *Parser) AssignSwitchBgpLoopback(version, ipcidr, wlName, netwType *string, bgpidx int) {
	// initialize sriov or ipvlan
	p.DeploymentIPAM[*wlName][*netwType] = make(map[string]*IpamApp)

	// initialize ipv4 subnet
	p.DeploymentIPAM[*wlName][*netwType][*ipcidr] = new(IpamApp)

	_, ipNet, err := net.ParseCIDR(*ipcidr)
	if err != nil {
		log.WithError(err).Error("Cidr Parsing error")
	}

	// assign leaf1 BGP Loopback
	// we use the connectivity map of sriov which gives us the real switch names and amounts
	// e.g. map[master0:[leaf1 leaf2]]
	// We are assuming uniform connectivity
	once := true
	for _, clientLinkInfo := range p.ClientServer2NetworkLinks["sriov"] {
		for _, numaInfo := range clientLinkInfo {
			if once {
				for switchName := range numaInfo {
					switchIndex, err := strconv.Atoi(switchName[len(switchName)-1:])
					if err != nil {
						log.Fatalf("switch name should end with a integer")
					}

					// assign BGP Loopback
					var allocateIP *AllocatedIPInfo
					allocateIP, err = AllocateIPIndex(StringPtr(switchName), StringPtr("BGP loopback"), IntPtr(bgpidx+switchIndex-1), ipNet, p.DeploymentIPAM[*wlName][*netwType])
					if err != nil {
						log.WithError(err).Errorf("Allocating IP error")
					}

					switch *version {
					case "ipv4":
						p.SwitchInfo.switchBgpPeersIPv4[*wlName][switchIndex] = &BGPPeerInfo{
							IP: allocateIP.IPAddress,
							AS: p.Nodes[switchName].AS,
						}
					case "ipv6":
						p.SwitchInfo.switchBgpPeersIPv6[*wlName][switchIndex] = &BGPPeerInfo{
							IP: allocateIP.IPAddress,
							AS: p.Nodes[switchName].AS,
						}
					}
				}
				once = false
			}
		}
	}
}

// initializes the POD/Container information per CNF
func (a *AppConfig) InitializeCnfContainerData(c *ContainerRegistry, pods map[string]map[string]interface{}) {
	// initialize containe registry
	var containerRegistry string
	var containerRegistryImageDir string
	var containerRegistrySecret string
	if c.Server != nil {
		containerRegistry = *c.Server
	}
	if c.ImageDir != nil {
		containerRegistryImageDir = *c.ImageDir
	}
	if c.Secret != nil {
		containerRegistrySecret = *c.Secret
	}

	a.ContainerRepo = &ContainerRepo{
		ImageRepo:   StringPtr(containerRegistry + "/" + containerRegistryImageDir),
		ImageSecret: StringPtr(containerRegistrySecret),
	}

	// initialize the CNF POD/container information
	a.Containers = make(map[string]*ContainerInfo)
	for podName, podInfo := range pods {
		// specific for AMF
		switch podName {
		case "dbs":
			a.Dbs = false
			if v, ok := podInfo["enabled"]; ok {
				switch enabled := v.(type) {
				case bool:
					a.Dbs = enabled
				}
			}
		case "emms_amms":
			a.Emms = false
			a.Amms = false
			if v, ok := podInfo["enabled"]; ok {
				switch enabled := v.(type) {
				case bool:
					a.Amms = enabled
				}
			}
		case "ipds":
			a.Ipds = false
			if v, ok := podInfo["enabled"]; ok {
				switch enabled := v.(type) {
				case bool:
					a.Ipds = enabled
				}
			}
		case "ipps":
			a.Ipps = false
			if v, ok := podInfo["enabled"]; ok {
				switch enabled := v.(type) {
				case bool:
					a.Ipps = enabled
				}
			}
		case "necc":
			a.Necc = false
			if v, ok := podInfo["enabled"]; ok {
				switch enabled := v.(type) {
				case bool:
					a.Necc = enabled
				}
			}
		case "paps":
			a.Paps = false
			if v, ok := podInfo["enabled"]; ok {
				switch enabled := v.(type) {
				case bool:
					a.Paps = enabled
				}
			}
		default:
		}
		var enabled bool
		if v, ok := podInfo["enabled"]; ok {
			switch e := v.(type) {
			case bool:
				enabled = e
			}
		}

		// generic for all pods: amf, upf, smf
		var tag string
		if podtag, ok := podInfo["tag"]; ok {
			switch podtag.(type) {
			case string:
				tag = podtag.(string)
			}
		}
		var cpu int
		if podcpu, ok := podInfo["cpu"]; ok {
			switch podcpu.(type) {
			case int:
				cpu = podcpu.(int)
			}
		}
		var mem string
		if podmem, ok := podInfo["memory"]; ok {
			switch podmem.(type) {
			case string:
				mem = podmem.(string)
			}
		}
		var hugepages string
		if podhp, ok := podInfo["hugepages1Gi"]; ok {
			switch podhp.(type) {
			case string:
				hugepages = podhp.(string)
			}
		}
		var nodeSelector string
		if podnodeSelector, ok := podInfo["nodeSelector"]; ok {
			switch podnodeSelector.(type) {
			case string:
				nodeSelector = podnodeSelector.(string)
			default:
				log.Debugf("Type: %v", reflect.TypeOf(podnodeSelector))
			}
		}
		antiaffinity := make([]string, 0)
		if podantiaffinity, ok := podInfo["antiaffinity"]; ok {
			switch x := podantiaffinity.(type) {
			case []interface{}:
				for _, v := range x {
					switch v.(type) {
					case string:
						antiaffinity = append(antiaffinity, v.(string))
					}
				}
			default:
				log.Debugf("Type: %v", reflect.TypeOf(podantiaffinity))

			}
		}
		var initialDelaySeconds int
		if podinitialDelaySeconds, ok := podInfo["initialDelaySeconds"]; ok {
			switch podinitialDelaySeconds.(type) {
			case int:
				initialDelaySeconds = podinitialDelaySeconds.(int)
			}
		}
		var periodSeconds int
		if podperiodSeconds, ok := podInfo["periodSeconds"]; ok {
			switch podperiodSeconds.(type) {
			case int:
				periodSeconds = podperiodSeconds.(int)
			}
		}

		imageName := podName
		if podName == "emms_amms" {
			imageName = "cpps"
		}
		a.Containers[podName] = &ContainerInfo{
			Enabled:             BoolPtr(enabled),
			ImageName:           StringPtr(imageName),
			ImageTag:            StringPtr(tag),
			CPU:                 IntPtr(cpu),
			Memory:              StringPtr(mem),
			Hugepages1Gi:        StringPtr(hugepages),
			NodeSelector:        StringPtr(nodeSelector),
			Antiaffinity:        antiaffinity,
			InitialDelaySeconds: IntPtr(initialDelaySeconds),
			PeriodSeconds:       IntPtr(periodSeconds),
		}
	}
}

func getSwitchIndexes(netwType string) (int, int) {
	switchIndex := 0
	networkIndex := 0
	split := strings.Split(netwType, ".")
	if len(split) > 1 {
		// sriov
		var err error
		networkIndex, err = strconv.Atoi(split[1])
		if err != nil {
			log.Fatalf("error in sriov definition: -> sriov1.1 or sriov2.1, first integer represents the switch, 2nd integer represents the subnet")
		}
		switchIndex, err = strconv.Atoi(strings.TrimPrefix(split[0], "sriov"))
		if err != nil {
			log.Fatalf("error in sriov definition: -> sriov1.1 or sriov2.1")
		}
		log.Debugf("Sriov Name %s, NetworkIndex %d, SwitchIndex %d", netwType, networkIndex, switchIndex)

	} else {
		// ipvlan, do nothing since ipvlan is connected via a bond and multi-homing
		// network Index = 0
		// switchIndex = 0
	}
	return switchIndex, networkIndex

}

func getUpGatewaysPerLmg(switchInfo *switchInfo, wlName *string, pods *int) (map[int][]string, map[int][]string) {
	upIpv4GWs := make(map[int][]string)
	upIpv6GWs := make(map[int][]string)

	for netwType := range switchInfo.switchGwsIPv4[*wlName] {
		if strings.Contains(netwType, "sriov") {
			// indicates which switch the network is connected to
			switchIndex, networkIndex := getSwitchIndexes(netwType)
			log.Infof("getUpGatewaysPerLmg: %d, %d %d", switchIndex, networkIndex, *pods)
			if networkIndex <= *pods {
				upIpv4GWs[switchIndex] = append(upIpv4GWs[switchIndex], switchInfo.switchGwsIPv4[*wlName][netwType][switchIndex][networkIndex-1])
			}
		}
	}

	for netwType := range switchInfo.switchGwsIPv6[*wlName] {
		if strings.Contains(netwType, "sriov") {
			// indicates which switch the network is connected to
			switchIndex, networkIndex := getSwitchIndexes(netwType)
			if networkIndex <= *pods {
				upIpv6GWs[switchIndex] = append(upIpv6GWs[switchIndex], switchInfo.switchGwsIPv4[*wlName][netwType][switchIndex][networkIndex-1])
			}
		}
	}

	return upIpv4GWs, upIpv6GWs
}

func getUpGateways(switchInfo *switchInfo, wlName, deployment *string) (map[int][]string, map[int][]string) {
	upIpv4GWs := make(map[int][]string)
	upIpv6GWs := make(map[int][]string)

	switch *deployment {
	case "ntok":
		// switch 1 and switch 2
		upIpv4GWs[1] = make([]string, 0)
		upIpv4GWs[2] = make([]string, 0)
		upIpv4GWs[1] = append(upIpv4GWs[1], switchInfo.switchGwsIPv4[*wlName]["sriov1.1"][1][0])
		upIpv4GWs[2] = append(upIpv4GWs[2], switchInfo.switchGwsIPv4[*wlName]["sriov2.1"][2][0])

		upIpv6GWs[1] = make([]string, 0)
		upIpv6GWs[2] = make([]string, 0)
		upIpv6GWs[1] = append(upIpv6GWs[1], switchInfo.switchGwsIPv6[*wlName]["sriov1.1"][1][0])
		upIpv6GWs[2] = append(upIpv6GWs[2], switchInfo.switchGwsIPv6[*wlName]["sriov2.1"][2][0])
	case "1to1":
		// switch 1 and switch 2
		upIpv4GWs[1] = make([]string, 0)
		upIpv4GWs[2] = make([]string, 0)
		upIpv4GWs[1] = append(upIpv4GWs[1], switchInfo.switchGwsIPv4[*wlName]["sriov1.1"][1][0])
		upIpv4GWs[2] = append(upIpv4GWs[2], switchInfo.switchGwsIPv4[*wlName]["sriov2.1"][2][0])
		upIpv4GWs[1] = append(upIpv4GWs[1], switchInfo.switchGwsIPv4[*wlName]["sriov1.2"][1][1])
		upIpv4GWs[2] = append(upIpv4GWs[2], switchInfo.switchGwsIPv4[*wlName]["sriov2.2"][2][1])

		upIpv6GWs[1] = make([]string, 0)
		upIpv6GWs[2] = make([]string, 0)
		upIpv6GWs[1] = append(upIpv6GWs[1], switchInfo.switchGwsIPv6[*wlName]["sriov1.1"][1][0])
		upIpv6GWs[2] = append(upIpv6GWs[2], switchInfo.switchGwsIPv6[*wlName]["sriov2.1"][2][0])
		upIpv6GWs[1] = append(upIpv6GWs[1], switchInfo.switchGwsIPv6[*wlName]["sriov1.2"][1][1])
		upIpv6GWs[2] = append(upIpv6GWs[2], switchInfo.switchGwsIPv6[*wlName]["sriov2.2"][2][1])
	}

	return upIpv4GWs, upIpv6GWs
}

func AllocateIP(ipAddresses *map[string]map[int]map[string][]*AllocatedIPInfo, cnfName *string, group *int, allocShortName, allocLongName *string, ipIndex *int, ipv4Net, ipv6Net *net.IPNet, deployIPAM map[string]*IpamApp) (*string, *string) {
	if len((*ipAddresses)["ipv4"][*group][*allocShortName]) == 0 {
		(*ipAddresses)["ipv4"][*group][*allocShortName] = make([]*AllocatedIPInfo, 0)
		(*ipAddresses)["ipv6"][*group][*allocShortName] = make([]*AllocatedIPInfo, 0)
	}
	if *cnfName == "upf" && *allocShortName == "intIP" {
		log.Debugf("Allocation: %s, %d, %d", *allocShortName, *group, *ipIndex)
	}

	log.Debugf("allocateTBD: ipv4 net %v, ipv6 net %v", *ipv4Net, *ipv6Net)

	allocateIPv4, err := AllocateIPIndex(cnfName, allocLongName, ipIndex, ipv4Net, deployIPAM)
	if err != nil {
		log.WithError(err).Errorf("Allocating IP error")
	}
	(*ipAddresses)["ipv4"][*group][*allocShortName] = append((*ipAddresses)["ipv4"][*group][*allocShortName], allocateIPv4)

	allocateIPv6, err := AllocateIPIndex(cnfName, allocLongName, ipIndex, ipv6Net, deployIPAM)
	if err != nil {
		log.WithError(err).Errorf("Allocating IP error")
	}
	(*ipAddresses)["ipv6"][*group][*allocShortName] = append((*ipAddresses)["ipv6"][*group][*allocShortName], allocateIPv6)

	return allocateIPv4.IPAddress, allocateIPv6.IPAddress
}

func (a *AppConfig) AssignNetworkInfoLoopback(group *int, itfceType, wlName, multusGenericWlName *string, netwInfo *NetworkInfo, ipv4PrefixLength, ipv6PrefixLength *int, ipAddresses *map[string]map[int]map[string][]*AllocatedIPInfo, switchInfo *switchInfo, cnfInfo *CnfInfo, cnfName, ipv4BGPAddress, ipv6BGPAddress *string, multusInfo map[string]*MultusInfo) {

	if len(a.Networks[*multusGenericWlName][*group]) == 0 {
		a.Networks[*multusGenericWlName][*group] = make(map[int]map[string]map[string][]*RenderedNetworkInfo)
		a.Networks[*multusGenericWlName][*group][0] = make(map[string]map[string][]*RenderedNetworkInfo)
		a.Networks[*multusGenericWlName][*group][0]["loopback"] = make(map[string][]*RenderedNetworkInfo)
		a.Networks[*multusGenericWlName][*group][0]["loopback"]["lmgLbk"] = make([]*RenderedNetworkInfo, 0)
		a.Networks[*multusGenericWlName][*group][0]["loopback"]["sigLbk"] = make([]*RenderedNetworkInfo, 0)
		a.Networks[*multusGenericWlName][*group][0]["itfce"] = make(map[string][]*RenderedNetworkInfo)
		a.Networks[*multusGenericWlName][*group][0]["itfce"]["intIP"] = make([]*RenderedNetworkInfo, 0)
	} else {
		if len(a.Networks[*multusGenericWlName][*group][0]) == 0 {
			a.Networks[*multusGenericWlName][*group][0] = make(map[string]map[string][]*RenderedNetworkInfo)
			a.Networks[*multusGenericWlName][*group][0]["loopback"] = make(map[string][]*RenderedNetworkInfo)
			a.Networks[*multusGenericWlName][*group][0]["loopback"]["lmgLbk"] = make([]*RenderedNetworkInfo, 0)
			a.Networks[*multusGenericWlName][*group][0]["loopback"]["sigLbk"] = make([]*RenderedNetworkInfo, 0)
			a.Networks[*multusGenericWlName][*group][0]["itfce"] = make(map[string][]*RenderedNetworkInfo)
			a.Networks[*multusGenericWlName][*group][0]["itfce"]["intIP"] = make([]*RenderedNetworkInfo, 0)
		} else {
			if len(a.Networks[*multusGenericWlName][*group][0]["loopback"]) == 0 {
				a.Networks[*multusGenericWlName][*group][0]["loopback"] = make(map[string][]*RenderedNetworkInfo)
				a.Networks[*multusGenericWlName][*group][0]["loopback"]["intIP"] = make([]*RenderedNetworkInfo, 0)
			}
		}
	}

	// TODO -> handling different addressing scheme, we assume dual stack for now
	var ipv4Cidr *string
	var ipv6Cidr *string
	for i := 0; i < len(netwInfo.Ipv4Cidr); i++ {
		ipv4Cidr = netwInfo.Ipv4Cidr[i]
		ipv6Cidr = netwInfo.Ipv6Cidr[i]

		rnInfo := &RenderedNetworkInfo{
			Ipv4Cidr:         ipv4Cidr,
			Ipv6Cidr:         ipv6Cidr,
			Ipv4PrefixLength: ipv4PrefixLength,
			Ipv6PrefixLength: ipv6PrefixLength,
			Ipv4Addresses:    (*ipAddresses)["ipv4"][*group][*itfceType],
			Ipv6Addresses:    (*ipAddresses)["ipv6"][*group][*itfceType],
			Target:           netwInfo.Target,
			//InterfaceName:    clientLinkInfo.InterfaceName,
			VrfCpId:        multusInfo[*multusGenericWlName].VrfCpId,
			IPv4BGPAddress: ipv4BGPAddress,
			IPv6BGPAddress: ipv6BGPAddress,
			IPv4BGPPeers:   switchInfo.switchBgpPeersIPv4[*wlName],
			IPv6BGPPeers:   switchInfo.switchBgpPeersIPv6[*wlName],
			Ipv4GwPerWl:    switchInfo.switchGwsPerWlNameIpv4[*wlName],
			Ipv6GwPerWl:    switchInfo.switchGwsPerWlNameIpv6[*wlName],
			AS:             cnfInfo.Networking.AS,
		}
		a.Networks[*multusGenericWlName][*group][0]["loopback"][*itfceType] = append(a.Networks[*multusGenericWlName][*group][0]["loopback"][*itfceType], rnInfo)

		if *itfceType == "lmgLpb" && a.Lmgs != nil {
			upIpv4GWs, upIpv6GWs := getUpGatewaysPerLmg(switchInfo, wlName, a.Lmgs)
			rnInfo.Ipv4GwPerWl = upIpv4GWs
			rnInfo.Ipv6GwPerWl = upIpv6GWs
		}

		if *group > 0 {
			// this is lmg signalling -> the gatway is limited to the 1st subnet and maybe 2nd subnet for 1to1
			upIpv4GWs, upIpv6GWs := getUpGateways(switchInfo, wlName, cnfInfo.Deployment)
			rnInfo.Ipv4GwPerWl = upIpv4GWs
			rnInfo.Ipv6GwPerWl = upIpv6GWs
		}

		switch *multusGenericWlName {
		case "oam":
			rnInfo.VrfUpId = multusInfo[*multusGenericWlName].VrfUpId
			rnInfo.NetworkShortName = StringPtr("oam")
		case "3GPP_Internal":
			rnInfo.VrfUpId = multusInfo[*multusGenericWlName].VrfUpId
			rnInfo.NetworkShortName = StringPtr("int")
		case "3GPP_External":
			rnInfo.VrfUpId = multusInfo[*multusGenericWlName].VrfUpId
			rnInfo.NetworkShortName = StringPtr("ext")
		case "3GPP_SBA":
			rnInfo.VrfUpId = multusInfo[*multusGenericWlName].VrfUpId
			rnInfo.NetworkShortName = StringPtr("sba")
		case "3GPP_Internet":
			rnInfo.VrfUpId = multusInfo[*multusGenericWlName].VrfUpId
			rnInfo.NetworkShortName = StringPtr("gilan")
		}
	}

}

func (a *AppConfig) AssignNetworkInfoItfce(group *int, itfceType, wlName, multusGenericWlName *string, netwInfo *NetworkInfo, ipv4PrefixLength, ipv6PrefixLength *int, ipAddresses *map[string]map[int]map[string][]*AllocatedIPInfo, switchInfo *switchInfo, cnfInfo *CnfInfo, cnfName *string, networkIndex, switchIndex *int, clientLinkInfo *ClientLinkInfo, fipv4, fipv6, connType, netwType *string, multusInfo map[string]*MultusInfo) {

	log.Debugf("AssignNetworkInfoItfce: %s %d %s", *wlName, *group, *itfceType)

	if len(a.Networks[*multusGenericWlName][*group]) == 0 {
		a.Networks[*multusGenericWlName][*group] = make(map[int]map[string]map[string][]*RenderedNetworkInfo)
		a.Networks[*multusGenericWlName][*group][*switchIndex] = make(map[string]map[string][]*RenderedNetworkInfo)
		a.Networks[*multusGenericWlName][*group][*switchIndex]["loopback"] = make(map[string][]*RenderedNetworkInfo)
		a.Networks[*multusGenericWlName][*group][*switchIndex]["loopback"]["lmgLbk"] = make([]*RenderedNetworkInfo, 0)
		a.Networks[*multusGenericWlName][*group][*switchIndex]["loopback"]["sigLbk"] = make([]*RenderedNetworkInfo, 0)
		a.Networks[*multusGenericWlName][*group][*switchIndex]["itfce"] = make(map[string][]*RenderedNetworkInfo)
		a.Networks[*multusGenericWlName][*group][*switchIndex]["itfce"]["intIP"] = make([]*RenderedNetworkInfo, 0)
	} else {
		if len(a.Networks[*multusGenericWlName][*group][*switchIndex]) == 0 {
			a.Networks[*multusGenericWlName][*group][*switchIndex] = make(map[string]map[string][]*RenderedNetworkInfo)
			a.Networks[*multusGenericWlName][*group][*switchIndex]["loopback"] = make(map[string][]*RenderedNetworkInfo)
			a.Networks[*multusGenericWlName][*group][*switchIndex]["loopback"]["lmgLbk"] = make([]*RenderedNetworkInfo, 0)
			a.Networks[*multusGenericWlName][*group][*switchIndex]["loopback"]["sigLbk"] = make([]*RenderedNetworkInfo, 0)
			a.Networks[*multusGenericWlName][*group][*switchIndex]["itfce"] = make(map[string][]*RenderedNetworkInfo)
			a.Networks[*multusGenericWlName][*group][*switchIndex]["itfce"]["intIP"] = make([]*RenderedNetworkInfo, 0)
		} else {
			if len(a.Networks[*multusGenericWlName][*group][*switchIndex]["itfce"]) == 0 {
				a.Networks[*multusGenericWlName][*group][*switchIndex]["itfce"] = make(map[string][]*RenderedNetworkInfo)
				a.Networks[*multusGenericWlName][*group][*switchIndex]["itfce"]["intIP"] = make([]*RenderedNetworkInfo, 0)
			}
		}
	}

	// TODO -> handling different addressing scheme, we assume dual stack for now
	var ipv4Cidr *string
	var ipv6Cidr *string
	for i := 0; i < len(netwInfo.Ipv4Cidr); i++ {
		ipv4Cidr = netwInfo.Ipv4Cidr[i]
		ipv6Cidr = netwInfo.Ipv6Cidr[i]

		rnInfo := &RenderedNetworkInfo{
			ResShortName:     StringPtr(strings.TrimLeft(*netwType, *connType)),
			NetworkIndex:     networkIndex,
			SwitchIndex:      switchIndex,
			InterfaceName:    clientLinkInfo.InterfaceName,
			Numa:             clientLinkInfo.Numa,
			Cni:              connType,
			VlanID:           netwInfo.VlanID,
			Ipv4Cidr:         ipv4Cidr,
			Ipv6Cidr:         ipv6Cidr,
			Ipv4PrefixLength: ipv4PrefixLength,
			Ipv6PrefixLength: ipv6PrefixLength,
			Ipv4Addresses:    (*ipAddresses)["ipv4"][*group][*itfceType],
			Ipv6Addresses:    (*ipAddresses)["ipv6"][*group][*itfceType],
			Ipv4FloatingIP:   fipv4,
			Ipv6FloatingIP:   fipv6,
			Ipv4Gw:           StringPtr(switchInfo.switchGwsIPv4[*wlName][*netwType][*switchIndex][0]),
			Ipv6Gw:           StringPtr(switchInfo.switchGwsIPv6[*wlName][*netwType][*switchIndex][0]),
			Target:           netwInfo.Target,
			VrfCpId:          multusInfo[*multusGenericWlName].VrfCpId,
			IPv4BGPPeers:     switchInfo.switchBgpPeersIPv4[*wlName],
			IPv6BGPPeers:     switchInfo.switchBgpPeersIPv6[*wlName],
			Ipv4GwPerWl:      switchInfo.switchGwsPerWlNameIpv4[*wlName],
			Ipv6GwPerWl:      switchInfo.switchGwsPerWlNameIpv6[*wlName],
			AS:               cnfInfo.Networking.AS,
		}
		a.Networks[*multusGenericWlName][*group][*switchIndex]["itfce"][*itfceType] = append(a.Networks[*multusGenericWlName][*group][*switchIndex]["itfce"][*itfceType], rnInfo)

		if *group > 0 {
			// this is lmg signalling -> the gatway is limited to the 1st subnet and maybe 2nd subnet for 1to1
			upIpv4GWs, upIpv6GWs := getUpGateways(switchInfo, wlName, cnfInfo.Deployment)
			rnInfo.Ipv4GwPerWl = upIpv4GWs
			rnInfo.Ipv6GwPerWl = upIpv6GWs
		}

		switch *multusGenericWlName {
		case "oam":
			rnInfo.VrfUpId = multusInfo[*multusGenericWlName].VrfUpId
			rnInfo.NetworkShortName = StringPtr("oam")
			rnInfo.InterfaceEthernetName = StringPtr("eth2")
		case "3GPP_Internal":
			rnInfo.VrfUpId = multusInfo[*multusGenericWlName].VrfUpId
			rnInfo.NetworkShortName = StringPtr("int")
			rnInfo.InterfaceEthernetName = StringPtr("eth3")
		case "3GPP_External":
			rnInfo.VrfUpId = multusInfo[*multusGenericWlName].VrfUpId
			rnInfo.NetworkShortName = StringPtr("ext")
			rnInfo.InterfaceEthernetName = StringPtr("eth3")
		case "3GPP_SBA":
			rnInfo.VrfUpId = multusInfo[*multusGenericWlName].VrfUpId
			rnInfo.NetworkShortName = StringPtr("sba")
			rnInfo.InterfaceEthernetName = StringPtr("eth4")
		case "3GPP_Internet":
			rnInfo.VrfUpId = multusInfo[*multusGenericWlName].VrfUpId
			rnInfo.NetworkShortName = StringPtr("gilan")
		}
	}
}

func (a *AppConfig) UpdateWorkloadShortNames(multusGenericWlName *string) {
	switch *multusGenericWlName {
	case "oam":
		a.WorkloadShortNames[*multusGenericWlName] = StringPtr("oam")
	case "3GPP_Internal":
		a.WorkloadShortNames[*multusGenericWlName] = StringPtr("int")
	case "3GPP_External":
		a.WorkloadShortNames[*multusGenericWlName] = StringPtr("ext")
	case "3GPP_SBA":
		a.WorkloadShortNames[*multusGenericWlName] = StringPtr("sba")
	case "3GPP_Internet":
		a.WorkloadShortNames[*multusGenericWlName] = StringPtr("gilan")
	}
}

func (p *Parser) GetApplicationIndex(itfceType, element, kind *string) *int {
	// get application indexes
	if _, ok := p.Config.AppNetworkIndexes[*itfceType][*element][*kind]; !ok {
		log.Fatalf(fmt.Sprintf("Assign app-netw-indexes/%s/%s/%s", *itfceType, *element, *kind))
	}
	return p.Config.AppNetworkIndexes[*itfceType][*element][*kind]
}

// initialize the network data per cnf
func (a *AppConfig) InitializeCnfNetworkData(cnfName string, cnfInfo *CnfInfo, p *Parser, appIPMap *AppIPMap, clientServer2NetworkLinks map[string]map[string]map[int]map[string][]*string, switchInfo *switchInfo, appLbResults *types.AppLbIpResult) {
	// initialize switches
	a.SwitchesPerServer = p.SwitchInfo.switchesPerServer
	// initialize networks
	a.Networks = make(map[string]map[int]map[int]map[string]map[string][]*RenderedNetworkInfo)
	// sriov or ipvlan
	connType := cnfInfo.Networking.Type
	log.Debugf("ConnectionType: %s", *connType)
	a.ConnType = connType

	// initialize client links
	a.UniqueClientServer2NetworkLinks = make(map[string]map[string]map[int]map[string][]*string)
	//a.ClientLinks = make(map[string]map[string]*ClientLinkInfo)
	//a.ClientLinks[*connType] = make(map[string]*ClientLinkInfo)
	//a.UniqueClientLinks = make(map[string][]*ClientLinkInfo)
	//a.UniqueClientLinks[*connType] = make([]*ClientLinkInfo, 0)

	// initialize WorkloadShortNames
	a.WorkloadShortNames = make(map[string]*string)

	// initialize the BGP src per cnf
	// key1 is multus workload network, key2 is the cnfName; value is IP address
	bgpCnfSrcIPv4 := make(map[string]map[string]string)
	bgpCnfSrcIPv6 := make(map[string]map[string]string)

	// check all networks that are relevant for the app
	for multusGenericWlName, multusInfo := range cnfInfo.Networking.Multus {
		// loop over all networks in the workloads

		a.UpdateWorkloadShortNames(StringPtr(multusGenericWlName))

		for wlName, clients := range p.Config.Workloads {
			// if the network is relevant for the app we continue the processing

			if wlName == *multusInfo.WorkloadName {
				// initialize the bgp data per workload/multus network
				bgpCnfSrcIPv4[wlName] = make(map[string]string)
				bgpCnfSrcIPv6[wlName] = make(map[string]string)
				a.Networks[multusGenericWlName] = make(map[int]map[int]map[string]map[string][]*RenderedNetworkInfo)

				a.Networks[multusGenericWlName][0] = make(map[int]map[string]map[string][]*RenderedNetworkInfo)
				a.Networks[multusGenericWlName][0][0] = make(map[string]map[string][]*RenderedNetworkInfo)
				a.Networks[multusGenericWlName][0][0]["loopback"] = make(map[string][]*RenderedNetworkInfo)
				a.Networks[multusGenericWlName][0][0]["loopback"]["bgpLbk"] = make([]*RenderedNetworkInfo, 0)
				a.Networks[multusGenericWlName][0][0]["loopback"]["sysLbk"] = make([]*RenderedNetworkInfo, 0)
				a.Networks[multusGenericWlName][0][0]["loopback"]["llbLbk"] = make([]*RenderedNetworkInfo, 0)

				for _, wlInfo := range clients {
					for netwType, netwInfo := range wlInfo.Loopbacks {
						// initialize the loopback based networking informaation
						// BGP, LLB, LMG loopbacks
						// SMF, UPF loopbacks per multus network type -> TBD
						if strings.Contains(netwType, "loopback") {

							// TODO -> handling different addressing scheme, we assume dual stack for now

							for i := 0; i < len(netwInfo.Ipv4Cidr); i++ {

								_, ipv4Net, err := net.ParseCIDR(*netwInfo.Ipv4Cidr[i])
								if err != nil {
									log.WithError(err).Error("Cidr Parsing error")
								}
								_, ipv6Net, err := net.ParseCIDR(*netwInfo.Ipv6Cidr[i])
								if err != nil {
									log.WithError(err).Error("Cidr Parsing error")
								}

								// key1, ipv4 or ipv6, key2: group, key3: bgp/system/lmg-pod/llb-pod lbks
								// group: 0 -> amf, smf/upf llb; control-plane
								// group: 1..x -> upf lmg; user-plane
								ipAddresses := make(map[string]map[int]map[string][]*AllocatedIPInfo)
								ipAddresses["ipv4"] = make(map[int]map[string][]*AllocatedIPInfo)
								ipAddresses["ipv6"] = make(map[int]map[string][]*AllocatedIPInfo)
								ipAddresses["ipv4"][0] = make(map[string][]*AllocatedIPInfo)
								ipAddresses["ipv6"][0] = make(map[string][]*AllocatedIPInfo)

								var lmgs = 0
								var lmgPodsPerGroup = 0
								ipv4BGPAddress := new(string)
								ipv6BGPAddress := new(string)
								switch cnfName {
								case "smf":
									// allocate BGP Loopback
									//-> ipAddresses["ipv4"][0]["bgpLbk"] = make([]*AllocatedIPInfo, 0)
									bgpidx := p.GetApplicationIndex(StringPtr("loopback"), StringPtr(cnfName), StringPtr("bgp"))
									ipv4BGPAddress, ipv6BGPAddress = AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(0), StringPtr("bgpLbk"), StringPtr("BGP loopback"), bgpidx, ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

									// allocate SigLoopback
									//-> ipAddresses["ipv4"][0]["sigLbk"] = make([]*AllocatedIPInfo, 0)
									llbsigidx := p.GetApplicationIndex(StringPtr("loopback"), StringPtr(cnfName), StringPtr("llb-sig"))
									AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(0), StringPtr("sigLbk"), StringPtr("Sig loopback"), llbsigidx, ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

									appIPMap.IPinfo[multusGenericWlName].SmfIPv4 = ipAddresses["ipv4"][0]["sigLbk"][0].IPAddress
									appIPMap.IPinfo[multusGenericWlName].SmfIPv6 = ipAddresses["ipv6"][0]["sigLbk"][0].IPAddress

									// allocate System Loopback
									if multusGenericWlName == "3GPP_Internal" {
										sysidx := p.GetApplicationIndex(StringPtr("loopback"), StringPtr(cnfName), StringPtr("system"))
										AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(0), StringPtr("sysLbk"), StringPtr("System loopback"), sysidx, ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])
									}

									// allocate LLB loopbacks
									llbs := getLLBs(cnfInfo)
									a.Llbs = IntPtr(llbs)
									llbpodidx := p.GetApplicationIndex(StringPtr("loopback"), StringPtr(cnfName), StringPtr("llb-pod"))
									for i := 0; i < llbs; i++ {
										// allocate BGP Loopback
										llblinkipv4, llblinkipv6 := AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(0), StringPtr("llbLbk"), StringPtr("LLB loopback"), IntPtr(*llbpodidx+i), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])
										appLbResults.AddBgpIP(cnfName, wlName, types.NewIPInfo(*llblinkipv4, *llblinkipv6))
									}
								case "upf":
									// allocate BGP Loopback
									//-> ipAddresses["ipv4"][0]["bgpLbk"] = make([]*AllocatedIPInfo, 0)
									bgpidx := p.GetApplicationIndex(StringPtr("loopback"), StringPtr(cnfName), StringPtr("bgp"))
									ipv4BGPAddress, ipv6BGPAddress = AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(0), StringPtr("bgpLbk"), StringPtr("BGP loopback"), bgpidx, ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

									// allocate SigLoopback
									//-> ipAddresses["ipv4"][0]["sigLbk"] = make([]*AllocatedIPInfo, 0)
									llbsigidx := p.GetApplicationIndex(StringPtr("loopback"), StringPtr(cnfName), StringPtr("llb-sig"))
									AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(0), StringPtr("sigLbk"), StringPtr("Sig loopback"), llbsigidx, ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

									appIPMap.IPinfo[multusGenericWlName].UpfCpIPv4 = ipAddresses["ipv4"][0]["sigLbk"][0].IPAddress
									appIPMap.IPinfo[multusGenericWlName].UpfCpIPv6 = ipAddresses["ipv6"][0]["sigLbk"][0].IPAddress

									// allocate System Loopback
									if multusGenericWlName == "3GPP_Internal" {
										sysidx := p.GetApplicationIndex(StringPtr("loopback"), StringPtr(cnfName), StringPtr("system"))
										AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(0), StringPtr("sysLbk"), StringPtr("System loopback"), sysidx, ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])
									}

									// alloacte LLB loopbacks
									llbs := getLLBs(cnfInfo)
									a.Llbs = IntPtr(llbs)
									llbpodidx := p.GetApplicationIndex(StringPtr("loopback"), StringPtr(cnfName), StringPtr("llb-pod"))
									for i := 0; i < llbs; i++ {
										// allocate BGP Loopback
										//-> ipAddresses["ipv4"][0]["bgpLbk"] = make([]*AllocatedIPInfo, 0)
										llblinkipv4, llblinkipv6 := AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(0), StringPtr("llbLbk"), StringPtr("LLB loopback"), IntPtr(*llbpodidx+i), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])
										appLbResults.AddBgpIP(cnfName, wlName, types.NewIPInfo(*llblinkipv4, *llblinkipv6))
									}

									lmgs, lmgPodsPerGroup = getLMGs(cnfInfo, a.K)
									a.Lmgs = IntPtr(lmgs)
									log.Debugf("LMGS: %d", *a.Lmgs)

									// first loop over lmg groups = total lmgs divide bylmgPodsPerGroup
									// 2nd loop over lmgs per group
									lmgpodidx := p.GetApplicationIndex(StringPtr("loopback"), StringPtr(cnfName), StringPtr("lmg-pod"))
									lmgsigidx := p.GetApplicationIndex(StringPtr("loopback"), StringPtr(cnfName), StringPtr("lmg-sig"))
									for g := 1; g <= lmgs/lmgPodsPerGroup; g++ {
										ipAddresses["ipv4"][g] = make(map[string][]*AllocatedIPInfo)
										ipAddresses["ipv6"][g] = make(map[string][]*AllocatedIPInfo)

										for i := 0; i < lmgPodsPerGroup; i++ {
											// allocate BGP Loopback
											//-> ipAddresses["ipv4"][g]["lmgLbk"] = make([]*AllocatedIPInfo, 0)
											log.Debugf("lmgLbk: %d", g)

											AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(g), StringPtr("lmgLbk"), StringPtr("LMG loopback"), IntPtr(*lmgpodidx+i+g-1), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])
										}

										// allocate Sig Up Loopback per lmg group
										AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(g), StringPtr("sigLbk"), StringPtr("Sig loopback"), IntPtr(*lmgsigidx+g-1), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										//appIPMap.IPinfo[multusGenericWlName].UpfUpIPv4 = allocateIP.IPAddress
										//appIPMap.IPinfo[multusGenericWlName].UpfUpIPv6 = allocateIP.IPAddress
									}
								}

								log.Debugf("Loopbacks per cnf: %s", cnfName)

								// bgp loopback
								a.AssignNetworkInfoLoopback(IntPtr(0), StringPtr("bgpLbk"), &wlName, &multusGenericWlName, netwInfo, IntPtr(32), IntPtr(128), &ipAddresses, switchInfo, cnfInfo, &cnfName, ipv4BGPAddress, ipv6BGPAddress, p.Config.Application["paco"].Global.Multus)

								// system loopback
								a.AssignNetworkInfoLoopback(IntPtr(0), StringPtr("sysLbk"), &wlName, &multusGenericWlName, netwInfo, IntPtr(32), IntPtr(128), &ipAddresses, switchInfo, cnfInfo, &cnfName, ipv4BGPAddress, ipv6BGPAddress, p.Config.Application["paco"].Global.Multus)

								// signalling control plane loopback
								a.AssignNetworkInfoLoopback(IntPtr(0), StringPtr("sigLbk"), &wlName, &multusGenericWlName, netwInfo, IntPtr(32), IntPtr(128), &ipAddresses, switchInfo, cnfInfo, &cnfName, ipv4BGPAddress, ipv6BGPAddress, p.Config.Application["paco"].Global.Multus)

								if cnfName == "upf" {
									if lmgPodsPerGroup > 0 {
										for g := 1; g <= lmgs/lmgPodsPerGroup; g++ {
											// signalling control plane loopback per group
											a.AssignNetworkInfoLoopback(IntPtr(g), StringPtr("sigLbk"), &wlName, &multusGenericWlName, netwInfo, IntPtr(32), IntPtr(128), &ipAddresses, switchInfo, cnfInfo, &cnfName, ipv4BGPAddress, ipv4BGPAddress, p.Config.Application["paco"].Global.Multus)
										}
									}
								}

								// llb loopback
								a.AssignNetworkInfoLoopback(IntPtr(0), StringPtr("llbLbk"), &wlName, &multusGenericWlName, netwInfo, IntPtr(32), IntPtr(128), &ipAddresses, switchInfo, cnfInfo, &cnfName, ipv4BGPAddress, ipv4BGPAddress, p.Config.Application["paco"].Global.Multus)

								if lmgPodsPerGroup > 0 {
									log.Debugf("lmgLbk lmg lmgPodsPerGroup: %d %d", lmgs, lmgPodsPerGroup)
									for g := 1; g <= lmgs/lmgPodsPerGroup; g++ {
										// lmg loopback
										log.Debugf("lmgLbk: %d", g)
										a.AssignNetworkInfoLoopback(IntPtr(g), StringPtr("lmgLbk"), &wlName, &multusGenericWlName, netwInfo, IntPtr(32), IntPtr(128), &ipAddresses, switchInfo, cnfInfo, &cnfName, ipv4BGPAddress, ipv4BGPAddress, p.Config.Application["paco"].Global.Multus)
									}
								}
							}
						}
					}
					// initialize the Multus interface related information
					// netwType is ipvlan, sriov1.x, sriov2.x
					count := 0
					for netwType, netwInfo := range wlInfo.Itfces {
						// Allocate the CNF interface IP(s)

						// Only process the information that is relevant for the application
						// Only SRIOV or IPVLAN e.g.
						if strings.Contains(netwType, *connType) {
							// key1 is network/subnet/vlan; key2 = switch;
							// e.g. sriov1.1 map[1]map[1][]*AllocatedIPInfo
							// e.g. sriov2.6 map[6]map[2][]*AllocatedIPInfo
							// e.g. ipvlan map[0]map[0][]*AllocatedIPInfo
							ipAddresses := make(map[string]map[int]map[string][]*AllocatedIPInfo)
							ipAddresses["ipv4"] = make(map[int]map[string][]*AllocatedIPInfo)
							ipAddresses["ipv6"] = make(map[int]map[string][]*AllocatedIPInfo)
							ipAddresses["ipv4"][0] = make(map[string][]*AllocatedIPInfo)
							ipAddresses["ipv6"][0] = make(map[string][]*AllocatedIPInfo)

							// indicates which switch the network is connected to
							switchIndex, networkIndex := getSwitchIndexes(netwType)

							var fipv4 *string
							var fipv6 *string

							// TODO -> handling different addressing scheme, we assume dual stack for now
							var ipv4Cidr *string
							var ipv6Cidr *string
							var ipv4PrefixLength int
							var ipv6PrefixLength int
							var clientLink *ClientLinkInfo
							var lmgClientLink *ClientLinkInfo
							var lmgs = 0
							var lmgPodsPerGroup = 0
							for i := 0; i < len(netwInfo.Ipv4Cidr); i++ {
								ipv4Cidr = netwInfo.Ipv4Cidr[i]
								ipv6Cidr = netwInfo.Ipv6Cidr[i]

								_, ipv4Net, err := net.ParseCIDR(*ipv4Cidr)
								if err != nil {
									log.WithError(err).Error("Cidr Parsing error")
								}
								_, ipv6Net, err := net.ParseCIDR(*ipv6Cidr)
								if err != nil {
									log.WithError(err).Error("Cidr Parsing error")
								}

								ipv4PrefixLength, _ = ipv4Net.Mask.Size()
								ipv6PrefixLength, _ = ipv6Net.Mask.Size()

								lmgs = 0
								lmgPodsPerGroup = 0
								switch cnfName {
								case "smf":

									// allocate an ip per llb from each subnet, max 6 subnets (6 llbs per cnf upf)
									llbs := getLLBs(cnfInfo)

									llbitfcidx := p.GetApplicationIndex(StringPtr("itfce"), StringPtr(cnfName), StringPtr("llb"))
									if i < llbs && networkIndex <= llbs {
										log.Infof("SMF LLBs: %d, %d, %d", llbs, i, networkIndex)
										// allocate interface LLB
										v4llbip, v6llbip := AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(0), StringPtr("intIP"), StringPtr("interface LLB"), llbitfcidx, ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])
										appLbResults.AddLinkIP(cnfName, wlName, types.NewIPInfo(*v4llbip, *v6llbip))

									}

									// update unique links to ensure we map the right multus interface
									clientLink = p.UpdateUniqueClientLink(cnfInfo, StringPtr("llb"), connType, a.UniqueClientServer2NetworkLinks)

								case "upf":
									// allocate an ip per llb from each subnet, max 6 subnets (6 llbs per cnf upf)
									llbs := getLLBs(cnfInfo)

									// update unique links to ensure we map the right multus interface
									clientLink = p.UpdateUniqueClientLink(cnfInfo, StringPtr("llb"), connType, a.UniqueClientServer2NetworkLinks)

									//a.ClientLinks[*connType]["llb"] = clientLink

									//a.UpdateUniqueClientLinks(connType, clientLink)

									llbitfcidx := p.GetApplicationIndex(StringPtr("itfce"), StringPtr(cnfName), StringPtr("llb"))
									if i < llbs && networkIndex <= llbs {
										log.Infof("UPF LLBs: %d, %d, %d", llbs, i, networkIndex)
										// allocate interface LLB
										//-> ipAddresses["ipv4"][0]["intLbk"] = make([]*AllocatedIPInfo, 0)
										v4llbip, v6llbip := AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(0), StringPtr("intIP"), StringPtr("interface LLB"), llbitfcidx, ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])
										appLbResults.AddLinkIP(cnfName, wlName, types.NewIPInfo(*v4llbip, *v6llbip))
									}
									// the amount of subnets per sriov differs for ntok versus 1to1
									lmgs, lmgPodsPerGroup = getLMGs(cnfInfo, a.K)

									// update unique links to ensure we map the right multus interface
									lmgClientLink = p.UpdateUniqueClientLink(cnfInfo, StringPtr("lmg"), connType, a.UniqueClientServer2NetworkLinks)

									//lmgClientLink = p.GetClientLink(cnfInfo, StringPtr("lmg"), connType)

									//a.ClientLinks[*connType]["lmg"] = lmgClientLink

									//a.UpdateUniqueClientLinks(connType, lmgClientLink)

									// for LMG/UP only allocate gws for subnet1 and or 2 based on deployment

									// allocate the ips in the respective subnet based on th deployment strategy
									// for 1to1 we use 2 pods per group, for ntok 1 pod per group
									log.Debugf("lmgPodsPerGroup: %d, %d, %d", lmgPodsPerGroup, networkIndex, lmgs)
									lmgitfcidx := p.GetApplicationIndex(StringPtr("itfce"), StringPtr(cnfName), StringPtr("lmg"))
									if i < lmgPodsPerGroup && networkIndex <= lmgPodsPerGroup {
										log.Infof("UPF LMGs: %d, %d, %d %d", lmgs, lmgPodsPerGroup, i, networkIndex)
										// within the subnet we allocate the amount of ips based on the deployment strategy, we allocate based on lngs per subnet
										// indicate the amount of lmgPodsPerGroup

										// first loop over lmg groups = total lmgs divide by lmgPodsPerGroup
										// 2nd loop over lmgs per group
										for g := 1; g <= lmgs/lmgPodsPerGroup; g++ {
											ipAddresses["ipv4"][g] = make(map[string][]*AllocatedIPInfo)
											ipAddresses["ipv6"][g] = make(map[string][]*AllocatedIPInfo)
											for i := 0; i < lmgPodsPerGroup; i++ {
												AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(g), StringPtr("intIP"), StringPtr("interface LMG"), IntPtr(*lmgitfcidx+g-1), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])
											}
										}
									}

								case "amf":
									// IPv4 and IPv6
									// allocate 4 addresses out of the multus network
									itfcidx := p.GetApplicationIndex(StringPtr("itfce"), StringPtr(cnfName), StringPtr("int"))
									for i := 0; i < 4; i++ {
										AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(0), StringPtr("intIP"), StringPtr("interface AMF"), IntPtr(*itfcidx+i), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])
									}
									fipidx := p.GetApplicationIndex(StringPtr("itfce"), StringPtr(cnfName), StringPtr("fip"))
									fipv4, fipv6 = AllocateIP(&ipAddresses, StringPtr(cnfName), IntPtr(0), StringPtr("fipIP"), StringPtr("interface AMF"), fipidx, ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

									// TODO assignment of clientLinks[*connType][0] -> currently we take the first interface
									//clientLink = clientLinks[*connType][0]
									//clientLink = p.UpdateUniqueClientLink(cnfInfo, StringPtr("lmg"), connType, a.UniqueClientServer2NetworkLinks)
									for clientIntfaceName := range p.ClientServer2NetworkLinks[*connType] {
										clientLink = &ClientLinkInfo{&clientIntfaceName, IntPtr(0)}
									}

									// assign AMF related paco interface from the ipvlan range
									switch multusGenericWlName {
									case "oam":
									case "3GPP_External":

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("n2"), StringPtr(multusGenericWlName), IntPtr(110), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

									case "3GPP_Internal":

									case "3GPP_Internet":

									case "3GPP_SBA":

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("n8"), StringPtr(multusGenericWlName), IntPtr(108), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("n11"), StringPtr(multusGenericWlName), IntPtr(111), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("n12"), StringPtr(multusGenericWlName), IntPtr(112), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("n14"), StringPtr(multusGenericWlName), IntPtr(114), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("n15"), StringPtr(multusGenericWlName), IntPtr(115), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("n17"), StringPtr(multusGenericWlName), IntPtr(117), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("n22"), StringPtr(multusGenericWlName), IntPtr(122), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("n20"), StringPtr(multusGenericWlName), IntPtr(120), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("n26"), StringPtr(multusGenericWlName), IntPtr(126), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("nnrf"), StringPtr(multusGenericWlName), IntPtr(109), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("nsms"), StringPtr(multusGenericWlName), IntPtr(130), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("amfSvcDefaultIp"), StringPtr(multusGenericWlName), IntPtr(105), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("amfSvcLocIp"), StringPtr(multusGenericWlName), IntPtr(131), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("amfSvcComIp"), StringPtr(multusGenericWlName), IntPtr(132), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("amfSvcEeIp"), StringPtr(multusGenericWlName), IntPtr(133), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("amfSvcMtIp"), StringPtr(multusGenericWlName), IntPtr(134), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("nfyEirIp"), StringPtr(multusGenericWlName), IntPtr(135), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("nfyAmfIp"), StringPtr(multusGenericWlName), IntPtr(136), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("nfyAusfIp"), StringPtr(multusGenericWlName), IntPtr(137), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("nfyNrfIp"), StringPtr(multusGenericWlName), IntPtr(138), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("nfyNssfIp"), StringPtr(multusGenericWlName), IntPtr(139), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("nfyPcfIp"), StringPtr(multusGenericWlName), IntPtr(140), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("nfySmfIp"), StringPtr(multusGenericWlName), IntPtr(141), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("nfyudmIp"), StringPtr(multusGenericWlName), IntPtr(142), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("dns1"), StringPtr(multusGenericWlName), IntPtr(151), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("amf"), StringPtr("dns2"), StringPtr(multusGenericWlName), IntPtr(152), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("ausf"), StringPtr("ausf"), StringPtr(multusGenericWlName), IntPtr(107), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])

										appIPMap.AllocateAppLoopback(StringPtr("udm"), StringPtr("udm"), StringPtr(multusGenericWlName), IntPtr(106), ipv4Net, ipv6Net, p.DeploymentIPAM[wlName][netwType])
									}
								}
							}
							if len(ipAddresses) > 0 || len(ipAddresses) > 0 {

								count++
								log.Debugf("Count: %d", count)

								// LLB SMF, UPF and AMF interfaces
								a.AssignNetworkInfoItfce(IntPtr(0), StringPtr("intIP"), &wlName, &multusGenericWlName, netwInfo, IntPtr(ipv4PrefixLength), IntPtr(ipv6PrefixLength), &ipAddresses, switchInfo, cnfInfo, &cnfName, &networkIndex, &switchIndex, clientLink, fipv4, fipv6, connType, &netwType, p.Config.Application["paco"].Global.Multus)

								if lmgPodsPerGroup > 0 {
									for g := 1; g <= lmgs/lmgPodsPerGroup; g++ {
										log.Debugf("G: %d", g)
										// lmg Interfaces
										a.AssignNetworkInfoItfce(IntPtr(g), StringPtr("intIP"), &wlName, &multusGenericWlName, netwInfo, IntPtr(ipv4PrefixLength), IntPtr(ipv6PrefixLength), &ipAddresses, switchInfo, cnfInfo, &cnfName, &networkIndex, &switchIndex, lmgClientLink, fipv4, fipv6, connType, &netwType, p.Config.Application["paco"].Global.Multus)
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func (p *Parser) UpdateUniqueClientLink(cnfInfo *CnfInfo, podName, connType *string, uniqueClientServer2NetworkLinks map[string]map[string]map[int]map[string][]*string) *ClientLinkInfo {
	// for connecton type sriov we need to understand the numa placement
	var n int // numa
	if x, ok := cnfInfo.Pods[*podName]["numa"]; ok {
		switch v := x.(type) { //x is numa
		case int:
			n = v
		default:
			log.Fatal("Numa should be specified in LLB or LMG pods as integer")
		}
	} else {
		log.Fatal("Numa should be specified in LLB or LMG pods")
	}

	for clientLinkName, clientLinkInfo := range p.ClientServer2NetworkLinks[*connType] {
		for numa := range clientLinkInfo {
			if numa == n {
				if _, ok := uniqueClientServer2NetworkLinks[*connType]; !ok {
					uniqueClientServer2NetworkLinks[*connType] = make(map[string]map[int]map[string][]*string)
					uniqueClientServer2NetworkLinks[*connType][clientLinkName] = make(map[int]map[string][]*string)
					uniqueClientServer2NetworkLinks[*connType][clientLinkName][numa] = make(map[string][]*string)
				} else {
					if _, ok := uniqueClientServer2NetworkLinks[*connType][clientLinkName]; !ok {
						uniqueClientServer2NetworkLinks[*connType][clientLinkName] = make(map[int]map[string][]*string)
						uniqueClientServer2NetworkLinks[*connType][clientLinkName][numa] = make(map[string][]*string)
					} else {
						if _, ok := uniqueClientServer2NetworkLinks[*connType][clientLinkName][numa]; !ok {
							uniqueClientServer2NetworkLinks[*connType][clientLinkName][numa] = make(map[string][]*string)
						}
					}
				}
				uniqueClientServer2NetworkLinks[*connType][clientLinkName][numa] = p.ClientServer2NetworkLinks[*connType][clientLinkName][numa]
				return &ClientLinkInfo{&clientLinkName, &numa}
			}
		}
	}

	log.Fatal("Numa not found")
	return nil
}

func getLLBs(cnfInfo *CnfInfo) int {
	log.Debugf("getLlbs: llbs: %d", cnfInfo.Pods["llb"]["total"])
	var llbs int
	if t, ok := cnfInfo.Pods["llb"]["total"]; ok {
		switch v := t.(type) {
		case int:
			llbs = v
			if v > 6 {
				llbs = 6
			}
		}
	} else {
		llbs = 6
	}
	return llbs
}

func getLMGs(cnfInfo *CnfInfo, k *int) (int, int) {
	// allocate LMG loopbacks and LMG group loopback
	log.Debugf("getLmgs: lmgs: %d k: %d", cnfInfo.Pods["lmg"]["total"], *k)
	var lmgs int
	if t, ok := cnfInfo.Pods["lmg"]["total"]; ok {
		switch v := t.(type) {
		case int:
			lmgs = v - *k
			// control max LMGs
			if lmgs >= 16 {
				lmgs = 16 - *k
			}
		}
	} else {
		lmgs = 16 - *k
	}

	// indicate the amount of lmgPodsPerGroup
	var lmgPodsPerGroup int
	switch *cnfInfo.Deployment {
	case "ntok":
		lmgPodsPerGroup = 1
	case "1to1":
		lmgPodsPerGroup = 2
	}
	log.Debugf("getLmgs: lmgs: %d lmgPodsPerGroup: %d", lmgs, lmgPodsPerGroup)
	return lmgs, lmgPodsPerGroup
}

func AllocateIPIndex(app, usage *string, index *int, ipNet *net.IPNet, ipam map[string]*IpamApp) (*AllocatedIPInfo, error) {
	if _, ok := ipam[ipNet.String()]; !ok {
		log.Debugf("AllocateIPIndex: new cidr: %s", ipNet.String())
		ipam[ipNet.String()] = new(IpamApp)
	}

	ip, err := cidr.Host(ipNet, *index)
	if err != nil {

		return nil, err
	}
	log.Debugf("AllocateIPIndex ip: %s", ip)
	allocateIP := &AllocatedIPInfo{
		Application: app,
		Usage:       usage,
		IPAddress:   StringPtr(ip.String()),
	}

	found := false
	for _, allocIP := range ipam[ipNet.String()].AllocatedIPs {
		if *allocIP.IPAddress == *allocateIP.IPAddress {
			found = true
		}
	}
	if !found {
		ipam[ipNet.String()].AllocatedIPs = append(ipam[ipNet.String()].AllocatedIPs, allocateIP)
	}

	return allocateIP, nil
}

func AllocateAppIndex(app, lpName *string, index *int, ipv4Net, ipv6Net *net.IPNet, ipam map[string]*IpamApp) (*string, *string, error) {
	log.Debugf("Allocate app Index: %s, %s, %v, %v", *app, *lpName, *ipv4Net, *ipv6Net)
	// Ipv4
	ipv4, err := cidr.Host(ipv4Net, *index)
	if err != nil {
		return nil, nil, err
	}

	allocateIPv4 := &AllocatedIPInfo{
		Application: StringPtr(*app),
		Usage:       StringPtr("Application loopback " + *lpName),
		IPAddress:   StringPtr(ipv4.String()),
	}

	found := false
	for _, allocIP := range ipam[ipv4Net.String()].AllocatedIPs {
		if *allocIP.IPAddress == *allocateIPv4.IPAddress {
			found = true
		}
	}
	if !found {
		ipam[ipv4Net.String()].AllocatedIPs = append(ipam[ipv4Net.String()].AllocatedIPs, allocateIPv4)
	}

	// Ipv6
	ipv6, err := cidr.Host(ipv6Net, *index)
	if err != nil {
		return nil, nil, err
	}

	allocateIPv6 := &AllocatedIPInfo{
		Application: StringPtr("amf"),
		Usage:       StringPtr("Application loopback " + *lpName),
		IPAddress:   StringPtr(ipv6.String()),
	}

	found = false
	for _, allocIP := range ipam[ipv6Net.String()].AllocatedIPs {
		if *allocIP.IPAddress == *allocateIPv6.IPAddress {
			found = true
		}
	}
	if !found {
		ipam[ipv6Net.String()].AllocatedIPs = append(ipam[ipv6Net.String()].AllocatedIPs, allocateIPv6)
	}

	return StringPtr(ipv4.String()), StringPtr(ipv6.String()), nil
}

func (appIPMap AppIPMap) AllocateAppLoopback(app, lpName, wlName *string, index *int, ipv4Net, ipv6Net *net.IPNet, ipam map[string]*IpamApp) (err error) {

	switch *lpName {
	case "n2":
		appIPMap.IPinfo[*wlName].N2Ipv4, appIPMap.IPinfo[*wlName].N2Ipv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "n8":
		appIPMap.IPinfo[*wlName].N8Ipv4, appIPMap.IPinfo[*wlName].N8Ipv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "n11":
		appIPMap.IPinfo[*wlName].N11Ipv4, appIPMap.IPinfo[*wlName].N11Ipv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "n12":
		appIPMap.IPinfo[*wlName].N12Ipv4, appIPMap.IPinfo[*wlName].N12Ipv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "n14":
		appIPMap.IPinfo[*wlName].N14Ipv4, appIPMap.IPinfo[*wlName].N14Ipv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "n15":
		appIPMap.IPinfo[*wlName].N15Ipv4, appIPMap.IPinfo[*wlName].N15Ipv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "n17":
		appIPMap.IPinfo[*wlName].N17Ipv4, appIPMap.IPinfo[*wlName].N17Ipv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "n22":
		appIPMap.IPinfo[*wlName].N22Ipv4, appIPMap.IPinfo[*wlName].N22Ipv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "n20":
		appIPMap.IPinfo[*wlName].N20Ipv4, appIPMap.IPinfo[*wlName].N20Ipv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "n26":
		appIPMap.IPinfo[*wlName].N26Ipv4, appIPMap.IPinfo[*wlName].N26Ipv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "nnrf":
		appIPMap.IPinfo[*wlName].NnrfIpv4, appIPMap.IPinfo[*wlName].NnrfIpv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "nsms":
		appIPMap.IPinfo[*wlName].NsmsIPv4, appIPMap.IPinfo[*wlName].NsmsIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "amfSvcDefaultIp":
		appIPMap.IPinfo[*wlName].AmfSvcDefaultIPv4, appIPMap.IPinfo[*wlName].AmfSvcDefaultIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "amfSvcLocIp":
		appIPMap.IPinfo[*wlName].AmfSvcLocIPv4, appIPMap.IPinfo[*wlName].AmfSvcLocIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "amfSvcComIp":
		appIPMap.IPinfo[*wlName].AmfSvcComIPv4, appIPMap.IPinfo[*wlName].AmfSvcComIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "amfSvcEeIp":
		appIPMap.IPinfo[*wlName].AmfSvcEeIPv4, appIPMap.IPinfo[*wlName].AmfSvcEeIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "amfSvcMtIp":
		appIPMap.IPinfo[*wlName].AmfSvcMtIPv4, appIPMap.IPinfo[*wlName].AmfSvcMtIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "nfyEirIp":
		appIPMap.IPinfo[*wlName].NfyEirIPv4, appIPMap.IPinfo[*wlName].NfyEirIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "nfyAmfIp":
		appIPMap.IPinfo[*wlName].NfyAmfIPv4, appIPMap.IPinfo[*wlName].NfyAmfIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "nfyAusfIp":
		appIPMap.IPinfo[*wlName].NfyAusfIPv4, appIPMap.IPinfo[*wlName].NfyAusfIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "nfyNrfIp":
		appIPMap.IPinfo[*wlName].NfyNrfIPv4, appIPMap.IPinfo[*wlName].NfyNrfIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "nfyNssfIp":
		appIPMap.IPinfo[*wlName].NfyNssfIPv4, appIPMap.IPinfo[*wlName].NfyNssfIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "nfyPcfIp":
		appIPMap.IPinfo[*wlName].NfyPcfIPv4, appIPMap.IPinfo[*wlName].NfyPcfIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "nfySmfIp":
		appIPMap.IPinfo[*wlName].NfySmfIPv4, appIPMap.IPinfo[*wlName].NfySmfIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "nfyudmIp":
		appIPMap.IPinfo[*wlName].NfyUdmIPv4, appIPMap.IPinfo[*wlName].NfyUdmIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "dns1":
		appIPMap.IPinfo[*wlName].DnsIpdsIPv41, appIPMap.IPinfo[*wlName].DnsIpdsIPv61, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "dns2":
		appIPMap.IPinfo[*wlName].DnsIpdsIPv42, appIPMap.IPinfo[*wlName].DnsIpdsIPv62, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "ausf":
		appIPMap.IPinfo[*wlName].AusfIPv4, appIPMap.IPinfo[*wlName].AusfIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	case "udm":
		appIPMap.IPinfo[*wlName].UdmIPv4, appIPMap.IPinfo[*wlName].UdmIPv6, err = AllocateAppIndex(app, lpName, index, ipv4Net, ipv6Net, ipam)
		if err != nil {
			return err
		}
	}
	return nil
}
