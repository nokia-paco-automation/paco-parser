package parser

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/nokia-paco-automation/paco-parser/srlygot"
	"github.com/openconfig/ygot/ygot"
	log "github.com/sirupsen/logrus"
)

func (p *Parser) WriteSwitchJSON() {
	log.Infof("Writing Switch infrastructure JSON objects...")

	dirName := filepath.Join(*p.BaseSwitchDir, "infra")
	p.CreateDirectory(dirName, 0777)

	k8ssrlinterfaces := make([]*k8ssrlinterface, 0)

	systemInterface := &k8ssrlinterface{
		Kind: "system",
		Name: "system0",
	}
	k8ssrlinterfaces = append(k8ssrlinterfaces, systemInterface)
	loopbackInterface := &k8ssrlinterface{
		Kind: "system",
		Name: "lo0",
	}
	k8ssrlinterfaces = append(k8ssrlinterfaces, loopbackInterface)
	irbInterface := &k8ssrlinterface{
		Kind: "system",
		Name: "irb0",
	}
	k8ssrlinterfaces = append(k8ssrlinterfaces, irbInterface)

	tunnelinterfaces := make([]*k8ssrlTunnelInterface, 0)
	tunnelInterface := &k8ssrlTunnelInterface{
		Name: "vxlan0",
	}
	tunnelinterfaces = append(tunnelinterfaces, tunnelInterface)

	// TODO need to add supernet
	var ipv4Cidr *string
	var ipv6Cidr *string
	for i := 0; i < len(p.Config.Infrastructure.Networks["loopback"].Ipv4Cidr); i++ {
		ipv4Cidr = p.Config.Infrastructure.Networks["loopback"].Ipv4Cidr[i]
		ipv6Cidr = p.Config.Infrastructure.Networks["loopback"].Ipv4Cidr[i]
	}

	routingPolicy := &k8ssrlRoutingPolicy{
		Name:              "export-underlay-local",
		IPv4Prefix:        *ipv4Cidr,
		IPv6Prefix:        *ipv6Cidr,
		IPv4PrefixSetName: "system-v4",
		IPv6PrefixSetName: "system-v6",
	}
	// DEBUG HANS
	fmt.Println(routingPolicy)

	for nodeName, n := range p.Nodes {
		// reinitialize parameters per node
		found := false
		islinterfaces := make([]*k8ssrlinterface, 0)
		islsubinterfaces := make([]*k8ssrlsubinterface, 0)
		systemsubinterfaces := make([]*k8ssrlsubinterface, 0)
		allsubinterfaces := make([]*k8ssrlsubinterface, 0)
		neighbors := make([]*Neighbor, 0)
		neighborLoopBackIPv4s := make(map[string]string)
		neighborLoopBackIPv6s := make(map[string]string)

		if *n.Position == "network" {
			devconfig := &srlygot.Device{}
			fmt.Printf("%T\n", nodeName)
			devconfig.System = &srlygot.SrlNokiaSystem_System{
				Name: &srlygot.SrlNokiaSystem_System_Name{
					HostName: &nodeName,
				},
			}

			for epName, ep := range n.Endpoints {
				if *ep.Kind == "isl" {
					found = true
					log.Debugf("Node name: %s, Interface: %s, %s, %t", nodeName, *ep.RealName, *ep.IPv4Prefix, *ep.VlanTagging)
					islinterface := &k8ssrlinterface{
						Kind:        "isl",
						Name:        *ep.RealName,
						VlanTagging: *ep.VlanTagging,
						PortSpeed:   "",
						Lag:         false,
						LagMember:   false,
					}

					devint, _ := devconfig.NewInterface(*ep.RealName)
					devint.Description = ep.RealName
					devint.AdminState = srlygot.SrlNokiaCommon_AdminState_enable

					if *ep.VlanTagging {
						devint.VlanTagging = ep.VlanTagging
					}

					//log.Infof("Ip Prefix: %s",  *ep.IPv4Address + "/" + strconv.Itoa(*ep.IPv4PrefixLength))
					//for _, ep := range ep.PeerNode.Endpoints {
					//	log.Infof("neighbor node Ip Prefix: %s %s %s",  *ep.IPv4Address + "/" + strconv.Itoa(*ep.IPv4PrefixLength), *ep.RealName, *ep.Kind)
					//}
					islsubinterface := &k8ssrlsubinterface{
						InterfaceRealName:  *ep.RealName,
						InterfaceShortName: epName,
						VlanTagging:        *ep.VlanTagging,
						VlanID:             *ep.VlanID,
						Kind:               "routed",
						IPv4Prefix:         *ep.IPv4Address + "/" + strconv.Itoa(*ep.IPv4PrefixLength),
						IPv6Prefix:         *ep.IPv6Address + "/" + strconv.Itoa(*ep.IPv6PrefixLength),
					}

					vlanid, _ := strconv.Atoi(*ep.VlanID)
					subint, _ := devint.NewSubinterface(uint32(vlanid))
					subint.Description = ep.RealName
					subint.Type = srlygot.SrlNokiaInterfaces_SiType_routed
					subint.AdminState = srlygot.SrlNokiaCommon_AdminState_enable
					subint.Vlan = &srlygot.SrlNokiaInterfaces_Interface_Subinterface_Vlan{Encap: &srlygot.SrlNokiaInterfaces_Interface_Subinterface_Vlan_Encap{}}
					if *ep.VlanTagging {
						if *ep.VlanID == "0" {
							subint.Vlan.Encap.Untagged = &srlygot.SrlNokiaInterfaces_Interface_Subinterface_Vlan_Encap_Untagged{}

						} else {
							vlan := &srlygot.SrlNokiaInterfaces_Interface_Subinterface_Vlan_Encap_SingleTagged_VlanId_Union_Uint16{Uint16: uint16(vlanid)}
							subint.Vlan.Encap.SingleTagged = &srlygot.SrlNokiaInterfaces_Interface_Subinterface_Vlan_Encap_SingleTagged{VlanId: vlan}
						}

					}

					neighbor := &Neighbor{
						PeerIP:           *ep.IPv4NeighborAddress,
						PeerAS:           *ep.PeerAS,
						PeerGroup:        "underlay",
						LocalAS:          0,
						TransportAddress: "",
					}
					islinterfaces = append(islinterfaces, islinterface)
					islsubinterfaces = append(islsubinterfaces, islsubinterface)
					allsubinterfaces = append(allsubinterfaces, islsubinterface)
					neighbors = append(neighbors, neighbor)
					// build a list of neighboring IPs to which BGP peering could be established in full mesh BGP w/o RR
					if _, ok := neighborLoopBackIPv4s[*ep.PeerNode.ShortName]; !ok {
						neighborLoopBackIPv4s[*ep.PeerNode.ShortName] = *ep.PeerNode.Endpoints["lo0"].IPv4Address
						neighborLoopBackIPv6s[*ep.PeerNode.ShortName] = *ep.PeerNode.Endpoints["lo0"].IPv6Address
					}
				}
				if *ep.Kind == "loopback" {
					systemsubinterface := &k8ssrlsubinterface{
						InterfaceRealName:  "system0",
						InterfaceShortName: "system0",
						VlanTagging:        false,
						VlanID:             "0",
						Kind:               "loopback", // used to indicate not to write the routed or bridged type
						IPv4Prefix:         *ep.IPv4Address + "/" + strconv.Itoa(*ep.IPv4PrefixLength),
						IPv6Prefix:         *ep.IPv6Address + "/" + strconv.Itoa(*ep.IPv6PrefixLength),
					}
					systemsubinterfaces = append(systemsubinterfaces, systemsubinterface)
					allsubinterfaces = append(allsubinterfaces, systemsubinterface)
				}
			}
			j, err := ygot.EmitJSON(devconfig, &ygot.EmitJSONConfig{
				Format: ygot.RFC7951,
				Indent: "  ",
				RFC7951Config: &ygot.RFC7951JSONConfig{
					AppendModuleName: true,
				},
			})
			if err != nil {
				panic(err)
			}
			fmt.Printf("FB JSON: %v\n", j)
		}
		if found {
			for neighborNodeName, neighborIP := range neighborLoopBackIPv4s {
				log.Debugf("Node Name: %s, Neighbor Node Name: %s", *n.ShortName, neighborNodeName)
				neighbor := &Neighbor{
					PeerIP:           neighborIP,
					PeerAS:           *p.Config.Infrastructure.Protocols.OverlayAs,
					PeerGroup:        "overlay",
					LocalAS:          *p.Config.Infrastructure.Protocols.OverlayAs,
					TransportAddress: *n.Endpoints["lo0"].IPv4Address,
				}
				neighbors = append(neighbors, neighbor)
			}

			defaultNetworkInstance := &k8ssrlNetworkInstance{
				Name:          "default",
				Kind:          "default",
				SubInterfaces: allsubinterfaces,
			}
			// DEBUG HANS
			fmt.Println(defaultNetworkInstance)

			peerGroups := make([]*PeerGroup, 0)
			switch *p.Config.Infrastructure.AddressingSchema {
			case "dual-stack":
				underlayPeerGroup := &PeerGroup{
					Name:       "underlay",
					PolicyName: "export-underlay-local",
					Protocols:  []string{"ipv4-unicast", "ipv6-unicast"},
				}
				peerGroups = append(peerGroups, underlayPeerGroup)
			case "v4-only":
				underlayPeerGroup := &PeerGroup{
					Name:      "underlay",
					Protocols: []string{"ipv4-unicast"},
				}
				peerGroups = append(peerGroups, underlayPeerGroup)
			case "v6-only":
				underlayPeerGroup := &PeerGroup{
					Name:      "underlay",
					Protocols: []string{"ipv6-unicast"},
				}
				peerGroups = append(peerGroups, underlayPeerGroup)
			}

			if p.Config.Infrastructure.Protocols.OverlayProtocol != nil && *p.Config.Infrastructure.Protocols.OverlayProtocol != "" {
				overlayPeerGroup := &PeerGroup{
					Name:      "overlay",
					Protocols: []string{*p.Config.Infrastructure.Protocols.OverlayProtocol},
				}
				peerGroups = append(peerGroups, overlayPeerGroup)
			}

			defaultProtocolBgp := &k8ssrlprotocolsbgp{
				NetworkInstanceName: "default",
				AS:                  *n.AS,
				RouterID:            *n.Endpoints["lo0"].IPv4Address,
				PeerGroups:          peerGroups,
				Neighbors:           neighbors,
			}
			fmt.Println(defaultProtocolBgp)

		}
	}

}
