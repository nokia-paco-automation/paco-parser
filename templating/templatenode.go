package templating

import "fmt"

type TemplateNode struct {
	RoutingPolicy    string
	Interfaces       map[string]string            // interfacename -> conf
	SubInterfaces    map[string]map[string]string // interfacename, subifidentifier -> conf
	NetworkInstances map[string]string            // instancename -> conf
	Esis             []string
	VxlanInterface   string            // tunnel-interfaces
	Bgp              map[string]string // instancename -> conf
}

func NewTemplateNode() *TemplateNode {
	return &TemplateNode{
		RoutingPolicy:    "",
		Interfaces:       map[string]string{},
		SubInterfaces:    map[string]map[string]string{},
		NetworkInstances: map[string]string{},
		Esis:             []string{},
		VxlanInterface:   "",
		Bgp:              map[string]string{},
	}
}

func (t *TemplateNode) AddInterface(ifname string, conf string) {
	t.Interfaces[ifname] = conf
}

func (t *TemplateNode) AddSubInterface(ifname string, subifidentifier string, conf string) {
	initDoubleMap(t.SubInterfaces, ifname, subifidentifier)
	t.SubInterfaces[ifname][subifidentifier] = conf
}

func (t *TemplateNode) SetRoutingPolicy(conf string) {
	t.RoutingPolicy = conf
}

func (t *TemplateNode) AddBgp(instance string, conf string) {
	t.Bgp[instance] = conf
}

func (t *TemplateNode) AddEsi(conf string) {
	t.Esis = append(t.Esis, conf)
}

func (t *TemplateNode) SetVxlanInterface(conf string) {
	t.VxlanInterface = conf
}

func (t *TemplateNode) AddNetworkInstance(instancename string, conf string) {
	t.NetworkInstances[instancename] = conf
}

func (t *TemplateNode) MergeConfig() string {
	merger := NewJsonMerger()

	//routing policies
	merger.Merge([]byte(t.RoutingPolicy))
	// tunnel interface / vxlan interface
	merger.Merge([]byte(t.VxlanInterface))

	// Interfaces
	interfaceArrB := NewJsonArrayBuilder()
	for interfname, interf := range t.Interfaces {
		interfmerger := NewJsonMerger()
		subinterfacearrb := NewJsonArrayBuilder()
		for _, subif := range t.SubInterfaces[interfname] {
			subinterfacearrb.AddEntry(subif)
		}
		interfmerger.Merge([]byte(interf))
		interfmerger.Merge([]byte(subinterfacearrb.ToStringObj("subinterface")))
		interfaceArrB.AddEntry(interfmerger.ToString())
	}
	merger.Merge([]byte(interfaceArrB.ToStringObj("interface")))

	// network instances
	networkinstanceArrB := NewJsonArrayBuilder()
	for _, ni := range t.NetworkInstances {
		networkinstanceArrB.AddEntry(ni)
	}
	merger.Merge([]byte(networkinstanceArrB.ToStringObj("network-instance")))

	//ESI
	esi_templ := `
	{
		"system": {
		"network-instance": {
		  "protocols": {
			"bgp-vpn": {
			  "bgp-instance": [
				{
				  "id": 1
				}
			  ]
			},
			"evpn": {
			  "ethernet-segments": {
				"bgp-instance": [
				  {
					"id": "1",
					%s
				  }
				]
			  }
			}
		  }
		}
		}
	  }
	`
	esiArrB := NewJsonArrayBuilder()
	for _, esi := range t.Esis {
		esiArrB.AddEntry(esi)
	}
	esi_full_config := fmt.Sprintf(esi_templ, esiArrB.ToStringElem("ethernet-segment"))
	merger.Merge([]byte(esi_full_config))
	return merger.ToString()
}
