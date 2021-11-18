package parser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	srlDefaultType    = "ixr6"
	vrsrosDefaultType = "sr-1"
	defaultPosition   = "network"
)

// Config defines lab configuration as it is provided in the YAML file
type Config struct {
	Name              *string                               `yaml:"name,omitempty"`
	Credentials       *Credentials                          `yaml:"credentials,omitempty"`
	Topology          *Topology                             `yaml:"topology,omitempty"`
	Cluster           *Cluster                              `yaml:"cluster,omitempty"`
	ContainerRegistry *ContainerRegistry                    `yaml:"container_registry,omitempty"`
	Infrastructure    *Infrastructure                       `yaml:"infrastructure,omitempty"`
	Workloads         map[string]map[string]*WorkloadInfo   `yaml:"workloads,omitempty"`
	Application       map[string]*PacoInfo                  `yaml:"application,omitempty"`
	AppNetworkIndexes map[string]map[string]map[string]*int `yaml:"appnetwindexes,omitempty"`
}

// PacoInfo
type PacoInfo struct {
	Global     *GlobalParameters   `yaml:"global,omitempty"`
	Deployment *PacoDeploymentInfo `yaml:"deployment,omitempty"`
	Cnfs       map[string]*CnfInfo `yaml:"cnfs,omitempty"`
}

type GlobalParameters struct {
	Multus map[string]*MultusInfo `yaml:"multus,omitempty"`
}

// PacoDeployemntInfo
type PacoDeploymentInfo struct {
	ConnectivityMode *string               `yaml:"connectivitymode,omitempty"`
	NetworkName      *string               `yaml:"networkname,omitempty"`
	NetworkShortName *string               `yaml:"networkshortname,omitempty"`
	Nat              *bool                 `yaml:"nat,omitempty"`
	SigRefPoints     *string               `yaml:"sigrefpoints,omitempty"`
	UePoolCidr       *string               `yaml:"uepoolcidr,omitempty"`
	Supi             [][]*string           `yaml:"supi,omitempty"`
	Plmn             map[string]*int       `yaml:"plmn,omitempty"`
	Dnn              []*string             `yaml:"dnn,omitempty"`
	TrackingArea     []*string             `yaml:"tac,omitempty"`
	Slices           map[string]*SliceInfo `yaml:"slices,omitempty"`
	Apn              *string               `yaml:"apn,omitempty"`
}

// SliceInfo
type SliceInfo struct {
	Value int                 `yaml:"value,omitempty"`
	Diff  []map[string]string `yaml:"diff,omitempty"`
}

// CnfInfo
type CnfInfo struct {
	Enabled      *bool                             `yaml:"enabled,omitempty"`
	Deployment   *string                           `yaml:"deployment,omitempty"`
	K            *int                              `yaml:"k,omitempty"`
	NameSpace    *string                           `yaml:"namespace,omitempty"`
	StorageClass *string                           `yaml:"storage_class,omitempty"`
	PrometheusIP *string                           `yaml:"prometheus_ip,omitempty"`
	HostDevice   *string                           `yaml:"host_device,omitempty"`
	Networking   *PacoNetworkInfo                  `yaml:"networking,omitempty"`
	Pods         map[string]map[string]interface{} `yaml:"pods,omitempty"`
}

type PacoNetworkInfo struct {
	Type   *string                `yaml:"type,omitempty"`
	AS     *uint32                `yaml:"as,omitempty"`
	Multus map[string]*MultusInfo `yaml:"multus,omitempty"`
}

type MultusInfo struct {
	WorkloadName *string `yaml:"wl-name,omitempty"`
	VrfCpId      *int    `yaml:"vrfcp-id,omitempty"`
	VrfUpId      *int    `yaml:"vrfup-id,omitempty"`
}

// Credentials
type Credentials struct {
	AnsibleUser              *string `yaml:"ansible_user,omitempty"`
	AnsibleSshPrivateKeyFile *string `yaml:"ansible_ssh_private_key_file,omitempty"`
	AnsibleSshExtraArgs      *string `yaml:"ansible_ssh_extra_args,omitempty"`
}

// Cluster
type Cluster struct {
	ProjectId     *string                 `yaml:"project_id,omitempty"`
	AnthosDeploymentType     *string                 `yaml:"anthos_deployment_type,omitempty"`
	ClusterName   *string                 `yaml:"cluster_name,omitempty"`
	Networks      map[string]*NetworkInfo `yaml:"networks,omitempty"`
	Kind          *string                 `yaml:"kind,omitempty"`
	Region        *string                 `yaml:"region,omitempty"`
	AnthosDir     *string                 `yaml:"anthos_dir,omitempty"`
	AnthosVersion *string                 `yaml:"anthos_version,omitempty"`
}

// ContainerRegistry
type ContainerRegistry struct {
	Kind     *string `yaml:"kind,omitempty"`
	Name     *string `yaml:"name,omitempty"`
	Url      *string `yaml:"url,omitempty"`
	Server   *string `yaml:"server,omitempty"`
	ImageDir *string `yaml:"image_dir,omitempty"`
	Email    *string `yaml:"email,omitempty"`
	Username *string `yaml:"username,omitempty"`
	Secret   *string `yaml:"secret,omitempty"`
}

// Infrastructure
type Infrastructure struct {
	UseVec			 bool					 `default:"false" yaml:"use_vec"`
	UseVecCni		 bool					 `default:"false" yaml:"use_vec_cni"`
	InternetDns      *string                 `yaml:"internet_dns,omitempty"`
	Protocols        *Protocols              `yaml:"protocols,omitempty"`
	AddressingSchema *string                 `yaml:"addressing_schema,omitempty"`
	Networks         map[string]*NetworkInfo `yaml:"networks,omitempty"`
}

// NetworkInfo
type NetworkInfo struct {
	Ipv4Cidr              []*string `yaml:"ipv4_cidr,omitempty"`
	Ipv4ItfcePrefixLength *int      `yaml:"ipv4_itfce_prefix_length,omitempty"`
	Ipv6Cidr              []*string `yaml:"ipv6_cidr,omitempty"`
	Ipv6ItfcePrefixLength *int      `yaml:"ipv6_itfce_prefix_length,omitempty"`
	AddressingSchema      *string   `yaml:"addressing_schema,omitempty"`
	PeerAS                *uint32   `yaml:"peer_as,omitempty"`
	Type                  *string   `yaml:"type,omitempty"`
	VlanID                *int      `yaml:"vlan_id,omitempty"`
	Idx                   *int      `yaml:"idx,omitempty"`
	Kind                  *string   `yaml:"kind,omitempty"`
	Target                *string   `yaml:"target,omitempty"`
	NetworkIndex          *int
	SwitchIndex           *int

	TemplateKind       string
	Ipv4ItfceAddresses []*AllocatedIPInfo
	Ipv6ItfceAddresses []*AllocatedIPInfo
	//Ipv4UpItfceAddresses  map[int][]*AllocatedIPInfo // key is the group
	//Ipv6UpItfceAddresses  map[int][]*AllocatedIPInfo // key is the group
	Ipv4CpAddresses       []*AllocatedIPInfo
	Ipv6CpAddresses       []*AllocatedIPInfo
	Ipv4UpAddresses       map[int][]*AllocatedIPInfo // key is the group
	Ipv6UpAddresses       map[int][]*AllocatedIPInfo // key is the group
	Ipv4FloatingIP        *string
	Ipv6FloatingIP        *string
	Ipv4Gw                *string
	Ipv6Gw                *string
	Ipv4GwPerWl           map[int][]string
	Ipv6GwPerWl           map[int][]string
	InterfaceName         *string
	InterfaceEthernetName *string
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

type BGPPeerInfo struct {
	IP   *string
	AS   *uint32
	Node string
}

// Protocols
type Protocols struct {
	Protocol        *string   `yaml:"protocol,omitempty"`
	AsPool          []*uint32 `yaml:"as_pool,omitempty"`
	AsPoolLoop      []*uint32 `yaml:"as_pool_loop,omitempty"`
	OverlayAs       *uint32   `yaml:"overlay_as,omitempty"`
	OverlayProtocol *string   `yaml:"overlay_protocol,omitempty"`
}

// WorkloadInfo
type WorkloadInfo struct {
	Itfces    map[string]*NetworkInfo `yaml:"itfces,omitempty"`
	Loopbacks map[string]*NetworkInfo `yaml:"loopbacks,omitempty"`
}

// Topology represents a lab topology
type Topology struct {
	Defaults *NodeConfig            `yaml:"defaults,omitempty"`
	Kinds    map[string]*NodeConfig `yaml:"kinds,omitempty"`
	Nodes    map[string]*NodeConfig `yaml:"nodes,omitempty"`
	Links    []*LinkConfig          `yaml:"links,omitempty"`
}

// NodeConfig represents a configuration a given node can have in the lab definition file
type NodeConfig struct {
	Kind     *string            `yaml:"kind,omitempty"`   // srl, vr-sros, linux
	Labels   map[string]*string `yaml:"labels,omitempty"` // Labels are attributes
	Group    *string            `yaml:"group,omitempty"`
	Type     *string            `yaml:"type,omitempty"` // ixrd2, sr-1s, etc
	Position *string            `yaml:"position,omitempty"`
	MgmtIPv4 *string            `yaml:"mgmt_ipv4,omitempty"` // user-defined IPv4 address in the management network
	MgmtIPv6 *string            `yaml:"mgmt_ipv6,omitempty"` // user-defined IPv6 address in the management network
	AS       *uint32            `yaml:"as,omitempty"`
}

type LinkConfig struct {
	Endpoints []*string
	Labels    map[string]*string `yaml:"labels,omitempty"`
}

// Node is a struct that contains the information of a node element
type Node struct {
	ShortName            *string
	Kind                 *string
	Type                 *string
	Labels               map[string]*string
	Group                *string
	Position             *string
	Topology             *string
	MgmtIPv4Address      *string
	MgmtIPv4PrefixLength *int
	MgmtIPv6Address      *string
	MgmtIPv6PrefixLength *int
	AS                   *uint32
	Endpoints            map[string]*Endpoint
	Target               *string
}

func (n *Node) String() string {
	sb := strings.Builder{}
	if n.ShortName != nil {
		sb.WriteString(fmt.Sprintf("Name: %s\n", n.ShortName))
	}
	return sb.String()
}

// Link is a struct that contains the information of a link between 2 containers
type Link struct {
	A             *Endpoint
	B             *Endpoint
	MTU           *int
	Labels        map[string]*string
	vWire         *bool   // tbd for what this is used
	Kind          *string // isl: inter switch links, access: links to clients
	VlanTagging   *bool   // VLANid, used only for isl links
	VlanID        *string // VLANid, used only for isl links
	Lag           *bool   // inidcation if the link is part of a LAG
	LagMemberLink *bool   // indication if a link is a member link of a LAG; e.g. ethernet-1/1 is a memer of lag1
	LagName       *string // name of a lag, clientname is used on linux
	ClientName    *string // used in linux as the lag name
	Numa          *int    // used to describe the numa
	Sriov         *bool   // inidcation if the link is sriov enabled
	IPVlan        *bool   // inidcation if the link is ipvlan enabled
	Speed         *string // link speed
	Pxe           *bool   // used to drive lacp-fallback
}

// Endpoint is a struct that contains information of a link endpoint
type Endpoint struct {
	Node                *Node
	PeerNode            *Node
	ShortName           *string // e1-1
	RealName            *string // ethernet-1/1
	Kind                *string // loopback, isl, access
	IPv4Prefix          *string
	IPv4NeighborPrefix  *string
	IPv4Address         *string
	IPv4NeighborAddress *string
	IPv4PrefixLength    *int
	IPv6Prefix          *string
	IPv6NeighborPrefix  *string
	IPv6Address         *string
	IPv6NeighborAddress *string
	IPv6PrefixLength    *int
	PeerAS              *uint32
	VlanTagging         *bool   // VLANid, used only for isl links
	VlanID              *string // VLANid, used only for isl links
	Lag                 *bool   // inidcation if the link is part of a LAG
	LagMemberLink       *bool   // indication if a link is a member link of a LAG; e.g. ethernet-1/1 is a memer of lag1
	LagName             *string // name of a lag, clientname is used on linux
	Sriov               *bool   // inidcation if the link is sriov enabled
	IPVlan              *bool   // inidcation if the link is ipvlan enabled
	Speed               *string // link speed
	Pxe                 *bool   // used to drive lacp-fallback
}

type Workload struct {
	NetworkInstance map[string]*NetworkInstance
}

type NetworkInstance struct {
	Kind           *string // irb, bridged, routed
	IPv4Cidr       *string
	IPv6Cidr       *string
	VlanId         *int
	RouteTarget    *string
	Interfaces     []string
	VxlanInterface string
}

type ClientGroup struct {
	TargetGroup *string
	Interfaces  map[string][]*InterfaceDetails // string index contains the leaf switch name
}

type InterfaceDetails struct {
	Endpoint *Endpoint
}

// ParseTopology parses the topology part of the configuration file
func (p *Parser) ParseTopology() (err error) {
	log.Info("Parsing topology information ...")

	// initialize the Node information from the topology map

	names := make([]string, 0, len(p.Config.Topology.Nodes))
	for n := range p.Config.Topology.Nodes {
		names = append(names, n)
	}

	sort.Strings(names)
	for _, name := range names {
		p.Nodes[name], err = p.NewNode(name, p.Config.Topology.Nodes[name])
		if err != nil {
			return err
		}
	}
	for _, l := range p.Config.Topology.Links {
		if err = p.NewLink(l); err != nil {
			return err
		}
	}
	return nil
}

// initialize Kind
func (p *Parser) kindInitialization(nodeCfg *NodeConfig) *string {
	if nodeCfg.Kind != nil {
		if *nodeCfg.Kind != "" {
			return nodeCfg.Kind
		}
	}
	return p.Config.Topology.Defaults.Kind
}

// initialize Type
func (p *Parser) typeInitialization(nodeCfg *NodeConfig, kind *string) *string {
	if nodeCfg.Type != nil {
		if *nodeCfg.Kind != "" {
			return nodeCfg.Kind
		}
	}
	if _, ok := p.Config.Topology.Kinds[*kind]; ok {
		if p.Config.Topology.Kinds[*kind].Type != nil {
			if *p.Config.Topology.Kinds[*kind].Type != "" {
				return p.Config.Topology.Kinds[*kind].Type
			}
		}
	}

	if p.Config.Topology.Defaults != nil {
		if p.Config.Topology.Defaults.Type != nil {
			if *p.Config.Topology.Defaults.Type != "" {
				return p.Config.Topology.Defaults.Type
			}
		}
	}

	// default type if not defined
	switch *kind {
	case "srl":
		return StringPtr(srlDefaultType)
	case "vr-sros":
		return StringPtr(vrsrosDefaultType)
	}
	return StringPtr("")
}

// initialize Position
func (p *Parser) positionInitialization(nodeCfg *NodeConfig, kind *string) *string {
	if nodeCfg.Position != nil {
		if *nodeCfg.Position != "" {
			return nodeCfg.Position
		}
	}
	if _, ok := p.Config.Topology.Kinds[*kind]; ok {
		if p.Config.Topology.Kinds[*kind].Position != nil {
			if *p.Config.Topology.Kinds[*kind].Position != "" {
				return p.Config.Topology.Kinds[*kind].Position
			}
		}
	}

	if p.Config.Topology.Defaults != nil {
		if p.Config.Topology.Defaults.Position != nil {
			if *p.Config.Topology.Defaults.Position != "" {
				return p.Config.Topology.Defaults.Position
			}
		}
	}
	// default type if not defined
	switch *kind {
	case "srl":
		return StringPtr(defaultPosition)
	case "vr-sros":
		return StringPtr(defaultPosition)
	}
	return StringPtr("")
}

// NewNode initializes a new node object
func (p *Parser) NewNode(nodeName string, nodeCfg *NodeConfig) (*Node, error) {
	// initialize a new node
	node := &Node{
		ShortName:            new(string),
		Kind:                 new(string),
		Type:                 new(string),
		Labels:               make(map[string]*string),
		Group:                new(string),
		Position:             new(string),
		Topology:             new(string),
		MgmtIPv4Address:      new(string),
		MgmtIPv4PrefixLength: new(int),
		MgmtIPv6Address:      new(string),
		MgmtIPv6PrefixLength: new(int),
		AS:                   new(uint32),
		Endpoints:            make(map[string]*Endpoint),
		Target:               new(string),
	}
	*node.ShortName = nodeName
	node.MgmtIPv4Address = nodeCfg.MgmtIPv4
	node.MgmtIPv6Address = nodeCfg.MgmtIPv6
	node.Labels = nodeCfg.Labels
	if nodeCfg.AS != nil {
		*node.AS = *nodeCfg.AS
	}

	// initialize kind, based on hierarchical information in the config file
	// most specific information is selected
	node.Kind = StringPtr(strings.ToLower(*p.kindInitialization(nodeCfg)))

	// initialize type, based on hierarchical information in the config file
	// most specific information is selected
	node.Type = StringPtr(strings.ToLower(*p.typeInitialization(nodeCfg, node.Kind)))

	// initialize position, based on hierarchical information in the config file
	// most specific information is selected
	node.Position = StringPtr(strings.ToLower(*p.positionInitialization(nodeCfg, node.Kind)))

	// Initialize the Interfaces map
	node.Endpoints = make(map[string]*Endpoint)

	// initialize target
	node.Target = StringPtr("")
	if _, ok := node.Labels["target"]; ok {
		node.Target = node.Labels["target"]
	}

	if p.Config.Infrastructure != nil {
		if *node.Position == "network" {
			// Allocate the node loopback address
			ipEP, err := p.IPAM["loopback"].IPAMAllocateAddress(StringPtr("loopback"), p.Config.Infrastructure.Networks["loopback"].Ipv4Cidr, p.Config.Infrastructure.Networks["loopback"].Ipv6Cidr)
			if err != nil {
				return nil, err
			}
			node.Endpoints["lo0"] = new(Endpoint)
			node.Endpoints["lo0"].ShortName = StringPtr("lo0")
			node.Endpoints["lo0"].RealName = StringPtr("lo0")
			node.Endpoints["lo0"].Kind = StringPtr("loopback")
			node.Endpoints["lo0"].IPv4Prefix = ipEP.IPv4Prefix
			node.Endpoints["lo0"].IPv4Address = ipEP.IPv4Address
			node.Endpoints["lo0"].IPv4PrefixLength = ipEP.IPv4PrefixLength
			node.Endpoints["lo0"].IPv6Prefix = ipEP.IPv6Prefix
			node.Endpoints["lo0"].IPv6Address = ipEP.IPv6Address
			node.Endpoints["lo0"].IPv6PrefixLength = ipEP.IPv6PrefixLength
			//log.Infof("IP loopback allocation on node: %s, %v", node.ShortName, node.Interfaces["lo1"])

			log.Debugf("Node Loopback: %s, ipv4: %s, ipv6: %s", *node.ShortName, *node.Endpoints["lo0"].IPv4Prefix, *node.Endpoints["lo0"].IPv6Prefix)
			log.Debugf("Position: %s %d", node.Position, p.NextAS)

			// dont apply the AS auto-config if the AS is supplied by config
			if nodeCfg.AS == nil {
				switch *p.Config.Infrastructure.Protocols.Protocol {
				case "ebgp":
					// update the AS from the original parser
					*node.AS = *p.NextAS
					*p.NextAS++
				default:
					// update the AS from the original parser
					*node.AS = *p.NextAS
				}
			}
		}
	}
	return node, nil
}

// NewLink initializes a new link object
func (p *Parser) NewLink(l *LinkConfig) error {
	// initialize a new link
	link := &Link{
		A:             new(Endpoint),
		B:             new(Endpoint),
		MTU:           new(int),
		Labels:        make(map[string]*string),
		vWire:         new(bool),
		Kind:          new(string),
		VlanTagging:   new(bool),
		VlanID:        new(string),
		Lag:           new(bool),
		LagMemberLink: new(bool),
		LagName:       new(string),
		ClientName:    new(string),
		Numa:          new(int),
		Sriov:         new(bool),
		IPVlan:        new(bool),
		Speed:         new(string),
	}

	// initalize endpointdata for the link
	var nodeShortNameA *string
	var epShortNameA *string
	var nodeShortNameB *string
	var epShortNameB *string

	for i, d := range l.Endpoints {
		// i indicates the number and d presents the string, which need to be
		// split in node and endpoint name
		// split the string to get node name and endpoint name
		split := strings.Split(*d, ":")
		if len(split) != 2 {
			log.Fatal(fmt.Sprintf("endpoint %s has wrong syntax", *d))
		}
		if i == 0 {
			nodeShortNameA = &split[0]
			epShortNameA = &split[1]
		} else {
			nodeShortNameB = &split[0]
			epShortNameB = &split[1]
		}
	}
	// initialize nodeA and nodeB
	nodeA := p.Nodes[*nodeShortNameA]
	nodeB := p.Nodes[*nodeShortNameB]

	// initialize the label parameters from the config
	link.Labels = l.Labels
	// Kind is either a backbone facing or customer facing interface
	// isl is an inter switch link, access is a client facing link
	// initialize default -> isl
	link.Kind = StringPtr("isl")
	if _, ok := link.Labels["kind"]; ok {
		link.Kind = link.Labels["kind"]
	}

	// Sriov initialization
	// initialize default -> false
	link.Sriov = BoolPtr(false)
	if _, ok := link.Labels["sriov"]; ok {
		if *link.Labels["sriov"] == "true" {
			link.Sriov = BoolPtr(true)
		}
	}

	// IPVlan initialization
	// initialize default -> false
	link.IPVlan = BoolPtr(false)
	if _, ok := link.Labels["ipvlan"]; ok {
		if *link.Labels["ipvlan"] == "true" {
			link.IPVlan = BoolPtr(true)
		}
	}

	// Speed initialization
	// initialize default -> false
	link.Speed = StringPtr("100G")
	if _, ok := link.Labels["speed"]; ok {
		link.Speed = link.Labels["speed"]
	}

	// PXE initialization
	// initialize default -> false
	link.Pxe = BoolPtr(false)
	if _, ok := link.Labels["pxe"]; ok {
		if *link.Labels["pxe"] == "true" {
			link.Pxe = BoolPtr(true)
		}
	}

	// initialize VLAN defaults -> untagged
	link.VlanTagging = BoolPtr(false)
	link.VlanID = StringPtr("0")
	// initialize VLAN: Vlan is a string which is 0 for untagged and > 0 for tagged
	if _, exists := link.Labels["vlan"]; exists {
		if *link.Labels["vlan"] != "0" {
			link.VlanTagging = BoolPtr(true)
			link.VlanID = link.Labels["vlan"]
		}
	}

	// initialize LAG defaults -> false
	link.Lag = BoolPtr(false)
	link.LagName = StringPtr("")
	// Process links in a LAG
	if _, ok := link.Labels["type"]; ok {
		split := strings.Split(*link.Labels["type"], ":")
		if len(split) != 1 {
			log.Fatal(fmt.Sprintf("Link label type %s has wrong syntax", *link.Labels["type"]))
		}
		if strings.Contains(split[0], "lag") || strings.Contains(split[0], "esi") {
			link.Lag = BoolPtr(true)
			link.LagName = &split[0]
		}
	}

	// initialize ClientName defaults -> ""
	// used in LAG as an override capability to name the linux clients based on this name
	link.ClientName = StringPtr("")
	if _, ok := link.Labels["client-name"]; ok {
		link.ClientName = link.Labels["client-name"]
	}

	// initialize Numa defaults -> 0
	// used in App to define the connectivity and numa definiton
	link.Numa = IntPtr(0)
	if _, ok := link.Labels["numa"]; ok {
		n, err := strconv.Atoi(*link.Labels["numa"])
		if err != nil {
			log.WithError(err).Errorf("issue converting numa into string")
		}
		link.Numa = IntPtr(n)
	}

	// When a link is a lag we have to expand the links to represent the LAG
	if *link.Lag {
		found := false
		// check if the lag was already created
		for nodeName, n := range p.Nodes {
			if nodeName == *nodeShortNameA || nodeName == *nodeShortNameB {
				for _, ep := range n.Endpoints {
					if ep.LagName == link.LagName {
						found = true
					}
				}
			}
		}
		// if lag is not found, we have to expand the interface with a new LAG object
		if !found {
			lag := new(Link)
			lag.vWire = BoolPtr(false)
			lag.Lag = BoolPtr(true)
			lag.LagMemberLink = BoolPtr(false)
			lag.LagName = link.LagName
			lag.Kind = link.Kind
			lag.VlanTagging = link.VlanTagging
			lag.VlanID = link.VlanID
			lag.Speed = link.Speed
			lag.Pxe = link.Pxe
			if *nodeA.Kind == "linux" {
				// override the lag name with the client-name supplied in the labels field
				lag.A = p.NewEndpoint(nodeShortNameA, link.ClientName, lag, nodeB)
			} else {
				lag.A = p.NewEndpoint(nodeShortNameA, lag.LagName, lag, nodeB)
			}
			if *nodeB.Kind == "linux" {
				log.Debugf("New Endpoint: %s", *link.ClientName)
				// override the lag name with the client-name supplied in the labels field
				lag.B = p.NewEndpoint(nodeShortNameB, link.ClientName, lag, nodeA)
			} else {
				lag.B = p.NewEndpoint(nodeShortNameB, lag.LagName, lag, nodeA)
			}

			p.Nodes[*nodeShortNameA].Endpoints[*lag.LagName] = lag.A
			p.Nodes[*nodeShortNameB].Endpoints[*lag.LagName] = lag.B
			p.Links = append(p.Links, lag)
			// Allocate IP addresses on the link if the link is of kind isl
			if *lag.Kind == "isl" {
				for _, ipv4Cidr := range p.Config.Infrastructure.Networks["isl"].Ipv4Cidr {
					for _, ipv6Cidr := range p.Config.Infrastructure.Networks["isl"].Ipv6Cidr {
						if err := p.IPAM["isl"].IPAMAllocateLinkPrefix(lag, ipv4Cidr, ipv6Cidr); err != nil {
							log.Error(err)
						}
					}
				}
			}
		}
	}

	// initialize a new link and add link to the clabCOnfig Links
	if *link.Lag {
		link.LagMemberLink = BoolPtr(true) // this is used to distinguish between a meber link of a LAG and a real LAG
		link.Lag = BoolPtr(false)
	}
	if *link.LagMemberLink {
		*link.LagName = strings.ReplaceAll(*link.LagName, "esi", "lag")
	}
	//link.LagNameA = lagID
	link.A = p.NewEndpoint(nodeShortNameA, epShortNameA, link, nodeB)
	link.B = p.NewEndpoint(nodeShortNameB, epShortNameB, link, nodeA)
	p.Nodes[*nodeShortNameA].Endpoints[*epShortNameA] = link.A
	p.Nodes[*nodeShortNameB].Endpoints[*epShortNameB] = link.B
	link.vWire = BoolPtr(true)
	p.Links = append(p.Links, link)
	// Allocate IP addresses on the link
	// only allocate IP link prefixes for LAG link bundle and links not part of a lag bundle
	if link.LagMemberLink != nil && !*link.LagMemberLink && *link.Kind == "isl" {
		for _, ipv4Cidr := range p.Config.Infrastructure.Networks["isl"].Ipv4Cidr {
			for _, ipv6Cidr := range p.Config.Infrastructure.Networks["isl"].Ipv6Cidr {
				if err := p.IPAM["isl"].IPAMAllocateLinkPrefix(link, ipv4Cidr, ipv6Cidr); err != nil {
					log.Error(err)
				}
			}
		}
	}
	return nil
}

// NewEndpoint initializes a new endpoint object
func (p *Parser) NewEndpoint(nodeShortName, epShortName *string, l *Link, peerNode *Node) *Endpoint {
	// initialize a new endpoint
	ep := &Endpoint{
		Node:                new(Node),
		PeerNode:            new(Node),
		ShortName:           new(string),
		RealName:            new(string),
		Kind:                new(string),
		IPv4Prefix:          new(string),
		IPv4NeighborPrefix:  new(string),
		IPv4Address:         new(string),
		IPv4NeighborAddress: new(string),
		IPv4PrefixLength:    new(int),
		IPv6Prefix:          new(string),
		IPv6NeighborPrefix:  new(string),
		IPv6Address:         new(string),
		IPv6NeighborAddress: new(string),
		IPv6PrefixLength:    new(int),
		PeerAS:              new(uint32),
		VlanTagging:         new(bool),
		VlanID:              new(string),
		Lag:                 new(bool),
		LagMemberLink:       new(bool),
		LagName:             new(string),
		Sriov:               new(bool),
		IPVlan:              new(bool),
		Speed:               new(string),
		Pxe:                 new(bool),
	}

	ep.PeerNode = peerNode
	log.Debugf("NewEndpoint Node Short Name: %s", *nodeShortName)
	//ep.Node = p.Nodes[*nodeShortName]
	*ep.Kind = *l.Kind

	// search the node pointer based in the name of the split function
	found := false
	for name, n := range p.Nodes {
		if name == *nodeShortName {
			ep.Node = n
			found = true
		}
	}
	if !found {
		log.Fatalf("Not all nodes are specified in the duts section or the names don't match in the duts/endpoint section: %s", nodeShortName)
	}

	// initialize the endpoint name

	*ep.ShortName = *epShortName
	*ep.RealName = *epShortName
	if !*l.Lag {
		if *ep.Node.Kind == "srl" {
			*ep.RealName = strings.ReplaceAll(*epShortName, "-", "/")
			*ep.RealName = strings.ReplaceAll(*ep.RealName, "e", "ethernet-")
		}
	} else {
		if *ep.Node.Kind == "srl" {
			*ep.RealName = strings.ReplaceAll(*epShortName, "esi", "lag")
		}
	}

	log.Debugf("PeerAS: %d", peerNode.AS)
	*ep.PeerAS = *peerNode.AS
	*ep.VlanTagging = *l.VlanTagging
	*ep.VlanID = *l.VlanID
	*ep.Lag = *l.Lag
	*ep.LagMemberLink = *l.LagMemberLink
	*ep.LagName = *l.LagName
	if l.IPVlan != nil {
		*ep.IPVlan = *l.IPVlan
	}
	if l.Sriov != nil {
		*ep.Sriov = *l.Sriov
	}
	if l.Speed != nil {
		*ep.Speed = *l.Speed
	}
	if l.Pxe != nil {
		*ep.Pxe = *l.Pxe
	}

	return ep
}

// ShowTopology show the topology that was initialized
func (p *Parser) ShowTopology() {
	for nName, n := range p.Nodes {
		if *n.Position == "network" {
			log.Infof("Node Name: %s, Kind: %s, Type: %s, Mgmt IPv4 Address: %s, ShortName: %s, AS: %d", nName, *n.Kind, *n.Type, *n.MgmtIPv4Address, *n.ShortName, *n.AS)
			for epName, ep := range n.Endpoints {
				log.Infof("Endpoint Name: %s, %s, %s", epName, *ep.RealName, *ep.Kind)
				if ep.Lag != nil && *ep.Lag {
					//log.Infof("Endpoint Name: %s, LAG info: %t", epName, *ep.Lag)
					//*ep.LagName
					//*ep.LagMemberLink
				}
				if *ep.Kind == "loopback" {
					log.Infof("IP Info loopback: IPv4 info: %s, IPv6 info: %s", *ep.IPv4Address, *ep.IPv6Address)
				} else {
					log.Infof("IP Info: IPv4 info: %s, %s, IPv6 info: %s, %s", *ep.IPv4Address, *ep.IPv4NeighborAddress, *ep.IPv6Address, *ep.IPv6NeighborAddress)
					log.Infof("IP Info: IPv4 info: %s, %s, IPv6 info: %s, %s", *ep.IPv4Prefix, *ep.IPv4NeighborPrefix, *ep.IPv6Prefix, *ep.IPv6NeighborPrefix)
				}
			}
		}
	}
}

// ParseClientGroup parses the client group part of the configuration file
// which client groups do we have and which device, interfaces are connected
func (p *Parser) ParseClientGroup() (err error) {
	log.Info("Parsing workload information ...")

	// check which client groups are connected
	// cls is a list with all the client groups in the system
	for _, clients := range p.Config.Workloads {
		for cgName := range clients {
			if _, ok := p.ClientGroups[cgName]; !ok {
				p.ClientGroups[cgName] = &ClientGroup{
					TargetGroup: new(string),
					Interfaces:  make(map[string][]*InterfaceDetails, 0), // first string index indicates the leaf anme, InterfaceDetails contain the interface names
				}
			}

			// walks over all links to get the names of the switch interfaces that are connected to the client
			// we initialize the real node name in Interfaces as well as
			for _, l := range p.Links {
				if *l.A.Node.Target == cgName {
					// Also include the member links
					//if !*l.B.LagMemberLink {
					log.Debugf("Enpoint %s %s %s %s", *l.B.Node.ShortName, *l.B.Node.Target, *l.B.ShortName, *l.B.RealName)
					*p.ClientGroups[cgName].TargetGroup = *l.B.Node.Target
					interfaceDetails := &InterfaceDetails{
						Endpoint: l.B,
					}
					// Real Node initialization
					// if the nodeName was not yet in the dictionary, initialize the Interfaces
					if _, ok := p.ClientGroups[cgName].Interfaces[*l.B.Node.ShortName]; !ok {
						p.ClientGroups[cgName].Interfaces[*l.B.Node.ShortName] = make([]*InterfaceDetails, 0)
					}
					// check id the initerface already exists

					p.ClientGroups[cgName].Interfaces[*l.B.Node.ShortName] = append(p.ClientGroups[cgName].Interfaces[*l.B.Node.ShortName], interfaceDetails)

					// Target Node initialization -> allows for node group configuration
					// Here we need to check duplicate interfaces and avoid duplicating the interfaces
					// if the nodeName was not yet in the dictionary, initialize the Interfaces
					if _, ok := p.ClientGroups[cgName].Interfaces[*l.B.Node.ShortName]; !ok {
						p.ClientGroups[cgName].Interfaces[*l.B.Node.ShortName] = make([]*InterfaceDetails, 0)
					}
					// check for duplicate interface information
					found := false
					for _, itfceDetail := range p.ClientGroups[cgName].Interfaces[*l.B.Node.Target] {
						if *itfceDetail.Endpoint.RealName == *l.B.RealName {
							found = true
						}
					}
					if !found {
						p.ClientGroups[cgName].Interfaces[*l.B.Node.Target] = append(p.ClientGroups[cgName].Interfaces[*l.B.Node.Target], interfaceDetails)
					}
					//}
				}
				if *l.B.Node.Target == cgName {
					//if !*l.A.LagMemberLink {
					log.Debugf("Enpoint %s %s %s %s", *l.A.Node.ShortName, *l.A.Node.Target, *l.A.ShortName, *l.A.RealName)
					*p.ClientGroups[cgName].TargetGroup = *l.A.Node.Target
					interfaceDetails := &InterfaceDetails{
						Endpoint: l.A,
					}
					// Real Node initialization
					// if the nodeName was not yet in the dictionary, initialize the Interfaces
					if _, ok := p.ClientGroups[cgName].Interfaces[*l.A.Node.ShortName]; !ok {
						p.ClientGroups[cgName].Interfaces[*l.A.Node.ShortName] = make([]*InterfaceDetails, 0)
					}
					// check for duplicate interface information
					found := false
					for _, itfceDetail := range p.ClientGroups[cgName].Interfaces[*l.A.Node.ShortName] {
						if *itfceDetail.Endpoint.RealName == *l.A.RealName {
							found = true
						}
					}
					if !found {
						p.ClientGroups[cgName].Interfaces[*l.A.Node.ShortName] = append(p.ClientGroups[cgName].Interfaces[*l.A.Node.ShortName], interfaceDetails)
					}

					// Target Node initialization -> allows for node group configuration
					// We need to check duplicate interfaces and avoid duplicating the interfaces
					// if the nodeName was not yet in the dictionary, initialize the Interfaces
					if _, ok := p.ClientGroups[cgName].Interfaces[*l.A.Node.ShortName]; !ok {
						p.ClientGroups[cgName].Interfaces[*l.A.Node.ShortName] = make([]*InterfaceDetails, 0)
					}
					// check for duplicate interface information
					found = false
					for _, itfceDetail := range p.ClientGroups[cgName].Interfaces[*l.A.Node.Target] {
						if *itfceDetail.Endpoint.RealName == *l.A.RealName {
							found = true
						}
					}
					if !found {
						p.ClientGroups[cgName].Interfaces[*l.A.Node.Target] = append(p.ClientGroups[cgName].Interfaces[*l.A.Node.Target], interfaceDetails)
					}
					//}
				}
			}
		}
	}
	return nil
}

func (p *Parser) ShowClientGroup() error {
	for cgName, clients := range p.ClientGroups {
		log.Infof("CgName: %s, TargetGroup: %s", cgName, *clients.TargetGroup)
		for nodeName, itfces := range clients.Interfaces {
			log.Infof("CgName: %s, NodeName: %s", cgName, nodeName)
			for _, itfce := range itfces {
				if *itfce.Endpoint.LagMemberLink {
					log.Infof("Interface Member Link: %s %s %s", *itfce.Endpoint.ShortName, *itfce.Endpoint.RealName, *itfce.Endpoint.LagName)
				} else {
					if *itfce.Endpoint.Lag {
						log.Infof("Interface LAG Link: %s %s", *itfce.Endpoint.ShortName, *itfce.Endpoint.RealName)
					} else {
						log.Infof("Interface Regular Link: %s %s", *itfce.Endpoint.ShortName, *itfce.Endpoint.RealName)
					}

				}
			}
		}
	}
	return nil
}

/*
// ParseWorkload parses the workload part of the configuration file
func (p *Parser) ParseWorkload() (err error) {
	log.Info("Parsing workload information ...")

	// initialize the Node information from the topology map
	for wlName, clients := range p.Config.Workloads {
		log.Infof("Workload Name: %s", wlName)
		p.Workloads[wlName] = &Workload{
			NetworkInstance: make(map[string]*NetworkInstance),
		}
		for cgName, wlInfo := range clients {
			log.Infof("clientGroup Name: %s", cgName)
			targetGroup := ""
			nodes := make([]string, 0)
			interfaces := make(map[string][]string)
			for _, l := range p.Links {
				if *l.A.Node.Target == cgName {
					if !*l.B.LagMemberLink {
						log.Infof("Enpoint %s %s %s %s", *l.B.Node.ShortName, *l.B.Node.Target, *l.B.ShortName, *l.B.RealName)
						targetGroup = *l.B.Node.Target
						found := false
						for _, n := range nodes {
							if n == *l.B.Node.ShortName {
								found = true
							}
						}
						if !found {
							nodes = append(nodes, *l.B.Node.ShortName)
						}
						interfaces[*l.B.Node.ShortName] = append(interfaces[*l.B.Node.ShortName], *l.B.RealName)
						found = false
						for _, itfceName := range interfaces[*l.B.Node.Target] {
							if itfceName == *l.B.RealName {
								found = true
							}
						}
						if !found {
							interfaces[*l.B.Node.Target] = append(interfaces[*l.B.Node.Target], *l.B.RealName)
						}
					}
				}
				if *l.B.Node.Target == cgName {
					if !*l.A.LagMemberLink {
						log.Infof("Enpoint %s %s %s %s", *l.A.Node.ShortName, *l.A.Node.Target, *l.A.ShortName, *l.A.RealName)
						targetGroup = *l.A.Node.Target
						found := false
						for _, n := range nodes {
							if n == *l.B.Node.ShortName {
								found = true
							}
						}
						if !found {
							nodes = append(nodes, *l.A.Node.ShortName)
						}
						interfaces[*l.A.Node.ShortName] = append(interfaces[*l.A.Node.ShortName], *l.A.RealName)
						found = false
						for _, itfceName := range interfaces[*l.A.Node.Target] {
							if itfceName == *l.A.RealName {
								found = true
							}
						}
						if !found {
							interfaces[*l.A.Node.Target] = append(interfaces[*l.A.Node.Target], *l.A.RealName)
						}
					}
				}
			}
			log.Info("group: %s", targetGroup)
			log.Info("nodes: %v", nodes)
			log.Info("interfaces: %v", interfaces)
			for k, netwInfo := range wlInfo.Vlans {
				switch k {
				case "itfce":
					switch *netwInfo.Kind {
					case "bridged":
						bridgedNetwInstName := strcase.LowerCamelCase(wlName) + "MacVrf" + strconv.Itoa(*netwInfo.Id)
						vxlanInterface := "vxlan0." + strconv.Itoa(*netwInfo.Id)
						p.Workloads[wlName].NetworkInstance[bridgedNetwInstName] = &NetworkInstance{
							Kind:           netwInfo.Kind,
							VlanId:         netwInfo.Id,
							Interfaces:     interfaces[targetGroup],
							VxlanInterface: vxlanInterface,
						}
					case "routed":
						routedNetwInstName := strcase.LowerCamelCase(wlName) + "IpVrf" + "IpVlan" + strconv.Itoa(*netwInfo.Id)
						vxlanInterface := "vxlan0." + strconv.Itoa(*netwInfo.Id)
						if _, ok := p.Workloads[wlName].NetworkInstance[routedNetwInstName]; !ok {
							p.Workloads[wlName].NetworkInstance[routedNetwInstName] = &NetworkInstance{
								Kind:           StringPtr("routed"),
								VlanId:         IntPtr(*netwInfo.Id),
								IPv4Cidr:       netwInfo.Ipv4Cidr,
								VxlanInterface: vxlanInterface,
								Interfaces:     make([]string, 0),
							}
						}
						p.Workloads[wlName].NetworkInstance[routedNetwInstName].Interfaces = append(p.Workloads[wlName].NetworkInstance[routedNetwInstName].Interfaces, interfaces[nodes[0]]...)
					case "irb":
					}
				case "ipvlan":
					bridgedNetwInstName := strcase.LowerCamelCase(wlName) + "MacVrf" + "IpVlan" + strconv.Itoa(*netwInfo.Id)
					interfaces[targetGroup] = append(interfaces[targetGroup], "irb0."+strconv.Itoa(*netwInfo.Id))
					vxlanInterface := "vxlan0." + strconv.Itoa(*netwInfo.Id)
					p.Workloads[wlName].NetworkInstance[bridgedNetwInstName] = &NetworkInstance{
						Kind:           StringPtr("bridged"),
						VlanId:         netwInfo.Id,
						IPv4Cidr:       netwInfo.Ipv4Cidr,
						Interfaces:     interfaces[targetGroup],
						VxlanInterface: vxlanInterface,
					}
					roundId := math.Round(float64(*netwInfo.Id) / 100)
					id := int(roundId)*100 + 5
					routedNetwInstName := strcase.LowerCamelCase(wlName) + "IpVrf" + "IpVlan" + strconv.Itoa(id)
					vxlanInterface = "vxlan0." + strconv.Itoa(id)
					if _, ok := p.Workloads[wlName].NetworkInstance[routedNetwInstName]; !ok {
						p.Workloads[wlName].NetworkInstance[routedNetwInstName] = &NetworkInstance{
							Kind:           StringPtr("routed"),
							VlanId:         IntPtr(id),
							IPv4Cidr:       netwInfo.Ipv4Cidr,
							VxlanInterface: vxlanInterface,
							Interfaces:     make([]string, 0),
						}
					}
					p.Workloads[wlName].NetworkInstance[routedNetwInstName].Interfaces = append(p.Workloads[wlName].NetworkInstance[routedNetwInstName].Interfaces, "irb0."+strconv.Itoa(*netwInfo.Id))
				case "sriov1":
					bridgedNetwInstName := strcase.LowerCamelCase(wlName) + "MacVrf" + "Sriov" + strconv.Itoa(*netwInfo.Id)
					interfaces[nodes[0]] = append(interfaces[nodes[0]], "irb0."+strconv.Itoa(*netwInfo.Id))
					vxlanInterface := "vxlan0." + strconv.Itoa(*netwInfo.Id)
					p.Workloads[wlName].NetworkInstance[bridgedNetwInstName] = &NetworkInstance{
						Kind:           StringPtr("bridged"),
						VlanId:         netwInfo.Id,
						IPv4Cidr:       netwInfo.Ipv4Cidr,
						Interfaces:     interfaces[nodes[0]],
						VxlanInterface: vxlanInterface,
					}
					roundId := math.Round(float64(*netwInfo.Id) / 100)
					id := int(roundId)*100 + 5
					routedNetwInstName := strcase.LowerCamelCase(wlName) + "IpVrf" + "IpVlan" + strconv.Itoa(id)
					vxlanInterface = "vxlan0." + strconv.Itoa(id)
					if _, ok := p.Workloads[wlName].NetworkInstance[routedNetwInstName]; !ok {
						p.Workloads[wlName].NetworkInstance[routedNetwInstName] = &NetworkInstance{
							Kind:           StringPtr("routed"),
							VlanId:         IntPtr(id),
							IPv4Cidr:       netwInfo.Ipv4Cidr,
							VxlanInterface: vxlanInterface,
							Interfaces:     make([]string, 0),
						}
					}
					p.Workloads[wlName].NetworkInstance[routedNetwInstName].Interfaces = append(p.Workloads[wlName].NetworkInstance[routedNetwInstName].Interfaces, "irb0."+strconv.Itoa(*netwInfo.Id))
				case "sriov2":
					bridgedNetwInstName := strcase.LowerCamelCase(wlName) + "MacVrf" + "Sriov" + strconv.Itoa(*netwInfo.Id)
					interfaces[nodes[1]] = append(interfaces[nodes[1]], "irb0."+strconv.Itoa(*netwInfo.Id))
					vxlanInterface := "vxlan0." + strconv.Itoa(*netwInfo.Id)
					p.Workloads[wlName].NetworkInstance[bridgedNetwInstName] = &NetworkInstance{
						Kind:           StringPtr("bridged"),
						VlanId:         netwInfo.Id,
						IPv4Cidr:       netwInfo.Ipv4Cidr,
						Interfaces:     interfaces[nodes[1]],
						VxlanInterface: vxlanInterface,
					}
					roundId := math.Round(float64(*netwInfo.Id) / 100)
					id := int(roundId)*100 + 5
					routedNetwInstName := strcase.LowerCamelCase(wlName) + "IpVrf" + "IpVlan" + strconv.Itoa(id)
					vxlanInterface = "vxlan0." + strconv.Itoa(id)
					if _, ok := p.Workloads[wlName].NetworkInstance[routedNetwInstName]; !ok {
						p.Workloads[wlName].NetworkInstance[routedNetwInstName] = &NetworkInstance{
							Kind:           StringPtr("routed"),
							VlanId:         IntPtr(id),
							IPv4Cidr:       netwInfo.Ipv4Cidr,
							VxlanInterface: vxlanInterface,
							Interfaces:     make([]string, 0),
						}
					}
					p.Workloads[wlName].NetworkInstance[routedNetwInstName].Interfaces = append(p.Workloads[wlName].NetworkInstance[routedNetwInstName].Interfaces, "irb0."+strconv.Itoa(*netwInfo.Id))
				}
			}
		}
	}

	return nil
}

// ShowWorkload show the workload that was initialized
func (p *Parser) ShowWorkload() {
	for wlName, w := range p.Workloads {
		for niName, ni := range w.NetworkInstance {
			log.Infof("Worload Name: %s, Network Instance: %s %s %d %s %v", wlName, niName, *ni.Kind, *ni.VlanId, ni.VxlanInterface, ni.Interfaces)
		}
	}
}
*/

func (p *Parser) InitializeIPAMWorkloads() (err error) {
	log.Info("initializing IPAM for workloads...")

	for wlName, clients := range p.Config.Workloads {
		for cgName, wlInfo := range clients {
			for _, netwInfo := range wlInfo.Itfces {
				if *netwInfo.Kind == "routed" {
					ipamName := wlName + cgName + strconv.Itoa(*netwInfo.VlanID)
					netwInfo := &NetworkInfo{
						Kind:                  StringPtr(ipamName),
						AddressingSchema:      netwInfo.AddressingSchema,
						Ipv4Cidr:              netwInfo.Ipv4Cidr,
						Ipv4ItfcePrefixLength: netwInfo.Ipv4ItfcePrefixLength,
						Ipv6Cidr:              netwInfo.Ipv6Cidr,
						Ipv6ItfcePrefixLength: netwInfo.Ipv6ItfcePrefixLength,
					}

					p.IPAM[ipamName], err = NewIPAM(netwInfo)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
