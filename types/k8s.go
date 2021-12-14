package types

type K8ssrlinterface struct {
	Kind        string
	Name        string
	VlanTagging bool
	PortSpeed   string // 40G, 10G
	Lag         bool
	LagMember   bool
	LagName     string
	AdminKey    int
	SystemMac   string
	Pxe         bool
}

type K8ssrlsubinterface struct {
	InterfaceRealName  string
	InterfaceShortName string
	VlanTagging        bool
	VlanID             string
	Kind               string // routed or bridged
	IPv4Prefix         string
	IPv6Prefix         string
}

type K8ssrlirbsubinterface struct {
	InterfaceRealName string
	VlanID            string
	Description       string
	Kind              string // only routed
	IPv4Prefix        []string
	IPv6Prefix        []string
	AnycastGW         bool
	VrID              int
	NwType            string
}

type K8ssrlTunnelInterface struct {
	Name string
}

type K8ssrlVxlanInterface struct {
	TunnelInterfaceName string
	Kind                string // routed or bridged
	VlanID              string
}

type K8ssrlNetworkInstance struct {
	Name                string
	Type                string // bridged, irb, routed -> used to distinguish what to do with the interfaces
	Kind                string // default, mac-vrf, ip-vrf
	SubInterfaces       []*K8ssrlsubinterface
	TunnelInterfaceName string
	RouteTarget         string
	Evi                 int
	TargetNode          string
}

type K8ssrlprotocolsbgp struct {
	NetworkInstanceName string
	AS                  uint32
	RouterID            string
	PeerGroups          []*PeerGroup
	Neighbors           []*Neighbor
}

type PeerGroup struct {
	Name       string
	PolicyName string
	Protocols  []string
}

type Neighbor struct {
	PeerIP           string
	PeerAS           uint32
	PeerGroup        string
	LocalAS          uint32
	TransportAddress string
}

type K8ssrlESI struct {
	ESI     string
	LagName string
}

type K8ssrlRoutingPolicy struct {
	Name              string
	IPv4Prefix        string
	IPv4PrefixSetName string
	IPv6Prefix        string
	IPv6PrefixSetName string
}
