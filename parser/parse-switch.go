package parser

import (
	"fmt"
	"net"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/nokia-paco-automation/paco-parser/types"
	log "github.com/sirupsen/logrus"
	"github.com/stoewer/go-strcase"
)

func (p *Parser) WriteBase() {
	p.CreateDirectory(*p.BaseSwitchDir, 0777)
}

func (p *Parser) WriteFinalBase(kdirs []string) {
	dirName := filepath.Join(*p.BaseSwitchDir, "base")
	p.CreateDirectory(dirName, 0777)
	p.WriteKustomize(StringPtr(dirName), StringPtr("kustomization.yaml"), kdirs)
}

func (p *Parser) WriteInfrastructure() ([]string, *types.InfrastructureResult) {
	infrastructureResult := types.NewInfrastructureResult()

	var fileName string
	var kuztomizedirs []string
	log.Infof("Writing infrastructure k8s yaml objects...")
	dirName := filepath.Join(*p.BaseSwitchDir, "infra")
	p.CreateDirectory(dirName, 0777)

	kuztomizedirs = append(kuztomizedirs, "../infra")

	// this variable is used to write the kustomize file with all resources
	resources := make([]string, 0)

	k8ssrlinterfaces := make([]*types.K8ssrlinterface, 0)

	systemInterface := &types.K8ssrlinterface{
		Kind: "system",
		Name: "system0",
	}
	k8ssrlinterfaces = append(k8ssrlinterfaces, systemInterface)
	loopbackInterface := &types.K8ssrlinterface{
		Kind: "system",
		Name: "lo0",
	}
	k8ssrlinterfaces = append(k8ssrlinterfaces, loopbackInterface)
	irbInterface := &types.K8ssrlinterface{
		Kind: "system",
		Name: "irb0",
	}
	k8ssrlinterfaces = append(k8ssrlinterfaces, irbInterface)

	fileName = "interface-system.yaml"
	p.WriteSrlInterface(&dirName,
		StringPtr(fileName),
		StringPtr("infra-interface-system0"),
		StringPtr("leaf-grp1"),
		k8ssrlinterfaces)
	infrastructureResult.AppendSystemInterface(k8ssrlinterfaces)
	resources = append(resources, fileName)

	tunnelinterfaces := make([]*types.K8ssrlTunnelInterface, 0)
	tunnelInterface := &types.K8ssrlTunnelInterface{
		Name: "vxlan0",
	}
	tunnelinterfaces = append(tunnelinterfaces, tunnelInterface)

	fileName = "tunnel-interface-vxlan0.yaml"
	p.WriteSrlTunnelInterface(&dirName,
		StringPtr(fileName),
		StringPtr("infra-tunnel-interface-vxlan0"),
		StringPtr("leaf-grp1"),
		tunnelinterfaces)
	infrastructureResult.AppendTunnelInterfaces(tunnelinterfaces)
	resources = append(resources, fileName)

	// TODO need to add supernet
	var ipv4Cidr *string
	var ipv6Cidr *string
	for i := 0; i < len(p.Config.Infrastructure.Networks["loopback"].Ipv4Cidr); i++ {
		ipv4Cidr = p.Config.Infrastructure.Networks["loopback"].Ipv4Cidr[i]
		ipv6Cidr = p.Config.Infrastructure.Networks["loopback"].Ipv6Cidr[i]
	}

	routingPolicy := &types.K8ssrlRoutingPolicy{
		Name:              "export-underlay-local",
		IPv4Prefix:        *ipv4Cidr,
		IPv6Prefix:        *ipv6Cidr,
		IPv4PrefixSetName: "system-v4",
		IPv6PrefixSetName: "system-v6",
	}

	fileName = "routing-policy.yaml"
	p.WriteSrlRoutingPolicy(&dirName,
		StringPtr(fileName),
		StringPtr("infra-routing-policy"),
		StringPtr("leaf-grp1"),
		routingPolicy)
	infrastructureResult.SetRoutingPolicy(routingPolicy)
	resources = append(resources, fileName)

	for nodeName, n := range p.Nodes {
		// reinitialize parameters per node
		found := false
		islinterfaces := make([]*types.K8ssrlinterface, 0)
		islsubinterfaces := make([]*types.K8ssrlsubinterface, 0)
		systemsubinterfaces := make([]*types.K8ssrlsubinterface, 0)
		allsubinterfaces := make([]*types.K8ssrlsubinterface, 0)
		neighbors := make([]*types.Neighbor, 0)
		neighborLoopBackIPv4s := make(map[string]string)
		neighborLoopBackIPv6s := make(map[string]string)
		if *n.Position == "network" {
			for epName, ep := range n.Endpoints {
				nn := ""
				if ep.Node != nil {
					nn = DerefString(ep.Node.ShortName)
				}
				fmt.Printf("Node: %s, EPName: %s, Kind: %s, IP: %s\n", nn, DerefString(ep.RealName), DerefString(ep.Kind), DerefString(ep.IPv4Address))
				if *ep.Kind == "isl" {
					found = true
					log.Debugf("Node name: %s, Interface: %s, %s, %t", nodeName, *ep.RealName, *ep.IPv4Prefix, *ep.VlanTagging)
					islinterface := &types.K8ssrlinterface{
						Kind:        "isl",
						Name:        *ep.RealName,
						VlanTagging: *ep.VlanTagging,
						PortSpeed:   "",
						Lag:         false,
						LagMember:   false,
					}
					//log.Infof("Ip Prefix: %s",  *ep.IPv4Address + "/" + strconv.Itoa(*ep.IPv4PrefixLength))
					//for _, ep := range ep.PeerNode.Endpoints {
					//	log.Infof("neighbor node Ip Prefix: %s %s %s",  *ep.IPv4Address + "/" + strconv.Itoa(*ep.IPv4PrefixLength), *ep.RealName, *ep.Kind)
					//}
					islsubinterface := &types.K8ssrlsubinterface{
						InterfaceRealName:  *ep.RealName,
						InterfaceShortName: epName,
						VlanTagging:        *ep.VlanTagging,
						VlanID:             *ep.VlanID,
						Kind:               "routed",
						IPv4Prefix:         *ep.IPv4Address + "/" + strconv.Itoa(*ep.IPv4PrefixLength),
						IPv6Prefix:         *ep.IPv6Address + "/" + strconv.Itoa(*ep.IPv6PrefixLength),
					}
					neighbor := &types.Neighbor{
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
					systemsubinterface := &types.K8ssrlsubinterface{
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
		}
		if found {
			for neighborNodeName, neighborIP := range neighborLoopBackIPv4s {
				log.Debugf("Node Name: %s, Neighbor Node Name: %s", *n.ShortName, neighborNodeName)
				neighbor := &types.Neighbor{
					PeerIP:           neighborIP,
					PeerAS:           *p.Config.Infrastructure.Protocols.OverlayAs,
					PeerGroup:        "overlay",
					LocalAS:          *p.Config.Infrastructure.Protocols.OverlayAs,
					TransportAddress: *n.Endpoints["lo0"].IPv4Address,
				}
				neighbors = append(neighbors, neighbor)
			}
			// write isl interfaces
			// we have to send per device since the ip addresses are unique
			fileName = "interface-isl-" + nodeName + ".yaml"
			p.WriteSrlInterface(&dirName,
				StringPtr(fileName),
				StringPtr("infra-isl-interface"+nodeName),
				StringPtr(nodeName),
				islinterfaces)
			infrastructureResult.AppendIslInterfaces(nodeName, islinterfaces)
			resources = append(resources, fileName)

			// write isl subinterfaces
			// we have to send per device since the ip addresses are unique
			fileName = "subinterface-isl-" + islsubinterfaces[0].InterfaceShortName + "-" + nodeName + ".yaml"
			p.WriteSrlSubInterface(&dirName,
				StringPtr(fileName),
				StringPtr("infra-isl-subinterface"+islsubinterfaces[0].InterfaceShortName+"-"+nodeName),
				StringPtr(nodeName),
				islsubinterfaces)
			infrastructureResult.AppendIslSubInterfaces(nodeName, islsubinterfaces)
			resources = append(resources, fileName)

			// write system0 subinterface
			// we have to send per device since the ip addresses are unique
			fileName = "subinterface-" + "system0" + "-" + nodeName + ".yaml"
			p.WriteSrlSubInterface(&dirName,
				StringPtr(fileName),
				StringPtr("infra-system0-subinterface"+"-"+nodeName),
				StringPtr(nodeName),
				systemsubinterfaces)
			infrastructureResult.AppendSystemSubInterfaces(nodeName, systemsubinterfaces)
			resources = append(resources, fileName)
			defaultNetworkInstance := &types.K8ssrlNetworkInstance{
				Name:          "default",
				Kind:          "default",
				SubInterfaces: allsubinterfaces,
			}

			// write network instance default
			// we assume symetric config, so we send to all devices at once
			fileName = "network-instance-default" + "-" + nodeName + ".yaml"
			p.WriteSrlNetworkInstance(&dirName,
				StringPtr(fileName),
				StringPtr("infra-default-network-instance"+"-"+nodeName),
				StringPtr(nodeName), // we send it to all leafs at once assuming the configuration is symmetric
				defaultNetworkInstance)
			infrastructureResult.AppendDefaultNetworkInstance(nodeName, defaultNetworkInstance)
			resources = append(resources, fileName)

			peerGroups := make([]*types.PeerGroup, 0)
			switch *p.Config.Infrastructure.AddressingSchema {
			case "dual-stack":
				underlayPeerGroup := &types.PeerGroup{
					Name:       "underlay",
					PolicyName: "export-underlay-local",
					Protocols:  []string{"ipv4-unicast", "ipv6-unicast"},
				}
				peerGroups = append(peerGroups, underlayPeerGroup)
			case "v4-only":
				underlayPeerGroup := &types.PeerGroup{
					Name:      "underlay",
					Protocols: []string{"ipv4-unicast"},
				}
				peerGroups = append(peerGroups, underlayPeerGroup)
			case "v6-only":
				underlayPeerGroup := &types.PeerGroup{
					Name:      "underlay",
					Protocols: []string{"ipv6-unicast"},
				}
				peerGroups = append(peerGroups, underlayPeerGroup)
			}

			if p.Config.Infrastructure.Protocols.OverlayProtocol != nil && *p.Config.Infrastructure.Protocols.OverlayProtocol != "" {
				overlayPeerGroup := &types.PeerGroup{
					Name:      "overlay",
					Protocols: []string{*p.Config.Infrastructure.Protocols.OverlayProtocol},
				}
				peerGroups = append(peerGroups, overlayPeerGroup)
			}

			defaultProtocolBgp := &types.K8ssrlprotocolsbgp{
				NetworkInstanceName: "default",
				AS:                  *n.AS,
				RouterID:            *n.Endpoints["lo0"].IPv4Address,
				PeerGroups:          peerGroups,
				Neighbors:           neighbors,
			}

			// TODO Add Policies
			// write protocols bgp
			fileName = "protocols-bgp-default" + nodeName + ".yaml"
			p.WriteSrlProtocolsBgp(&dirName,
				StringPtr(fileName),
				StringPtr("infra-default-protocols-bgp"+nodeName),
				StringPtr(nodeName),
				defaultProtocolBgp)
			infrastructureResult.SetDefaultProtocolBgp(nodeName, defaultProtocolBgp)
			resources = append(resources, fileName)
		}
	}
	p.WriteKustomize(&dirName, StringPtr("kustomization.yaml"), resources)
	return kuztomizedirs, infrastructureResult
}

func (p *Parser) WriteClientsGroups() ([]string, *types.ClientGroupResults) {
	log.Infof("Writing Client group k8s yaml objects...")

	clientGroupResults := types.NewClientGroupResults()
	kuztomizedirs := []string{}
	for cgName, clientgroup := range p.ClientGroups {
		dirName := filepath.Join(*p.BaseSwitchDir, "client-"+cgName)
		p.CreateDirectory(dirName, 0777)

		kuztomizedirs = append(kuztomizedirs, "../client-"+cgName)

		resources := make([]string, 0)

		// we add all clientinterfaces to a list which we write at the end of the loop
		// to the respective file/directory

		for nodeName, itfces := range clientgroup.Interfaces {
			if nodeName != *clientgroup.TargetGroup {
				clientInterfaces := make([]*types.K8ssrlinterface, 0)
				for _, itfce := range itfces {
					if *itfce.Endpoint.Lag {
						id, _ := strconv.Atoi(strings.ReplaceAll(*itfce.Endpoint.RealName, "lag", ""))
						var systemMac string
						if id < 16 {
							systemMac = "00:00:00:00:00:0" + fmt.Sprintf("%X", id)
						} else {
							systemMac = "00:00:00:00:00:" + fmt.Sprintf("%X", id)
						}
						clientInterface := &types.K8ssrlinterface{
							Kind:        "access",
							Name:        *itfce.Endpoint.RealName,
							VlanTagging: true,
							PortSpeed:   *itfce.Endpoint.Speed,
							Lag:         *itfce.Endpoint.Lag,
							LagMember:   *itfce.Endpoint.LagMemberLink,
							LagName:     *itfce.Endpoint.LagName,
							AdminKey:    id,
							SystemMac:   systemMac,
							Pxe:         *itfce.Endpoint.Pxe,
						}
						clientInterfaces = append(clientInterfaces, clientInterface)
					} else {
						if *itfce.Endpoint.LagMemberLink {
							// no VLAN Tagging for member links
							clientInterface := &types.K8ssrlinterface{
								Kind:      "access",
								Name:      *itfce.Endpoint.RealName,
								PortSpeed: *itfce.Endpoint.Speed,
								Lag:       *itfce.Endpoint.Lag,
								LagMember: *itfce.Endpoint.LagMemberLink,
								LagName:   *itfce.Endpoint.LagName,
							}
							clientInterfaces = append(clientInterfaces, clientInterface)
						} else {
							clientInterface := &types.K8ssrlinterface{
								Kind:        *itfce.Endpoint.Kind,
								Name:        *itfce.Endpoint.RealName,
								VlanTagging: true,
								PortSpeed:   *itfce.Endpoint.Speed,
								Lag:         *itfce.Endpoint.Lag,
								LagMember:   *itfce.Endpoint.LagMemberLink,
								LagName:     *itfce.Endpoint.LagName,
							}
							clientInterfaces = append(clientInterfaces, clientInterface)
						}
					}
				}
				// write client interfaces
				// we need to write per device since pxe/lacp-fallback is different per node
				fileName := "interface-cg-" + cgName + "-" + nodeName + ".yaml"
				p.WriteSrlInterface(&dirName,
					StringPtr(fileName),
					StringPtr("cg-"+cgName+"-"+"interface-"+nodeName),
					StringPtr(nodeName),
					clientInterfaces)
				clientGroupResults.AppendClientInterfaces(nodeName, cgName, clientInterfaces)
				resources = append(resources, fileName)
			} else {
				// Target group Name
				// we check the esifound to see if we have to write the object
				esifound := false
				esis := make([]*types.K8ssrlESI, 0)
				for _, itfce := range itfces {
					if *itfce.Endpoint.Lag {
						if strings.Contains(*itfce.Endpoint.ShortName, "esi") {
							esifound = true
							id, _ := strconv.Atoi(strings.ReplaceAll(*itfce.Endpoint.RealName, "lag", ""))
							var esi string
							if id < 16 {
								esi = "00:12:12:12:12:12:12:00:00:0" + fmt.Sprintf("%X", id)
							} else {
								esi = "00:12:12:12:12:12:12:00:00:" + fmt.Sprintf("%X", id)
							}
							esiInterface := &types.K8ssrlESI{
								ESI:     esi,
								LagName: *itfce.Endpoint.RealName,
							}
							esis = append(esis, esiInterface)
						}
					}
				}
				if esifound {
					// TODO we need to make this more flexible and have a resource per client group
					// so split in infra part + client group part
					fileName := "system-network-instance-" + cgName + ".yaml"
					p.WriteSrlSystemNetworkInstance(&dirName,
						StringPtr(fileName),
						StringPtr("cg-"+cgName+"-"+"system-network-instance-"+nodeName),
						StringPtr(*clientgroup.TargetGroup),
						esis)
					clientGroupResults.AppendEsis(cgName, esis)
					resources = append(resources, fileName)
				}
			}
		}
		p.WriteKustomize(&dirName, StringPtr("kustomization.yaml"), resources)
	}
	return kuztomizedirs, clientGroupResults
}

func (p *Parser) WriteWorkloads() ([]string, *types.WorkloadResults) {
	log.Infof("Writing workload k8s yaml objects...")

	workloadresults := types.NewWorkloadResults()
	kuztomizedirs := []string{}

	for wlName, clients := range p.Config.Workloads {
		log.Debugf("Workload Name: %s", wlName)
		dirName := filepath.Join(*p.BaseSwitchDir, "workload-"+wlName)
		p.CreateDirectory(dirName, 0777)
		kuztomizedirs = append(kuztomizedirs, "../workload-"+wlName)

		// subinterface vxlan
		// first (string) key represents node name, interface is always vxlan0
		vxlanSubInterfaces := make(map[string][]*types.K8ssrlVxlanInterface, 0)
		// subinterface lag or real interface
		// first (string) key represents node name, 2nd string is interface name
		clientSubInterfaces := make(map[string]map[string][]*types.K8ssrlsubinterface)
		// subinterface irb for routed
		// first (string) key represents node name, interface is always irb0
		irbSubInterfaces := make(map[string][]*types.K8ssrlirbsubinterface)
		// network-instance
		// first (string) key represents node name, 2nd key represents the VlanId or network instance Id
		niIrbSubInterfaces := make(map[string]map[int][]*types.K8ssrlsubinterface)
		niCsiSubInterfaces := make(map[string]map[int][]*types.K8ssrlsubinterface)
		networkInstance := make(map[string]map[int]*types.K8ssrlNetworkInstance)

		// records the target group, such that we can write to the target group for the resources that allow it
		var targetGroup string
		for cgName, wlInfo := range clients {
			// if _, ok := wlInfo.Loopbacks["loopback"]; ok {

			// 	ipvrfVlanId := wlInfo.Itfces["ipvlan"].VlanID
			// 	loopbackinstance := wlInfo.Loopbacks["loopback"]
			// 	targetGroup = *p.ClientGroups[cgName].TargetGroup
			// 	for nodeName, _ := range p.ClientGroups[cgName].Interfaces {
			// 		if nodeName != targetGroup {
			// 			if loopbackinstance != nil {
			// 				// loopbackSubIf := &types.K8ssrlsubinterface{
			// 				// 	InterfaceRealName:  "lo0",
			// 				// 	InterfaceShortName: "lo0",
			// 				// 	VlanID:             string(*ipvlan.VlanID),
			// 				// 	Kind:               "routed",
			// 				// 	IPv4Prefix:         DerefString(loopbackinstance.IPv4BGPAddress),
			// 				// 	IPv6Prefix:         DerefString(loopbackinstance.IPv6BGPAddress),
			// 				// }
			// 				niName := strcase.KebabCase(strings.Replace(wlName, "multus-", "", 1)) + "-ipvrf-itfce-" + strconv.Itoa(*ipvrfVlanId)
			// 				fmt.Printf("WLName: %s, CGName: %s, IPv4: %s, IPv6: %s, NI: %s", wlName, cgName, DerefString(loopbackinstance.IPv4BGPAddress), DerefString(loopbackinstance.IPv6BGPAddress), niName)
			// 				newsubif := &types.K8ssrlirbsubinterface{
			// 					InterfaceRealName: "lo0",
			// 					VlanID:            strconv.Itoa(*loopbackinstance.Idx),
			// 					Description:       "",
			// 					Kind:              "",
			// 					IPv4Prefix:        []string{DerefString(loopbackinstance.IPv4BGPAddress)},
			// 					IPv6Prefix:        []string{DerefString(loopbackinstance.IPv6BGPAddress)},
			// 					AnycastGW:         false,
			// 					VrID:              0,
			// 					NwType:            "",
			// 				}
			// 				_ = newsubif
			// 				if _, ok := networkInstance[nodeName]; !ok {
			// 					networkInstance[nodeName] = make(map[int]*types.K8ssrlNetworkInstance)
			// 				}

			// 			}
			// 		}
			// 	}
			// }
			// netwType = itfce, ipvlan, sriov; netwInfo:
			for netwType, netwInfo := range wlInfo.Itfces {
				// used for vxlan write operation, so that we can send it to all devices in the group at once
				targetGroup = *p.ClientGroups[cgName].TargetGroup
				switch nwtype := netwType; {
				case nwtype == "itfce", nwtype == "ipvlan":
					switch *netwInfo.Kind {
					case "bridged":

						// no irb interface required for bridged networks

						for nodeName, itfces := range p.ClientGroups[cgName].Interfaces {
							// client interfaces are implemented individually per node
							if nodeName != targetGroup {
								if _, ok := vxlanSubInterfaces[nodeName]; !ok {
									vxlanSubInterfaces[nodeName] = make([]*types.K8ssrlVxlanInterface, 0)
								}
								vxlanSubInterface := &types.K8ssrlVxlanInterface{
									TunnelInterfaceName: "vxlan0",
									VlanID:              strconv.Itoa(*netwInfo.VlanID),
									Kind:                "bridged",
								}
								vxlanSubInterfaces[nodeName] = append(vxlanSubInterfaces[nodeName], vxlanSubInterface)

								if _, ok := networkInstance[nodeName]; !ok {
									networkInstance[nodeName] = make(map[int]*types.K8ssrlNetworkInstance)
								}
								if _, ok := networkInstance[nodeName][*netwInfo.VlanID]; !ok {
									evi := *netwInfo.VlanID
									if evi == 0 {
										evi = 1
									}
									// remove 1 and 2 from sriov1 and sriov2
									netwTypeName := netwType
									//netwTypeName := strings.TrimRight(netwType, "1")
									//netwTypeName = strings.TrimRight(netwTypeName, "2")
									niName := strcase.KebabCase(strings.Replace(wlName, "multus-", "", 1)) + "-macvrf-" + netwTypeName + "-" + strconv.Itoa(*netwInfo.VlanID)
									networkInstance[nodeName][*netwInfo.VlanID] = &types.K8ssrlNetworkInstance{
										Name:                niName,
										Kind:                "mac-vrf",
										Type:                "bridged",
										TunnelInterfaceName: "vxlan0" + "." + strconv.Itoa(*netwInfo.VlanID),
										RouteTarget:         "target:" + strconv.Itoa(int(*p.Config.Infrastructure.Protocols.OverlayAs)) + ":" + strconv.Itoa(*netwInfo.VlanID),
										Evi:                 evi,
									}
								}
								// check if clientSubInterfaces[nodeName] was already initialized, if not initialize it
								if _, ok := clientSubInterfaces[nodeName]; !ok {
									clientSubInterfaces[nodeName] = make(map[string][]*types.K8ssrlsubinterface)
								}
								// check if clientSubInterfaces[nodeName] was already initialized, if not initialize it

								if _, ok := niCsiSubInterfaces[nodeName]; !ok {
									niCsiSubInterfaces[nodeName] = make(map[int][]*types.K8ssrlsubinterface)
								}

								for _, itfce := range itfces {
									// exclude the interfaces with member link since they will be covered as a lag
									if !*itfce.Endpoint.LagMemberLink {
										log.Debugf("Interface name: %s", *itfce.Endpoint.RealName)
										// check if clientSubInterfaces[nodeName][*itfce.Endpoint.RealName] was already initialized if not initialize it
										if _, ok := clientSubInterfaces[nodeName][*itfce.Endpoint.RealName]; !ok {
											clientSubInterfaces[nodeName][*itfce.Endpoint.RealName] = make([]*types.K8ssrlsubinterface, 0)
										}
										csi := &types.K8ssrlsubinterface{
											InterfaceRealName:  *itfce.Endpoint.RealName,
											InterfaceShortName: *itfce.Endpoint.ShortName,
											VlanID:             strconv.Itoa(*netwInfo.VlanID),
											Kind:               "bridged",
										}
										clientSubInterfaces[nodeName][*itfce.Endpoint.RealName] = append(clientSubInterfaces[nodeName][*itfce.Endpoint.RealName], csi)
										// check if niSubInterfaces[nodeName][*itfce.Endpoint.RealName] was already initialized if not initialize it

										if _, ok := niCsiSubInterfaces[nodeName][*netwInfo.VlanID]; !ok {
											niCsiSubInterfaces[nodeName][*netwInfo.VlanID] = make([]*types.K8ssrlsubinterface, 0)
										}
										niCsiSubInterfaces[nodeName][*netwInfo.VlanID] = append(niCsiSubInterfaces[nodeName][*netwInfo.VlanID], csi)

									}
								}
							}
						}
					case "routed":
						// bridged part of the config, also the irb part

						dataref := p.ClientGroups[cgName].Interfaces
						keys := make([]string, 0, len(dataref))
						for k := range dataref {
							keys = append(keys, k)
						}
						sort.Strings(keys)

						for _, nodeName := range keys {
							itfces := dataref[nodeName]
							// client interfaces are implemented individually per node
							if nodeName != targetGroup {
								if _, ok := vxlanSubInterfaces[nodeName]; !ok {
									vxlanSubInterfaces[nodeName] = make([]*types.K8ssrlVxlanInterface, 0)
								}
								vxlanSubInterface := &types.K8ssrlVxlanInterface{
									TunnelInterfaceName: "vxlan0",
									VlanID:              strconv.Itoa(*netwInfo.VlanID),
									Kind:                "routed",
								}
								vxlanSubInterfaces[nodeName] = append(vxlanSubInterfaces[nodeName], vxlanSubInterface)
								if _, ok := networkInstance[nodeName]; !ok {
									networkInstance[nodeName] = make(map[int]*types.K8ssrlNetworkInstance)
								}
								if _, ok := networkInstance[nodeName][*netwInfo.VlanID]; !ok {
									evi := *netwInfo.VlanID
									if evi == 0 {
										evi = 1
									}
									netwTypeName := netwType
									//netwTypeName := strings.TrimRight(netwType, "1")
									//netwTypeName = strings.TrimRight(netwTypeName, "2")
									niName := strcase.KebabCase(strings.Replace(wlName, "multus-", "", 1)) + "-ipvrf-" + netwTypeName + "-" + strconv.Itoa(*netwInfo.VlanID)
									networkInstance[nodeName][*netwInfo.VlanID] = &types.K8ssrlNetworkInstance{
										Name:                niName,
										Kind:                "ip-vrf",
										Type:                "routed",
										TunnelInterfaceName: "vxlan0" + "." + strconv.Itoa(*netwInfo.VlanID),
										RouteTarget:         "target:" + strconv.Itoa(int(*p.Config.Infrastructure.Protocols.OverlayAs)) + ":" + strconv.Itoa(*netwInfo.VlanID),
										Evi:                 evi,
									}
								}
								// check if clientSubInterfaces[nodeName] was already initialized is not initialize it
								if _, ok := clientSubInterfaces[nodeName]; !ok {
									clientSubInterfaces[nodeName] = make(map[string][]*types.K8ssrlsubinterface, 0)
								}
								// check if niCsiSubInterfaces[nodeName] was already initialized, if not initialize it

								if _, ok := niCsiSubInterfaces[nodeName]; !ok {
									niCsiSubInterfaces[nodeName] = make(map[int][]*types.K8ssrlsubinterface)
								}

								for _, itfce := range itfces {
									// exclude the interfaces with member link since they will be covered as a lag
									if !*itfce.Endpoint.LagMemberLink {
										// allocate an IP from IPAM
										// find the iplink
										var link *Link
										var foundA, foundB bool
										var ipv4prefix, ipv6prefix string
										for _, l := range p.Links {
											if l.A != nil && l.A == itfce.Endpoint {
												link = l
												foundA = true
												break
											}
											if l.B != nil && l.B == itfce.Endpoint {
												link = l
												foundB = true
												break
											}
										}
										if foundA || foundB {
											log.Debugf("Link Found")
											ipamName := wlName + cgName + strconv.Itoa(*netwInfo.VlanID)
											var ipv4Cidr *string
											var ipv6Cidr *string
											for i := 0; i < len(netwInfo.Ipv4Cidr); i++ {
												ipv4Cidr = netwInfo.Ipv4Cidr[i]
												ipv6Cidr = netwInfo.Ipv6Cidr[i]
												if err := p.IPAM[ipamName].IPAMAllocateLinkPrefix(link, ipv4Cidr, ipv6Cidr); err != nil {
													log.Error(err)
												}
												if foundA {
													ipv4prefix = *link.A.IPv4Prefix
													ipv6prefix = *link.A.IPv6Prefix
													log.Debugf("IP Address: %s %s %s", *link.A.IPv4Prefix, *link.A.RealName, *link.B.RealName)
												}
												if foundB {
													ipv4prefix = *link.A.IPv4Prefix
													ipv6prefix = *link.A.IPv6Prefix
													log.Debugf("IP Address: %s %s %s", *link.B.IPv4Prefix, *link.B.RealName, *link.A.RealName)
												}
											}
										} else {
											log.Fatalf("Link Not found")
										}

										//avoids using the srl long interface name with the ethernet-1/50
										var newName string
										if strings.Contains(*itfce.Endpoint.RealName, "/") {
											newName = *itfce.Endpoint.ShortName
										} else {
											newName = *itfce.Endpoint.ShortName
										}
										log.Debugf("Interface name: %s", newName)
										// check if clientSubInterfaces[nodeName][newName] was already initialized if not initialize it
										if _, ok := clientSubInterfaces[nodeName][newName]; !ok {
											clientSubInterfaces[nodeName][newName] = make([]*types.K8ssrlsubinterface, 0)
										}
										csi := &types.K8ssrlsubinterface{
											InterfaceRealName:  *itfce.Endpoint.RealName,
											InterfaceShortName: *itfce.Endpoint.ShortName,
											VlanTagging:        true,
											VlanID:             strconv.Itoa(*netwInfo.VlanID),
											Kind:               "routed",
											IPv4Prefix:         ipv4prefix,
											IPv6Prefix:         ipv6prefix,
										}
										clientSubInterfaces[nodeName][newName] = append(clientSubInterfaces[nodeName][*itfce.Endpoint.RealName], csi)

										// check if niSubInterfaces[nodeName][*itfce.Endpoint.RealName] was already initialized if not initialize it

										if _, ok := niCsiSubInterfaces[nodeName][*netwInfo.VlanID]; !ok {
											niCsiSubInterfaces[nodeName][*netwInfo.VlanID] = make([]*types.K8ssrlsubinterface, 0)
										}
										niCsiSubInterfaces[nodeName][*netwInfo.VlanID] = append(niCsiSubInterfaces[nodeName][*netwInfo.VlanID], csi)

									}
								}
							}
						}
					case "irb":
						// bridged part of the config

						for nodeName, itfces := range p.ClientGroups[cgName].Interfaces {
							// client interfaces are implemented individually per node
							if nodeName != targetGroup {
								if _, ok := vxlanSubInterfaces[nodeName]; !ok {
									vxlanSubInterfaces[nodeName] = make([]*types.K8ssrlVxlanInterface, 0)
								}
								vxlanSubInterface := &types.K8ssrlVxlanInterface{
									TunnelInterfaceName: "vxlan0",
									VlanID:              strconv.Itoa(*netwInfo.VlanID),
									Kind:                "bridged",
								}
								vxlanSubInterfaces[nodeName] = append(vxlanSubInterfaces[nodeName], vxlanSubInterface)
								if _, ok := networkInstance[nodeName]; !ok {
									networkInstance[nodeName] = make(map[int]*types.K8ssrlNetworkInstance)
								}
								if _, ok := networkInstance[nodeName][*netwInfo.VlanID]; !ok {
									evi := *netwInfo.VlanID
									if evi == 0 {
										evi = 1
									}
									netwTypeName := netwType
									//netwTypeName := strings.TrimRight(netwType, "1")
									//netwTypeName = strings.TrimRight(netwTypeName, "2")
									niName := strcase.KebabCase(strings.Replace(wlName, "multus-", "", 1)) + "-macvrf-" + netwTypeName + "-" + strconv.Itoa(*netwInfo.VlanID)
									networkInstance[nodeName][*netwInfo.VlanID] = &types.K8ssrlNetworkInstance{
										Name:                niName,
										Kind:                "mac-vrf",
										Type:                "irb",
										TunnelInterfaceName: "vxlan0" + "." + strconv.Itoa(*netwInfo.VlanID),
										RouteTarget:         "target:" + strconv.Itoa(int(*p.Config.Infrastructure.Protocols.OverlayAs)) + ":" + strconv.Itoa(*netwInfo.VlanID),
										Evi:                 evi,
									}
								}
								// check if clientSubInterfaces[nodeName] was already initialized, if not initialize it
								if _, ok := clientSubInterfaces[nodeName]; !ok {
									clientSubInterfaces[nodeName] = make(map[string][]*types.K8ssrlsubinterface)
								}
								// check if niIrbSubInterfaces[nodeName] was already initialized, if not initialize it

								if _, ok := niIrbSubInterfaces[nodeName]; !ok {
									niIrbSubInterfaces[nodeName] = make(map[int][]*types.K8ssrlsubinterface)
								}

								if _, ok := niCsiSubInterfaces[nodeName]; !ok {
									niCsiSubInterfaces[nodeName] = make(map[int][]*types.K8ssrlsubinterface)
								}

								for _, itfce := range itfces {
									// exclude the interfaces with member link since they will be covered as a lag
									if !*itfce.Endpoint.LagMemberLink {
										log.Debugf("Interface name: %s", *itfce.Endpoint.RealName)
										// check if clientSubInterfaces[nodeName][*itfce.Endpoint.RealName] was already initialized if not initialize it
										if _, ok := clientSubInterfaces[nodeName][*itfce.Endpoint.RealName]; !ok {
											clientSubInterfaces[nodeName][*itfce.Endpoint.RealName] = make([]*types.K8ssrlsubinterface, 0)
										}
										csi := &types.K8ssrlsubinterface{
											InterfaceRealName:  *itfce.Endpoint.RealName,
											InterfaceShortName: *itfce.Endpoint.ShortName,
											VlanTagging:        true,
											VlanID:             strconv.Itoa(*netwInfo.VlanID),
											Kind:               "bridged",
										}
										clientSubInterfaces[nodeName][*itfce.Endpoint.RealName] = append(clientSubInterfaces[nodeName][*itfce.Endpoint.RealName], csi)

										// check if niSubInterfaces[nodeName][*itfce.Endpoint.RealName] was already initialized if not initialize it

										if _, ok := niCsiSubInterfaces[nodeName][*netwInfo.VlanID]; !ok {
											niCsiSubInterfaces[nodeName][*netwInfo.VlanID] = make([]*types.K8ssrlsubinterface, 0)
										}
										niCsiSubInterfaces[nodeName][*netwInfo.VlanID] = append(niCsiSubInterfaces[nodeName][*netwInfo.VlanID], csi)

									}
								}
								// check if irbSubInterfaces[nodeName] was already initialized, if not initialize it
								if _, ok := irbSubInterfaces[nodeName]; !ok {
									irbSubInterfaces[nodeName] = make([]*types.K8ssrlirbsubinterface, 0)
								}

								var ipv4Cidr *string
								var ipv6Cidr *string
								var ipv4prefixlist []string
								var ipv6prefixlist []string
								ipv4prefixlist = make([]string, 0)
								ipv6prefixlist = make([]string, 0)
								for i := 0; i < len(netwInfo.Ipv4Cidr); i++ {
									ipv4Cidr = netwInfo.Ipv4Cidr[i]
									ipv6Cidr = netwInfo.Ipv6Cidr[i]

									ipv4prefix, err := getFirstIPPrefixInCidr(ipv4Cidr)
									if err != nil {
										log.Fatal(err)
									}
									ipv4prefixlist = append(ipv4prefixlist, *ipv4prefix)

									ipv6prefix, err := getFirstIPPrefixInCidr(ipv6Cidr)
									if err != nil {
										log.Fatal(err)
									}
									ipv6prefixlist = append(ipv6prefixlist, *ipv6prefix)
								}

								irb := &types.K8ssrlirbsubinterface{
									InterfaceRealName: "irb0",
									Description:       "irb0",
									VlanID:            strconv.Itoa(*netwInfo.VlanID),
									Kind:              "routed",
									AnycastGW:         true,
									VrID:              10,
									IPv4Prefix:        ipv4prefixlist,
									IPv6Prefix:        ipv6prefixlist,
								}
								irbSubInterfaces[nodeName] = append(irbSubInterfaces[nodeName], irb)

								// check if niSubInterfaces[nodeName][*itfce.Endpoint.RealName] was already initialized if not initialize it

								if _, ok := niIrbSubInterfaces[nodeName][*netwInfo.VlanID]; !ok {
									niIrbSubInterfaces[nodeName][*netwInfo.VlanID] = make([]*types.K8ssrlsubinterface, 0)
								}

								nii := &types.K8ssrlsubinterface{
									InterfaceRealName:  "irb0",
									InterfaceShortName: "irb0",
									VlanTagging:        false,
									VlanID:             strconv.Itoa(*netwInfo.VlanID),
									Kind:               "routed",
								}
								niIrbSubInterfaces[nodeName][*netwInfo.VlanID] = append(niIrbSubInterfaces[nodeName][*netwInfo.VlanID], nii)

							}
						}
						// routed part of the config is covered in the routed part of the config

					}
				case strings.HasPrefix(nwtype, "sriov"):
					// we assume sriov is always irb based
					for nodeName, itfces := range p.ClientGroups[cgName].Interfaces {
						// client interfaces are implemented individually per node
						// we only add the interface if they belong to a target group in the netwInfo
						if nodeName != targetGroup && netwInfo.Target != nil && *netwInfo.Target == nodeName {
							if _, ok := vxlanSubInterfaces[nodeName]; !ok {
								vxlanSubInterfaces[nodeName] = make([]*types.K8ssrlVxlanInterface, 0)
							}
							vxlanSubInterface := &types.K8ssrlVxlanInterface{
								TunnelInterfaceName: "vxlan0",
								VlanID:              strconv.Itoa(*netwInfo.VlanID),
								Kind:                "bridged",
							}
							vxlanSubInterfaces[nodeName] = append(vxlanSubInterfaces[nodeName], vxlanSubInterface)

							if _, ok := networkInstance[nodeName]; !ok {
								networkInstance[nodeName] = make(map[int]*types.K8ssrlNetworkInstance)
							}
							if _, ok := networkInstance[nodeName][*netwInfo.VlanID]; !ok {
								evi := *netwInfo.VlanID
								if evi == 0 {
									evi = 1
								}
								netwTypeName := netwType
								//netwTypeName := strings.TrimRight(netwType, "1")
								//netwTypeName = strings.TrimRight(netwTypeName, "2")
								niName := strcase.KebabCase(strings.Replace(wlName, "multus-", "", 1)) + "-macvrf-" + netwTypeName + "-" + strconv.Itoa(*netwInfo.VlanID)
								networkInstance[nodeName][*netwInfo.VlanID] = &types.K8ssrlNetworkInstance{
									Name:                niName,
									Kind:                "mac-vrf",
									Type:                "irb",
									TunnelInterfaceName: "vxlan0" + "." + strconv.Itoa(*netwInfo.VlanID),
									RouteTarget:         "target:" + strconv.Itoa(int(*p.Config.Infrastructure.Protocols.OverlayAs)) + ":" + strconv.Itoa(*netwInfo.VlanID),
									Evi:                 evi,
								}
							}
							// check if clientSubInterfaces[nodeName] was already initialized is not initialize it
							if _, ok := clientSubInterfaces[nodeName]; !ok {

								clientSubInterfaces[nodeName] = make(map[string][]*types.K8ssrlsubinterface, 0)
							}
							// check if clientSubInterfaces[nodeName] was already initialized, if not initialize it

							if _, ok := niIrbSubInterfaces[nodeName]; !ok {
								niIrbSubInterfaces[nodeName] = make(map[int][]*types.K8ssrlsubinterface)
							}

							if _, ok := niCsiSubInterfaces[nodeName]; !ok {
								niCsiSubInterfaces[nodeName] = make(map[int][]*types.K8ssrlsubinterface)
							}

							for _, itfce := range itfces {
								// exclude the interfaces with member link since they will be covered as a lag
								if !*itfce.Endpoint.LagMemberLink {
									log.Debugf("Interface name: %s", *itfce.Endpoint.RealName)
									// check if clientSubInterfaces[nodeName][*itfce.Endpoint.RealName] was already initialized if not initialize it
									if _, ok := clientSubInterfaces[nodeName][*itfce.Endpoint.RealName]; !ok {
										clientSubInterfaces[nodeName][*itfce.Endpoint.RealName] = make([]*types.K8ssrlsubinterface, 0)
									}
									csi := &types.K8ssrlsubinterface{
										InterfaceRealName:  *itfce.Endpoint.RealName,
										InterfaceShortName: *itfce.Endpoint.ShortName,
										VlanTagging:        true,
										VlanID:             strconv.Itoa(*netwInfo.VlanID),
										Kind:               "bridged",
									}
									clientSubInterfaces[nodeName][*itfce.Endpoint.RealName] = append(clientSubInterfaces[nodeName][*itfce.Endpoint.RealName], csi)

									// check if niSubInterfaces[nodeName][*itfce.Endpoint.RealName] was already initialized if not initialize it

									if _, ok := niCsiSubInterfaces[nodeName][*netwInfo.VlanID]; !ok {
										niCsiSubInterfaces[nodeName][*netwInfo.VlanID] = make([]*types.K8ssrlsubinterface, 0)
									}
									niCsiSubInterfaces[nodeName][*netwInfo.VlanID] = append(niCsiSubInterfaces[nodeName][*netwInfo.VlanID], csi)

								}
							}
							// check if irbSubInterfaces[nodeName] was already initialized, if not initialize it
							if _, ok := irbSubInterfaces[nodeName]; !ok {
								irbSubInterfaces[nodeName] = make([]*types.K8ssrlirbsubinterface, 0)

							}

							ipv4prefixlist := make([]string, 0)
							ipv6prefixlist := make([]string, 0)
							var ipNet *net.IPNet
							var err error
							//var ipNetList []net.IPNet
							for i := 0; i < len(netwInfo.Ipv4Cidr); i++ {
								_, ipNet, err = net.ParseCIDR(*netwInfo.Ipv4Cidr[i])
								if err != nil {
									log.Fatal(err)
								}
								/*
									ipNetList, err = ipam.Split(*ipNet, 8)
									if err != nil {
										log.Fatal(err)
									}
									ipv4prefixlist = make([]string, 0)
									for _, ipnet := range ipNetList {
										ipv4prefix, err := getLastIPPrefixInIPnet(ipnet)
										if err != nil {
											log.Fatal(err)
										}
										ipv4prefixlist = append(ipv4prefixlist, *ipv4prefix)
									}
								*/
								ipv4prefix, err := getFirstIPPrefixInIPnet(*ipNet)
								if err != nil {
									log.Fatal(err)
								}
								ipv4prefixlist = append(ipv4prefixlist, *ipv4prefix)

								_, ipNet, err = net.ParseCIDR(*netwInfo.Ipv6Cidr[i])
								if err != nil {
									log.Fatal(err)
								}
								/*
									ipNetList, err = ipam.Split(*ipNet, 8)
									if err != nil {
										log.Fatal(err)
									}
									ipv6prefixlist = make([]string, 0)
									for _, ipnet := range ipNetList {
										ipv4prefix, err := getLastIPPrefixInIPnet(ipnet)
										if err != nil {
											log.Fatal(err)
										}
										ipv6prefixlist = append(ipv6prefixlist, *ipv4prefix)
									}
								*/
								ipv6prefix, err := getFirstIPPrefixInIPnet(*ipNet)
								if err != nil {
									log.Fatal(err)
								}
								ipv6prefixlist = append(ipv6prefixlist, *ipv6prefix)
							}

							// TODO SPLIT FUNCTIOn FOR IPv6 needs to be added for SRIOV

							/*
								_, ipNet, err = net.ParseCIDR(*netwInfo.Ipv6Cidr)
								ipNetList, err = ipam.Split(*ipNet, 8)
								if err != nil {
									log.Fatal(err)
								}
								ipv6prefixlist := make([]string, 0)
								for _, ipnet := range ipNetList {
									ipv6prefix, err := getLastIPPrefixInIPnet(ipnet)
									if err != nil {
										log.Fatal(err)
									}
									ipv6prefixlist = append(ipv4prefixlist, *ipv6prefix)
								}
							*/

							irb := &types.K8ssrlirbsubinterface{
								InterfaceRealName: "irb0",
								Description:       "irb0",
								VlanID:            strconv.Itoa(*netwInfo.VlanID),
								Kind:              "routed",
								AnycastGW:         false,
								VrID:              10,
								IPv4Prefix:        ipv4prefixlist,
								//IPv6Prefix:        ipv6prefixlist,
								NwType: nwtype,
							}
							irbSubInterfaces[nodeName] = append(irbSubInterfaces[nodeName], irb)

							if _, ok := niIrbSubInterfaces[nodeName][*netwInfo.VlanID]; !ok {
								niIrbSubInterfaces[nodeName][*netwInfo.VlanID] = make([]*types.K8ssrlsubinterface, 0)
							}

							nii := &types.K8ssrlsubinterface{
								InterfaceRealName:  "irb0",
								InterfaceShortName: "irb0",
								VlanTagging:        false,
								VlanID:             strconv.Itoa(*netwInfo.VlanID),
								Kind:               "routed",
							}
							niIrbSubInterfaces[nodeName][*netwInfo.VlanID] = append(niIrbSubInterfaces[nodeName][*netwInfo.VlanID], nii)
						}
					}
					// routed part of the config is covered in the routed part of the config
				}
			}
		}
		resources := make([]string, 0)

		// we have to create seperate files, since the interface is unique
		for nodeName, clientSubInterface := range clientSubInterfaces {
			if _, ok := vxlanSubInterfaces[nodeName]; ok {
				fileName := "vxlaninterface" + "-" + "vxlan0" + "-" + nodeName + ".yaml"
				p.WriteSrlVxlanInterface(&dirName,
					StringPtr(fileName),
					StringPtr(wlName+"-vxlaninterface-"+"vxlan0"+"-"+nodeName),
					StringPtr(nodeName),
					vxlanSubInterfaces[nodeName])
				workloadresults.AppendVxlanSubInterfaces(nodeName, vxlanSubInterfaces[nodeName])
				resources = append(resources, fileName)
			}

			for itfceName, csi := range clientSubInterface {
				fileName := "subinterface" + "-" + itfceName + "-" + nodeName + ".yaml"
				p.WriteSrlSubInterface(&dirName,
					StringPtr(fileName),
					StringPtr(wlName+"-subinterface-"+itfceName+"-"+nodeName),
					StringPtr(nodeName),
					csi)
				workloadresults.AppendClientSubInterfaces(nodeName, itfceName, csi)
				resources = append(resources, fileName)
			}
			if _, ok := irbSubInterfaces[nodeName]; ok {
				fileName := "subinterface" + "-" + "irb0" + "-" + nodeName + ".yaml"
				p.WriteSrlIrbSubInterface(&dirName,
					StringPtr(fileName),
					StringPtr(wlName+"-subinterface-"+"irb0"+"-"+nodeName),
					StringPtr(nodeName),
					irbSubInterfaces[nodeName])
				workloadresults.AppendIrbSubInterfaces(nodeName, irbSubInterfaces[nodeName])
				resources = append(resources, fileName)
			}

			if _, ok := networkInstance[nodeName]; ok {
				for id, niInfo := range networkInstance[nodeName] {
					log.Debugf("NetworkInstance Info: %s, %v %v", nodeName, id, niInfo)
					switch niInfo.Type {
					case "bridged":
						niInfo.SubInterfaces = append(niInfo.SubInterfaces, niCsiSubInterfaces[nodeName][id]...)
						log.Debugf("Subinterfaces bridged: %v", niInfo.SubInterfaces)
					case "routed":
						niInfo.SubInterfaces = append(niInfo.SubInterfaces, niCsiSubInterfaces[nodeName][id]...)
						// add all irb interfaces of this workload to the routed interface/IPvrf
						for _, irb := range niIrbSubInterfaces[nodeName] {
							niInfo.SubInterfaces = append(niInfo.SubInterfaces, irb...)
						}
					case "irb":
						niInfo.SubInterfaces = append(niInfo.SubInterfaces, niCsiSubInterfaces[nodeName][id]...)
						niInfo.SubInterfaces = append(niInfo.SubInterfaces, niIrbSubInterfaces[nodeName][id]...)
					}
					fileName := "network-instance-" + strconv.Itoa(id) + "-" + nodeName + ".yaml"
					p.WriteSrlNetworkInstance(&dirName,
						StringPtr(fileName),
						StringPtr(wlName+"-"+strconv.Itoa(niInfo.Evi)+"-network-instance"+"-"+nodeName),
						StringPtr(nodeName),
						niInfo)
					workloadresults.AppendNetworkInstance(nodeName, id, niInfo)
					resources = append(resources, fileName)

					if strings.Contains(niInfo.Name, "provisioning") {
						log.Debugf("Subinterfaces: %v", niInfo.SubInterfaces)
					}

					fileName = "network-instance-protocol-bgpvpn" + strconv.Itoa(id) + "-" + nodeName + ".yaml"
					p.WriteSrlNetworkInstanceBgpVpn(&dirName,
						StringPtr(fileName),
						StringPtr(wlName+"-"+strconv.Itoa(niInfo.Evi)+"-protocolbgpvpn"+"-"+nodeName),
						StringPtr(nodeName),
						niInfo)
					resources = append(resources, fileName)

					fileName = "network-instance-protocol-bgpevpn" + strconv.Itoa(id) + "-" + nodeName + ".yaml"
					p.WriteSrlNetworkInstanceBgpEvpn(&dirName,
						StringPtr(fileName),
						StringPtr(wlName+"-"+strconv.Itoa(niInfo.Evi)+"-protocolbgpevpn"+"-"+nodeName),
						StringPtr(nodeName),
						niInfo)
					resources = append(resources, fileName)

					fileName = "network-instance-protocol-linux" + strconv.Itoa(id) + "-" + nodeName + ".yaml"
					p.WriteSrlNetworkInstanceLinux(&dirName,
						StringPtr(fileName),
						StringPtr(wlName+"-"+strconv.Itoa(niInfo.Evi)+"-protocollinux"+"-"+nodeName),
						StringPtr(nodeName),
						niInfo)
					resources = append(resources, fileName)
				}
			}
			p.WriteKustomize(&dirName, StringPtr("kustomization.yaml"), resources)
		}
	}
	return kuztomizedirs, workloadresults
}
