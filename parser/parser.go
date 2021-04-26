package parser

import (
	log "github.com/sirupsen/logrus"
)

type Parser struct {
	BaseSwitchDir  *string
	BaseAppDir     *string
	BaseAppIpamDir *string
	ConfigFile     *ConfigFile
	Config         *Config
	Nodes          map[string]*Node
	Links          []*Link
	IPAM           map[string]*Ipam
	NextAS         *uint32
	Workloads      map[string]*Workload
	ClientGroups   map[string]*ClientGroup
	// DeploymentIPAM is a map where
	// first string key = multusNetworkName
	// 2nd Key string = ipvlan, sriov1, sriov2
	// 3rd key string = IP subnet, could be v4 or v6
	DeploymentIPAM map[string]map[string]map[string]*IpamApp

	// get the sriov and ipvlan networks and naming, etc
	// clientLinks["ipvlan"] -> map with sriov/ipvlan -> interface name of the server (bond0 or bond1 or multiple bonds
	// e.g. map[ipvlan:[bond0] sriov:[bond0]]
	// clientSriovInfo -> map[key=servername] with list fo switches
	// e.g. map[master0:[leaf1 leaf2]]
	ClientLinks map[string][]*ClientLinkInfo
	// identify the server nodes and its respective leaf/tor switches
	ClientSriovInfo map[string][]string
	SwitchInfo      *switchInfo

	debug bool
}

type ParserOption func(p *Parser)

// WithDebug initializes the debug flag
func WithDebug(d bool) ParserOption {
	return func(p *Parser) {
		p.debug = d
	}
}

// WithConfigFile initializes and marshals the config file
func WithConfigFile(file *string) ParserOption {
	return func(p *Parser) {
		if *file == "" {
			return
		}
		if err := p.GetConfig(file); err != nil {
			log.Fatalf("failed to read topology file: %v", err)
		}
	}
}

// WithOutput initializes the output variable
func WithOutput(o *string) ParserOption {
	return func(p *Parser) {
		p.BaseSwitchDir = StringPtr(*o + "/" + "kustomize")
		p.BaseAppDir = StringPtr(*o + "/" + "app")
		p.BaseAppIpamDir = StringPtr(*o + "/" + "app-ipam-csv")
	}
}

// NewParser function defines a new parser
func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{
		BaseSwitchDir:   new(string),
		BaseAppDir:      new(string),
		Config:          new(Config),
		ConfigFile:      new(ConfigFile),
		Nodes:           make(map[string]*Node),
		Links:           make([]*Link, 0),
		IPAM:            make(map[string]*Ipam),
		Workloads:       make(map[string]*Workload),
		ClientGroups:    make(map[string]*ClientGroup),
		NextAS:          new(uint32),
		DeploymentIPAM:  make(map[string]map[string]map[string]*IpamApp),
		ClientSriovInfo: make(map[string][]string),
		ClientLinks:     make(map[string][]*ClientLinkInfo),
	}

	// initialize the deployment IPAM, only use the ipvlan and sriov networks
	p.SwitchInfo = &switchInfo{
		// assign infrastructure/switch IP addresses
		// key1 = wlName, key2 = switchid -> value is bgp peer; 1 per switch
		switchBgpPeersIPv4: make(map[string]map[int]*BGPPeerInfo),
		switchBgpPeersIPv6: make(map[string]map[int]*BGPPeerInfo),
		// key1 = wlName; key2 = sriov1.1, ipvlan, key3 is switch id -> value gw ip
		switchGwsIPv4: make(map[string]map[string]map[int]map[int]string),
		switchGwsIPv6: make(map[string]map[string]map[int]map[int]string),
		// key1 = wlName;  key2 is switch id -> value = list of gw ip
		switchGwsPerWlNameIpv4: make(map[string]map[int][]string),
		switchGwsPerWlNameIpv6: make(map[string]map[int][]string),
	}

	for _, o := range opts {
		o(p)
	}
	return p
}
