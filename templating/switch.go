package templating

import (
	"encoding/json"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"

	"github.com/nokia-paco-automation/paco-parser/parser"
	"github.com/nokia-paco-automation/paco-parser/types"
	"github.com/stoewer/go-strcase"

	log "github.com/sirupsen/logrus"
)

func ProcessSwitchTemplates(wr *types.WorkloadResults, ir *types.InfrastructureResult, cg *types.ClientGroupResults, n map[string]*parser.Node, appConfig map[string]*parser.AppConfig, multusInfo map[string]*parser.MultusInfo, config *parser.Config, p *parser.Parser) map[string]string {
	log.Infof("ProcessingSwitchTemplates")

	templatenodes := map[string]*TemplateNode{}

	bgp_later := generateLoop(p, ir.IslSubInterfaces, wr, templatenodes)

	// routing policy & systeminterfaces
	for nodename, node := range n {
		if *node.Kind != "srl" {
			continue
		}
		// already add the node to the map
		templatenode := NewTemplateNode()
		templatenodes[nodename] = templatenode

		// routing policies
		conf := processRoutingPolicy(ir.RoutingPolicy)
		templatenode.SetRoutingPolicy(conf)

		// system interfaces
		for _, sysinterf := range ir.SystemInterfaces {
			conf := processInterface(nodename, sysinterf)
			templatenode.AddInterface(sysinterf.Name, conf)
		}

		// esis
		for _, esis := range cg.Esis {
			for _, esi := range esis {
				conf := processEsi(esi)
				templatenode.AddEsi(conf)
			}
		}

		// figure out gw interfaces
		gwinterfaces := []string{}
		for _, y := range cg.ClientInterfaces[nodename] {
			for _, v := range y {
				if v.Kind == "gw" {
					gwinterfaces = append(gwinterfaces, v.Name)
				}
			}
		}

		// iterating over network instance
		for _, netwinst := range wr.NetworkInstances[nodename] {
			// looking for infrastructure nwinstance
			if strings.Contains(netwinst.Name, "infrastructure") {
				// iterating over infra nw instances sub interfaces
				for _, subif := range netwinst.SubInterfaces {
					// iterating over gwinterfaces
					for _, gwifnames := range gwinterfaces {
						// chicking if there is a subinterface for that very interface
						if subif.InterfaceRealName == gwifnames {
							ir.DefaultNetworkInstances[nodename].SubInterfaces = append(ir.DefaultNetworkInstances[nodename].SubInterfaces, subif)
						}
					}
				}
			}
		}

		// Default network instance
		conf = processNetworkInstanceDefault(ir.DefaultNetworkInstances[nodename], ir.DefaultProtocolBGP[nodename])
		templatenodes[nodename].AddNetworkInstance("default", conf)

	}

	// process subinterfaces on a per interface basis
	for nodename, syssubifs := range ir.SystemSubInterfaces {
		for _, syssubif := range syssubifs {
			conf := processSrlSubInterface(nodename, syssubif.InterfaceRealName, syssubif)
			templatenodes[nodename].AddSubInterface(syssubif.InterfaceRealName, syssubif.VlanID, conf)
		}
	}

	// Clientinterfaces
	for nodename, data := range cg.ClientInterfaces {
		for _, interfs := range data {
			for _, interf := range interfs {
				conf := processInterface(nodename, interf)
				templatenodes[nodename].AddInterface(interf.Name, conf)
			}
		}
	}

	//clientsubinterfaces
	for nodename, clientinterfaces := range wr.ClientSubInterfaces {
		for _, clientsubifs := range clientinterfaces {
			for _, clientsubif := range clientsubifs {

				for _, y := range cg.ClientInterfaces[nodename] {
					for _, v := range y {
						if v.Name == clientsubif.InterfaceRealName {
							clientsubif.VlanTagging = v.VlanTagging
						}
					}
				}

				conf := processSrlSubInterface(nodename, clientsubif.InterfaceRealName, clientsubif)
				templatenodes[nodename].AddSubInterface(clientsubif.InterfaceRealName, clientsubif.VlanID, conf)
			}
		}
	}

	// Infrastructure Interfaces
	// BFD Infra interfaces
	for nodename, nodeInterfaces := range ir.IslInterfaces {
		for _, interf := range nodeInterfaces {
			conf := processInterface(nodename, interf)
			templatenodes[nodename].AddInterface(interf.Name, conf)
		}
	}

	//vxlaninterfaces
	for nodename, vxinterf := range wr.VxlanInterfaces {
		conf := processVxlanInterfaces(ir.TunnelInterfaces[0].Name, vxinterf)
		templatenodes[nodename].SetVxlanInterface(conf)
	}

	// process subinterfaces on a per interface basis
	for nodename, ifs := range ir.IslSubInterfaces {
		for ifname, srlsubifs := range ifs {
			for _, srlsubif := range srlsubifs {
				conf := processSrlSubInterface(nodename, ifname, srlsubif)
				templatenodes[nodename].AddSubInterface(srlsubif.InterfaceRealName, srlsubif.VlanID, conf)
				conf = processBfdInterface(srlsubif)
				templatenodes[nodename].AddBfd(conf)
			}
		}
	}

	// Networkinstance attached subinterfaces
	for nodename, nodeni := range wr.NetworkInstances {
		for niid, networkinstance := range nodeni {
			for index, subif := range networkinstance.SubInterfaces {
				_ = index
				_ = subif
				_ = niid
				_ = nodename
				conf := processSrlSubInterface(nodename, subif.InterfaceRealName, subif)
				templatenodes[nodename].AddSubInterface(subif.InterfaceRealName, subif.VlanID, conf)
			}
		}
	}

	// // lag subinterfaces
	// for nodename, nodeni := range wr.NetworkInstances {
	// 	for _, networkinstance := range nodeni {
	// 		if networkinstance.Kind == "mac-vrf" {
	// 			for _, subif := range networkinstance.SubInterfaces {
	// 				if subif.Kind == "bridged" {
	// 					conf := processInterface(nodename, &types.K8ssrlinterface{Name: subif.InterfaceRealName, VlanTagging: subif.VlanTagging})
	// 					templatenodes[nodename].AddInterface(subif.InterfaceRealName, conf)

	// 					conf = processSrlSubInterface(nodename, subif.InterfaceRealName, subif)
	// 					templatenodes[nodename].AddSubInterface(subif.InterfaceRealName, subif.VlanID, conf)
	// 				}
	// 			}
	// 		}
	// 	}
	// }

	// irbsub interfaces
	// irb bfd
	for nodename, irbsubifs := range wr.IrbSubInterfaces {
		for _, irbsubif := range irbsubifs {
			conf := processIrbSubInterfaces(irbsubif)
			templatenodes[nodename].AddSubInterface(irbsubif.InterfaceRealName, irbsubif.VlanID, conf)
			if strings.Contains(irbsubif.NwType, "sriov") {
				conf = processBfdIrb(irbsubif)
				templatenodes[nodename].AddBfd(conf)
			}
		}
	}
	// first run before "processAppConfBgp()"
	// networkinstance
	for nodename, data := range wr.NetworkInstances {
		for _, networkinstance := range data {
			conf := processNetworkInstance(networkinstance)
			templatenodes[nodename].AddNetworkInstance(networkinstance.Name, conf)
		}
	}

	processAppConfBgp(appConfig, wr, ir, multusInfo, templatenodes, config, bgp_later)

	// Second run after "processAppConfBgp()"
	// networkinstance
	for nodename, data := range wr.NetworkInstances {
		for _, networkinstance := range data {
			conf := processNetworkInstance(networkinstance)
			templatenodes[nodename].AddNetworkInstance(networkinstance.Name, conf)
		}
	}

	// BGP for NetworkInstances without Loopbacks
	BgpForNonLoopbackNIs(config, templatenodes, ir.DefaultProtocolBGP, wr, bgp_later)

	// bgp
	for nodename, bgp := range ir.DefaultProtocolBGP {
		conf := processBgp(bgp)
		templatenodes[nodename].AddBgp(bgp.NetworkInstanceName, conf)
	}

	// Static routes with Nexthop groups
	// and Next-Hop-Groups
	gsr := processAppConfSrNhg(appConfig, ir, wr)
	for nodename, entry := range gsr.Data {
		for instancename, staticroutes := range entry {
			for _, staticroute := range staticroutes {
				srconf := processStaticRoute(staticroute)
				templatenodes[nodename].AddStaticRoute(instancename, srconf)
				nhgconf := processNextHopGroup(staticroute.NHGroup)
				templatenodes[nodename].AddNextHopGroup(instancename, nhgconf)
			}
		}
	}

	// store pretty printed config per node in the map
	// and return that result
	perNodeConfig := map[string]string{}
	for name, node := range templatenodes {
		var tmp interface{}
		err := json.Unmarshal([]byte(node.MergeConfig()), &tmp)
		if err != nil {
			log.Fatalf("%v", err)
		}
		indentresult, err := json.MarshalIndent(tmp, "", "  ")
		if err != nil {
			log.Fatalf("%v", err)
		}
		perNodeConfig[name] = string(indentresult)
	}
	return perNodeConfig
}

func processNetworkInstanceDefault(defaultinstance *types.K8ssrlNetworkInstance, defaultprotobgp *types.K8ssrlprotocolsbgp) string {
	templateFile := path.Join("templates", "switch", "networkinstancedefault.tmpl")
	parameter := struct {
		NetworkInstance *types.K8ssrlNetworkInstance
		BGPProtocol     *types.K8ssrlprotocolsbgp
	}{NetworkInstance: defaultinstance, BGPProtocol: defaultprotobgp}
	return GeneralTemplateProcessing(templateFile, "networkinstancedefault", parameter)
}

func processNextHopGroup(nhg *types.NHGroup) string {
	nhgj := NewJsonMerger()

	templateFile := path.Join("templates", "switch", "nhgroup.tmpl")
	nhgj.Merge([]byte(GeneralTemplateProcessing(templateFile, "nhgroup", nhg)))

	nhgentryArrB := NewJsonArrayBuilder()
	for _, nhgentry := range nhg.Entries {
		entryconf := processNextHopGroupEntry(nhgentry)
		nhgentryArrB.AddEntry(entryconf)
	}
	nhgj.Merge([]byte(nhgentryArrB.ToStringObj("nexthop")))
	return nhgj.ToString()
}

func processNextHopGroupEntry(nhge *types.NHGroupEntry) string {
	templateFile := path.Join("templates", "switch", "nhgroupentry.tmpl")
	return GeneralTemplateProcessing(templateFile, "nhgroupentry", nhge)
}

// func BGPPeerMapToString(peerinfo map[int]*parser.BGPPeerInfo) string {
// 	result := ""
// 	counter := 0
// 	for _, peer := range peerinfo {
// 		if counter > 0 {
// 			result += " | "
// 		}
// 		counter++
// 		result += fmt.Sprintf("Peer: %s, AS: %d", *peer.IP, *peer.AS)
// 	}

// 	return result
// }

// this is to extract the static route config as well as the next-hop-groups.
func processAppConfSrNhg(appconf map[string]*parser.AppConfig, ir *types.InfrastructureResult, wr *types.WorkloadResults) *GlobalStaticRoutes {
	globalStaticRoutes := NewGlobalStaticRoutes()

	for cnfName, cnf := range appconf {
		if cnfName != "upf" && cnfName != "smf" {
			continue
		}
		for wlName, workloads := range cnf.Networks {
			generateLlbBgpRoutes(workloads, wlName, cnfName, globalStaticRoutes, wr)
			generateLlbInterfaceRoutes(workloads, wlName, cnfName, globalStaticRoutes, wr)
			if cnfName == "upf" {
				generateLmgRoutes(workloads, wlName, cnfName, globalStaticRoutes, wr)
			}
		}
	}
	return globalStaticRoutes
}

type AppConfBgpLoTempStoreEntry struct {
	lo  map[string]*types.K8ssrlsubinterface
	bgp map[string]*types.K8ssrlprotocolsbgp
	vid int
}

func processAppConfBgp(appconf map[string]*parser.AppConfig, wr *types.WorkloadResults, ir *types.InfrastructureResult, multusInfo map[string]*parser.MultusInfo, templatenodes map[string]*TemplateNode, config *parser.Config, bgp_later []*BGPLaterAdd) {

	appConfBgpLoStore := map[string]*AppConfBgpLoTempStoreEntry{} // niname indexed

	dcgw_needs_add := map[string]map[string]map[string]bool{} // nodename, niname, ipversion -> bool

	for cnfName, cnf := range appconf {
		if cnfName != "upf" && cnfName != "smf" {
			continue
		}

		for wlName, workloads := range cnf.Networks {
			for _, bar := range workloads[0][0]["loopback"]["bgpLbk"] {
				var perNodeLo map[string]*types.K8ssrlsubinterface
				var perNodeBgp map[string]*types.K8ssrlprotocolsbgp

				mywlname := wlnametranslate(wlName, multusInfo)
				mywlname = strcase.KebabCase(strings.Replace(mywlname, "multus-", "", 1))
				niName := mywlname + "-ipvrf-itfce-" + strconv.Itoa(*bar.VlanID)
				mywlname = mywlname + "-" + cnfName

				if _, ok := appConfBgpLoStore[niName]; ok {
					perNodeLo = appConfBgpLoStore[niName].lo
					perNodeBgp = appConfBgpLoStore[niName].bgp
				} else {
					perNodeLo = map[string]*types.K8ssrlsubinterface{}  // nodename indexed
					perNodeBgp = map[string]*types.K8ssrlprotocolsbgp{} // nodename indexed
					appConfBgpLoStore[niName] = &AppConfBgpLoTempStoreEntry{lo: perNodeLo, bgp: perNodeBgp, vid: *bar.VlanID}
				}

				for _, nodename := range filterNodesContainingNI(niName, templatenodes) {
					if _, ok := perNodeLo[nodename]; !ok {
						perNodeLo[nodename] = &types.K8ssrlsubinterface{
							InterfaceRealName:  "lo0",
							InterfaceShortName: "lo0",
							VlanTagging:        false,
							VlanID:             strconv.Itoa(*bar.VlanID),
							Kind:               "loopback",
							//IPv4Prefix:         "",
							//IPv6Prefix:         "",
						}
					}
					if _, ok := perNodeBgp[nodename]; !ok {
						perNodeBgp[nodename] = &types.K8ssrlprotocolsbgp{
							NetworkInstanceName: niName,
							//AS:                  *y.AS,
							//RouterID:            *y.IP,
							PeerGroups: []*types.PeerGroup{
								{Protocols: []string{"bgp"}, Name: "DCGW", PolicyName: "bgp_export_policy_default"},
							},
						}
					}
					// check if the peergroup needs to be added
					peergrouop_exists := false
					for _, entry := range perNodeBgp[nodename].PeerGroups {
						if entry.Name == mywlname {
							peergrouop_exists = true
						}
					}
					// add peergroup if required
					if !peergrouop_exists {
						perNodeBgp[nodename].PeerGroups = append(perNodeBgp[nodename].PeerGroups, &types.PeerGroup{Protocols: []string{"bgp"}, Name: mywlname, PolicyName: "bgp_export_policy_default"})
					}
				}

				for _, y := range bar.IPv4BGPPeers {

					//fmt.Printf("node: %s, wlname: %s, PeerIP: %s, PeerAS: %d, LocalAddress: %s, LocalAS: %d, vlanid: %d\n", niName, wlName, *bar.IPv4BGPAddress, *bar.AS, *y.IP, *y.AS, *bar.VlanID)
					//dcgw_ip := wr.NetworkInstances[strconv.Itoa(*bar.VlanID)]
					nodename := y.Node

					// init the storage for remembering if dcgw was already added as neighbor (per ip version)
					if _, ok := dcgw_needs_add[nodename]; !ok {
						dcgw_needs_add[nodename] = map[string]map[string]bool{}
					}
					if _, ok := dcgw_needs_add[nodename][niName]; !ok {
						dcgw_needs_add[nodename][niName] = map[string]bool{}
						dcgw_needs_add[nodename][niName]["v4"] = true
						dcgw_needs_add[nodename][niName]["v6"] = true
					}

					losubif := perNodeLo[nodename]
					losubif.IPv4Prefix = *y.IP + "/32"

					foo := perNodeBgp[nodename]

					if dcgw_needs_add[nodename][niName]["v4"] {

						transportIP := wr.NetworkInstances[nodename][*bar.VlanID].SubInterfaces[0].IPv4Prefix

						ip, ipnet, err := net.ParseCIDR(transportIP)
						peerIP := incrementIP(ip)
						_ = ipnet
						_ = ip
						_ = err

						dcgwNbrv4 := &types.Neighbor{
							PeerIP:    peerIP.String(),
							PeerAS:    searchLocalASInConfig(config, *bar.VlanID),
							PeerGroup: "DCGW",
							//LocalAS:          *y.AS,
							TransportAddress: ip.String(),
						}
						foo.Neighbors = append(foo.Neighbors, dcgwNbrv4)
						dcgw_needs_add[nodename][niName]["v4"] = false
					}

					Nbrv4 := &types.Neighbor{
						PeerIP:    *bar.IPv4BGPAddress,
						PeerAS:    *bar.AS,
						PeerGroup: mywlname,
						//LocalAS:          *y.AS,
						TransportAddress: *y.IP,
					}

					foo.AS = *config.Infrastructure.Protocols.OverlayAs
					foo.RouterID = *y.IP

					foo.Neighbors = append(foo.Neighbors, Nbrv4)

				}
				for _, y := range bar.IPv6BGPPeers {

					//fmt.Printf("node: %s, wlname: %s, PeerIP: %s, PeerAS: %d, LocalAddress: %s, LocalAS: %d, vlanid: %d\n", niName, wlName, *bar.IPv4BGPAddress, *bar.AS, *y.IP, *y.AS, *bar.VlanID)
					//dcgw_ip := wr.NetworkInstances[strconv.Itoa(*bar.VlanID)]

					nodename := y.Node

					// init the storage for remembering if dcgw was already added as neighbor (per ip version)
					if _, ok := dcgw_needs_add[nodename]; !ok {
						dcgw_needs_add[nodename] = map[string]map[string]bool{}
					}
					if _, ok := dcgw_needs_add[nodename][niName]; !ok {
						dcgw_needs_add[nodename][niName] = map[string]bool{}
						dcgw_needs_add[nodename][niName]["v4"] = true
						dcgw_needs_add[nodename][niName]["v6"] = true
					}

					losubif := perNodeLo[nodename]
					losubif.IPv6Prefix = *y.IP + "/128"

					foo := perNodeBgp[nodename]

					if dcgw_needs_add[nodename][niName]["v6"] {

						transportIP := wr.NetworkInstances[nodename][*bar.VlanID].SubInterfaces[0].IPv6Prefix

						ip, ipnet, err := net.ParseCIDR(transportIP)
						peerIP := incrementIP(ip)
						_ = ipnet
						_ = ip
						_ = err

						dcgwNbrv6 := &types.Neighbor{
							PeerIP:    peerIP.String(),
							PeerAS:    searchLocalASInConfig(config, *bar.VlanID),
							PeerGroup: "DCGW",
							//LocalAS:          *y.AS,
							TransportAddress: ip.String(),
						}
						foo.Neighbors = append(foo.Neighbors, dcgwNbrv6)

						dcgw_needs_add[nodename][niName]["v6"] = false
					}
					Nbrv6 := &types.Neighbor{
						PeerIP:    *bar.IPv6BGPAddress,
						PeerAS:    *bar.AS,
						PeerGroup: mywlname,
						//LocalAS:          *y.AS,
						TransportAddress: *y.IP,
					}

					foo.AS = *config.Infrastructure.Protocols.OverlayAs
					foo.Neighbors = append(foo.Neighbors, Nbrv6)
				}
			}
		}
	}
	for niName, appconfbgpstoreentry := range appConfBgpLoStore {
		perNodeLo := appconfbgpstoreentry.lo
		perNodeBgp := appconfbgpstoreentry.bgp
		vid := appconfbgpstoreentry.vid

		for nodename, lo0conf := range perNodeLo {
			templatenodes[nodename].AddSubInterface(lo0conf.InterfaceShortName, lo0conf.VlanID, processSrlSubInterface(nodename, lo0conf.InterfaceShortName, lo0conf))
			if !checkIfSubIFAlreadyExists(wr.NetworkInstances[nodename][vid].SubInterfaces, lo0conf.InterfaceRealName, lo0conf.VlanID) {
				wr.NetworkInstances[nodename][vid].SubInterfaces = append(wr.NetworkInstances[nodename][vid].SubInterfaces, lo0conf)
			}
		}
		for nodename, bgpconf := range perNodeBgp {
			for _, bgp_later_entry := range bgp_later {
				if bgp_later_entry.nivid == vid && bgp_later_entry.nodename == nodename {
					bgpconf.Neighbors = append(bgpconf.Neighbors, bgp_later_entry.bgpconf...)
					bgpconf.PeerGroups = append(bgpconf.PeerGroups, &types.PeerGroup{Protocols: []string{"bgp"}, Name: "LOOP", PolicyName: "bgp_export_policy_default"})
				}
			}
			templatenodes[nodename].AddBgp(niName, processBgp(bgpconf))
		}
	}
}

func BgpForNonLoopbackNIs(config *parser.Config, templatenodes map[string]*TemplateNode, defProtoBgp map[string]*types.K8ssrlprotocolsbgp, wr *types.WorkloadResults, bgp_later []*BGPLaterAdd) {
	for wlname, wl := range config.Workloads {
		wlname = strings.TrimPrefix(wlname, "multus-")
		if len(wl["servers"].Loopbacks) > 0 {
			// skip NIs with loopbacks configure, they are handled in 'processAppConfBgp(...)'
			continue
		}
		if _, ok := wl["dcgw-grp1"]; !ok {
			continue
		}

		vlanid := *wl["dcgw-grp1"].Itfces["itfce"].VlanID
		niName := wlname + "-ipvrf-itfce-" + strconv.Itoa(vlanid)

		for _, nodename := range filterNodesContainingNI(niName, templatenodes) {

			ip, ipnet, err := net.ParseCIDR(wr.NetworkInstances[nodename][vlanid].SubInterfaces[0].IPv4Prefix)
			peerIP := incrementIP(ip)
			_ = ipnet
			_ = ip
			_ = err

			foo := &types.K8ssrlprotocolsbgp{
				NetworkInstanceName: niName,
				AS:                  *config.Infrastructure.Protocols.OverlayAs,
				RouterID:            defProtoBgp[nodename].RouterID,
				PeerGroups: []*types.PeerGroup{
					{Protocols: []string{"bgp"}, Name: "DCGW", PolicyName: "bgp_export_policy_default"},
				},
			}

			neighborv4 := &types.Neighbor{
				PeerIP:    peerIP.String(),
				PeerAS:    searchLocalASInConfig(config, vlanid),
				PeerGroup: "DCGW",
				//LocalAS:          defProtoBgp[nodename].AS,
				TransportAddress: ip.String(),
			}

			foo.Neighbors = append(foo.Neighbors, neighborv4)

			ip, ipnet, err = net.ParseCIDR(wr.NetworkInstances[nodename][vlanid].SubInterfaces[0].IPv6Prefix)
			peerIP = incrementIP(ip)
			_ = ipnet
			_ = ip
			_ = err

			neighborv6 := &types.Neighbor{
				PeerIP:    peerIP.String(),
				PeerAS:    searchLocalASInConfig(config, vlanid),
				PeerGroup: "DCGW",
				//LocalAS:          defProtoBgp[nodename].AS,
				TransportAddress: ip.String(),
			}

			foo.Neighbors = append(foo.Neighbors, neighborv6)

			for _, bgp_later_entry := range bgp_later {
				if bgp_later_entry.nivid == vlanid && bgp_later_entry.nodename == nodename {
					foo.Neighbors = append(foo.Neighbors, bgp_later_entry.bgpconf...)
					foo.PeerGroups = append(foo.PeerGroups, &types.PeerGroup{Protocols: []string{"bgp"}, Name: "LOOP", PolicyName: "bgp_export_policy_default"})
				}
			}
			templatenodes[nodename].AddBgp(niName, processBgp(foo))
		}
	}
}

func searchLocalASInConfig(config *parser.Config, vlanid int) uint32 {
	for _, wl := range config.Workloads {
		if dcgwgrp, ok := wl["dcgw-grp1"]; ok {
			if *dcgwgrp.Itfces["itfce"].VlanID == vlanid {
				return *dcgwgrp.Itfces["itfce"].PeerAS
			}
		}
	}
	return 0
}

func checkIfSubIFAlreadyExists(subifs []*types.K8ssrlsubinterface, name string, id string) bool {
	for _, entry := range subifs {
		if entry.InterfaceRealName == name && id == entry.VlanID {
			return true
		}
	}
	return false
}

func filterNodesContainingNI(name string, templatenodes map[string]*TemplateNode) []string {
	result := []string{}
	for nodename, y := range templatenodes {
		for instancename, _ := range y.NetworkInstances {
			if name == instancename {
				result = append(result, nodename)
			}
		}
	}
	return result
}

func wlnametranslate(name string, data map[string]*parser.MultusInfo) string {
	if _, ok := data[name]; !ok {
		log.Error("FAIL")
	}
	return *data[name].WorkloadName
}

func generateLmgRoutes(workloads map[int]map[int]map[string]map[string][]*parser.RenderedNetworkInfo, wlName string, cnfName string, globalStaticRoutes *GlobalStaticRoutes, wr *types.WorkloadResults) {
	// loop over lmg's
	for i := 1; i < len(workloads); i++ {
		if len(workloads[i][0]["loopback"]["lmgLbk"]) == 0 {
			continue
		}
		destPrefix := *workloads[i][0]["loopback"]["lmgLbk"][0].Ipv4Addresses[0].IPAddress + "/32"
		lmgNo := i - 1
		log.Debugf(fmt.Sprintf("SR - LMG %d - WLName: %s, prefix: %s/32", lmgNo, wlName, destPrefix))
		// loop over switch
		for x := 1; x < len(workloads[i]); x++ {
			lmgInterfInfoArr := workloads[i][x]["itfce"]["intIP"][0]

			nhindex := x
			nextHop := *lmgInterfInfoArr.Ipv4Addresses[0].IPAddress
			leafnode := *lmgInterfInfoArr.Target
			sourceIP := lmgInterfInfoArr.Ipv4GwPerWl[x][0]

			log.Debugf(fmt.Sprintf("Lmg %d loopback NH - WLName: %s, NH: %s, Targetleaf: %s, BFD SRC: %s", lmgNo, wlName, nextHop, leafnode, sourceIP))

			sr := types.NewStaticRouteNHG(destPrefix)
			sr.RType = "lmg"
			sr.CnfName = cnfName
			sr.IpVersion = "v4"
			sr.WlName = wlName
			sr.TargetLeaf = leafnode
			sr.SetNHGroupName(fmt.Sprintf("%s-%s-lmg%d", wlName, cnfName, lmgNo))
			nhgentry := &types.NHGroupEntry{
				Index:     nhindex,
				NHIp:      nextHop,
				LocalAddr: sourceIP,
			}
			sr.AddNHGroupEntry(nhgentry)

			irbintef := findRelatedIRBv4(wr.IrbSubInterfaces, sourceIP)
			networkInstance := findNetworkInstanceOfIrb(wr.NetworkInstances, irbintef)
			sr.VlanID = networkInstance.networkInstance.Evi
			globalStaticRoutes.addEntry(leafnode, networkInstance.networkInstance.Name, sr)
		}

		// v6 start
		destPrefix = *workloads[i][0]["loopback"]["lmgLbk"][0].Ipv6Addresses[0].IPAddress + "/128"
		log.Debugf(fmt.Sprintf("SR - LMG %d - WLName: %s, prefix: %s/128", lmgNo, wlName, destPrefix))
		// loop over switch
		for x := 1; x < len(workloads[i]); x++ {
			lmgInterfInfoArr := workloads[i][x]["itfce"]["intIP"][0]

			nhindex := x
			nextHop := *lmgInterfInfoArr.Ipv6Addresses[0].IPAddress
			leafnode := *lmgInterfInfoArr.Target
			sourceIP := lmgInterfInfoArr.Ipv6GwPerWl[x][0]

			log.Debugf(fmt.Sprintf("Lmg %d loopback NH - WLName: %s, NH: %s, Targetleaf: %s, BFD SRC: %s", lmgNo, wlName, nextHop, leafnode, sourceIP))

			sr := types.NewStaticRouteNHG(destPrefix)
			sr.CnfName = cnfName
			sr.RType = "lmg"
			sr.IpVersion = "v6"
			sr.WlName = wlName
			sr.TargetLeaf = leafnode
			sr.SetNHGroupName(fmt.Sprintf("%s-%s-lmg%d-v6", wlName, cnfName, lmgNo))
			nhgentry := &types.NHGroupEntry{
				Index:     nhindex,
				NHIp:      nextHop,
				LocalAddr: sourceIP,
			}
			sr.AddNHGroupEntry(nhgentry)

			irbintef := findRelatedIRBv6(wr.IrbSubInterfaces, sourceIP)
			networkInstance := findNetworkInstanceOfIrb(wr.NetworkInstances, irbintef)
			sr.VlanID = networkInstance.networkInstance.Evi
			globalStaticRoutes.addEntry(leafnode, networkInstance.networkInstance.Name, sr)
		}
	}
}

func generateLlbInterfaceRoutes(workloads map[int]map[int]map[string]map[string][]*parser.RenderedNetworkInfo, wlName string, cnfName string, globalStaticRoutes *GlobalStaticRoutes, wr *types.WorkloadResults) {
	if len(workloads[0][0]["loopback"]["llbLbk"]) == 0 {
		return
	}
	llbLoopbackInfoArr := workloads[0][0]["loopback"]["llbLbk"][0]

	for llbLoopbackIndex, llbLoopbackIPAddress := range llbLoopbackInfoArr.Ipv4Addresses {
		destPrefix := *llbLoopbackIPAddress.IPAddress + "/32"
		groupindex := llbLoopbackIndex
		log.Debugf(fmt.Sprintf("SR - LLB %d - WLName: %s, prefix: %s/32", groupindex, wlName, destPrefix))

		for x := 1; x < len(workloads[0]); x++ {
			llbInterfInfoArr := workloads[0][x]["itfce"]["intIP"][0]

			nhindex := x
			nextHop := *llbInterfInfoArr.Ipv4Addresses[llbLoopbackIndex].IPAddress
			sourceIP := llbInterfInfoArr.Ipv4GwPerWl[x][llbLoopbackIndex]
			leafNode := *llbInterfInfoArr.Target

			log.Debugf(fmt.Sprintf("LLb %d loopback NH - WLName: %s, NH: %s, Targetleaf: %s, BFD SRC: %s", groupindex, wlName, nextHop, leafNode, sourceIP))

			sr := types.NewStaticRouteNHG(destPrefix)
			sr.CnfName = cnfName
			sr.RType = "llb"
			sr.IpVersion = "v4"
			sr.WlName = wlName
			sr.TargetLeaf = leafNode
			sr.SetNHGroupName(fmt.Sprintf("%s-%s-llb%d", wlName, cnfName, groupindex))

			nhgentry := &types.NHGroupEntry{
				Index:     nhindex,
				NHIp:      nextHop,
				LocalAddr: sourceIP,
			}
			sr.AddNHGroupEntry(nhgentry)

			irbintef := findRelatedIRBv4(wr.IrbSubInterfaces, sourceIP)
			networkInstance := findNetworkInstanceOfIrb(wr.NetworkInstances, irbintef)
			sr.VlanID = networkInstance.networkInstance.Evi
			globalStaticRoutes.addEntry(leafNode, networkInstance.networkInstance.Name, sr)
		}
	}
	for llbLoopbackIndex, llbLoopbackIPAddress := range llbLoopbackInfoArr.Ipv6Addresses {
		destPrefix := *llbLoopbackIPAddress.IPAddress + "/128"
		groupindex := llbLoopbackIndex
		log.Debugf(fmt.Sprintf("SR - LLB %d - WLName: %s, prefix: %s/128", groupindex, wlName, destPrefix))

		for x := 1; x < len(workloads[0]); x++ {
			llbInterfInfoArr := workloads[0][x]["itfce"]["intIP"][0]

			nhindex := x
			nextHop := *llbInterfInfoArr.Ipv6Addresses[llbLoopbackIndex].IPAddress
			sourceIP := llbInterfInfoArr.Ipv6GwPerWl[x][llbLoopbackIndex]
			leafNode := *llbInterfInfoArr.Target

			log.Debugf(fmt.Sprintf("LLb %d loopback NH - WLName: %s, NH: %s, Targetleaf: %s, BFD SRC: %s", groupindex, wlName, nextHop, leafNode, sourceIP))

			sr := types.NewStaticRouteNHG(destPrefix)
			sr.CnfName = cnfName
			sr.RType = "llb"
			sr.IpVersion = "v6"
			sr.WlName = wlName
			sr.TargetLeaf = leafNode
			sr.SetNHGroupName(fmt.Sprintf("%s-%s-llb%d-v6", wlName, cnfName, groupindex))

			nhgentry := &types.NHGroupEntry{
				Index:     nhindex,
				NHIp:      nextHop,
				LocalAddr: sourceIP,
			}
			sr.AddNHGroupEntry(nhgentry)

			irbintef := findRelatedIRBv6(wr.IrbSubInterfaces, sourceIP)
			networkInstance := findNetworkInstanceOfIrb(wr.NetworkInstances, irbintef)
			sr.VlanID = networkInstance.networkInstance.Evi
			globalStaticRoutes.addEntry(leafNode, networkInstance.networkInstance.Name, sr)
		}
	}
}

func generateLlbBgpRoutes(workloads map[int]map[int]map[string]map[string][]*parser.RenderedNetworkInfo, wlName string, cnfName string, globalStaticRoutes *GlobalStaticRoutes, wr *types.WorkloadResults) {

	if len(workloads[0][0]["loopback"]["bgpLbk"]) == 0 {
		return
	}

	bgpLoopbackInfoArr := workloads[0][0]["loopback"]["bgpLbk"][0]

	for x := 1; x < len(workloads[0]); x++ {
		llbInterfInfoArr := workloads[0][x]["itfce"]["intIP"][0]

		destPrefix := *bgpLoopbackInfoArr.Ipv4Addresses[0].IPAddress + "/32"

		log.Debugf(fmt.Sprintf("SR - BGP - WLName: %s, prefix: %s/32", wlName, destPrefix))

		sr := types.NewStaticRouteNHG(destPrefix)
		sr.CnfName = cnfName
		sr.RType = "llbbgp"
		sr.IpVersion = "v4"
		sr.WlName = wlName
		sr.TargetLeaf = ""
		sr.SetNHGroupName(fmt.Sprintf("%s-%s-llb-bgp", wlName, cnfName))

		for ipIndex, ipAddress := range llbInterfInfoArr.Ipv4Addresses {
			nextHop := *ipAddress.IPAddress
			nhIndex := ipIndex
			leafName := *llbInterfInfoArr.Target
			sourceIP := llbInterfInfoArr.Ipv4GwPerWl[x][ipIndex]

			log.Debugf(fmt.Sprintf("BGP loopback NH - WLName: %s, NH: %s, Targetleaf: %s, BFD SRC: %s", wlName, nextHop, leafName, sourceIP))

			nhgentry := &types.NHGroupEntry{
				Index:     nhIndex,
				NHIp:      nextHop,
				LocalAddr: sourceIP,
			}
			sr.AddNHGroupEntry(nhgentry)
		}
		localIRBIP := llbInterfInfoArr.Ipv4GwPerWl[x][0]
		irbintef := findRelatedIRBv4(wr.IrbSubInterfaces, localIRBIP)
		networkInstance := findNetworkInstanceOfIrb(wr.NetworkInstances, irbintef)
		sr.VlanID = networkInstance.networkInstance.Evi
		globalStaticRoutes.addEntry(*llbInterfInfoArr.Target, networkInstance.networkInstance.Name, sr)
	}
	for x := 1; x < len(workloads[0]); x++ {
		llbInterfInfoArr := workloads[0][x]["itfce"]["intIP"][0]

		destPrefix := *bgpLoopbackInfoArr.Ipv6Addresses[0].IPAddress + "/128"

		log.Debugf(fmt.Sprintf("SR - BGP - WLName: %s, prefix: %s/128", wlName, destPrefix))

		sr := types.NewStaticRouteNHG(destPrefix)
		sr.CnfName = cnfName
		sr.RType = "llbbgp"
		sr.IpVersion = "v6"
		sr.WlName = wlName
		sr.TargetLeaf = ""
		sr.SetNHGroupName(fmt.Sprintf("%s-%s-llb-bgp-v6", wlName, cnfName))

		for ipIndex, ipAddress := range llbInterfInfoArr.Ipv6Addresses {
			nextHop := *ipAddress.IPAddress
			nhIndex := ipIndex
			leafName := *llbInterfInfoArr.Target
			sourceIP := llbInterfInfoArr.Ipv6GwPerWl[x][ipIndex]

			log.Debugf(fmt.Sprintf("BGP loopback NH - WLName: %s, NH: %s, Targetleaf: %s, BFD SRC: %s", wlName, nextHop, leafName, sourceIP))

			nhgentry := &types.NHGroupEntry{
				Index:     nhIndex,
				NHIp:      nextHop,
				LocalAddr: sourceIP,
			}
			sr.AddNHGroupEntry(nhgentry)
		}
		localIRBIP := llbInterfInfoArr.Ipv6GwPerWl[x][0]
		irbintef := findRelatedIRBv6(wr.IrbSubInterfaces, localIRBIP)
		networkInstance := findNetworkInstanceOfIrb(wr.NetworkInstances, irbintef)
		sr.VlanID = networkInstance.networkInstance.Evi
		globalStaticRoutes.addEntry(*llbInterfInfoArr.Target, networkInstance.networkInstance.Name, sr)
	}
}

func generateLoop(p *parser.Parser, subifs map[string]map[string][]*types.K8ssrlsubinterface, wr *types.WorkloadResults, templatenodes map[string]*TemplateNode) []*BGPLaterAdd {

	bgplater := []*BGPLaterAdd{}

	config := p.Config
	//var ipv4prefix, ipv6prefix string
	netwInfo := config.Infrastructure.Networks["loop"]

	var infraNIName string
	var infraVID int

	// determine name of Infrastructure NI
	for wlname, workload := range p.Config.Workloads {
		if strings.Contains(strings.ToLower(wlname), "infrastru") {
			infraNIName = wlname
			infraVID = *workload["dcgw-grp1"].Itfces["itfce"].VlanID
			break
		}
	}

	for wlname, workload := range p.Config.Workloads {
		_ = workload
		if !strings.Contains(wlname, "mgmt") && !strings.Contains(wlname, "sba") {
			continue
		}
		for _, l := range p.Links {
			if *l.Kind == "loop" {
				ipamName := "loop"
				var ipv4Cidr *string
				var ipv6Cidr *string
				for i := 0; i < len(netwInfo.Ipv4Cidr); i++ {
					ipv4Cidr = netwInfo.Ipv4Cidr[i]
					ipv6Cidr = netwInfo.Ipv6Cidr[i]
					if err := p.IPAM[ipamName].IPAMAllocateLinkPrefix(l, ipv4Cidr, ipv6Cidr); err != nil {
						log.Error(err)
					}
				}

				vlanid := *workload["dcgw-grp1"].Itfces["itfce"].VlanID

				csiA := &types.K8ssrlsubinterface{
					InterfaceRealName:  *l.A.RealName,
					InterfaceShortName: *l.A.ShortName,
					VlanTagging:        true,
					VlanID:             strconv.Itoa(vlanid),
					Kind:               "routed",
					IPv4Prefix:         *l.A.IPv4Address + "/" + strconv.Itoa(*l.A.IPv4PrefixLength),
					IPv6Prefix:         *l.A.IPv6Address + "/" + strconv.Itoa(*l.A.IPv6PrefixLength),
				}
				csiB := &types.K8ssrlsubinterface{
					InterfaceRealName:  *l.B.RealName,
					InterfaceShortName: *l.B.ShortName,
					VlanTagging:        true,
					VlanID:             strconv.Itoa(vlanid),
					Kind:               "routed",
					IPv4Prefix:         *l.B.IPv4Address + "/" + strconv.Itoa(*l.B.IPv4PrefixLength),
					IPv6Prefix:         *l.B.IPv6Address + "/" + strconv.Itoa(*l.B.IPv6PrefixLength),
				}

				subifs[*l.A.Node.ShortName][*l.A.RealName] = append(subifs[*l.A.Node.ShortName][*l.A.RealName], csiA)
				wr.NetworkInstances[*l.A.Node.ShortName][infraVID].SubInterfaces = append(wr.NetworkInstances[*l.A.Node.ShortName][infraVID].SubInterfaces, csiA)

				subifs[*l.B.Node.ShortName][*l.B.RealName] = append(subifs[*l.B.Node.ShortName][*l.B.RealName], csiB)
				wr.NetworkInstances[*l.B.Node.ShortName][vlanid].SubInterfaces = append(wr.NetworkInstances[*l.B.Node.ShortName][vlanid].SubInterfaces, csiB)

				NeighAv4 := []*types.Neighbor{
					{
						PeerIP:           *l.B.IPv4Address,
						PeerAS:           *p.Config.Infrastructure.Protocols.AsPoolLoop[1],
						PeerGroup:        "LOOP",
						LocalAS:          *p.Config.Infrastructure.Protocols.AsPoolLoop[0],
						TransportAddress: *l.A.IPv4Address,
					},
				}
				NeighBv4 := []*types.Neighbor{
					{
						PeerIP:           *l.A.IPv4Address,
						PeerAS:           *p.Config.Infrastructure.Protocols.AsPoolLoop[0],
						PeerGroup:        "LOOP",
						LocalAS:          *p.Config.Infrastructure.Protocols.AsPoolLoop[1],
						TransportAddress: *l.B.IPv4Address,
					},
				}
				NeighAv6 := []*types.Neighbor{
					{
						PeerIP:           *l.B.IPv6Address,
						PeerAS:           *p.Config.Infrastructure.Protocols.AsPoolLoop[1],
						PeerGroup:        "LOOP",
						LocalAS:          *p.Config.Infrastructure.Protocols.AsPoolLoop[0],
						TransportAddress: *l.A.IPv6Address,
					},
				}
				NeighBv6 := []*types.Neighbor{
					{
						PeerIP:           *l.A.IPv6Address,
						PeerAS:           *p.Config.Infrastructure.Protocols.AsPoolLoop[0],
						PeerGroup:        "LOOP",
						LocalAS:          *p.Config.Infrastructure.Protocols.AsPoolLoop[1],
						TransportAddress: *l.B.IPv6Address,
					},
				}

				bgplater = append(bgplater,
					&BGPLaterAdd{
						nodename: *l.A.Node.ShortName,
						niname:   infraNIName,
						nivid:    infraVID,
						bgpconf:  NeighAv4,
					},
					&BGPLaterAdd{
						nodename: *l.A.Node.ShortName,
						niname:   infraNIName,
						nivid:    infraVID,
						bgpconf:  NeighAv6,
					},
					&BGPLaterAdd{
						nodename: *l.B.Node.ShortName,
						niname:   wlname,
						nivid:    vlanid,
						bgpconf:  NeighBv4,
					},
					&BGPLaterAdd{
						nodename: *l.B.Node.ShortName,
						niname:   wlname,
						nivid:    vlanid,
						bgpconf:  NeighBv6,
					},
				)
			}
		}
	}
	return bgplater
}

type BGPLaterAdd struct {
	nodename string
	niname   string
	nivid    int
	bgpconf  []*types.Neighbor
}

func processStaticRoute(nhg *types.StaticRouteNHG) string {
	templateFile := path.Join("templates", "switch", "staticroute.tmpl")
	parameter := struct {
		Prefix      string
		Nhgroupname string
	}{Prefix: nhg.Prefix, Nhgroupname: nhg.NHGroup.Name}
	return GeneralTemplateProcessing(templateFile, "staticroute", parameter)
}

func processBfdInterface(interf *types.K8ssrlsubinterface) string {
	templateFile := path.Join("templates", "switch", "bfd.tmpl")
	return GeneralTemplateProcessing(templateFile, "bfd", interf.InterfaceRealName+"."+interf.VlanID)
}

func processBfdIrb(irbsubinterf *types.K8ssrlirbsubinterface) string {
	templateFile := path.Join("templates", "switch", "bfd.tmpl")
	return GeneralTemplateProcessing(templateFile, "bfd", irbsubinterf.InterfaceRealName+"."+irbsubinterf.VlanID)
}

func processEsi(esi *types.K8ssrlESI) string {
	templateFile := path.Join("templates", "switch", "esi.tmpl")
	return GeneralTemplateProcessing(templateFile, "esi", esi)
}

func processBgp(bgp *types.K8ssrlprotocolsbgp) string {
	templateFile := path.Join("templates", "switch", "bgp.tmpl")
	return GeneralTemplateProcessing(templateFile, "bgp", bgp)
}

func processNetworkInstance(networkinstance *types.K8ssrlNetworkInstance) string {
	templateFile := path.Join("templates", "switch", "networkinstance.tmpl")
	return GeneralTemplateProcessing(templateFile, "networkinstance", networkinstance)
}

func processIrbSubInterfaces(irbsubif *types.K8ssrlirbsubinterface) string {
	templateFile := path.Join("templates", "switch", "irbinterface.tmpl")
	return GeneralTemplateProcessing(templateFile, "irbinterface", irbsubif)
}

func processInterface(nodename string, islinterfaces *types.K8ssrlinterface) string {
	templateFile := path.Join("templates", "switch", "srlinterfaces.tmpl")
	return GeneralTemplateProcessing(templateFile, "srlinterface", islinterfaces)
}

func processVxlanInterfaces(tunifname string, vxinterf []*types.K8ssrlVxlanInterface) string {
	templateFile := path.Join("templates", "switch", "vxlanInterfaces.tmpl")
	data := struct {
		TunnelInterfaceName string
		VxlanInterfaces     []*types.K8ssrlVxlanInterface
	}{TunnelInterfaceName: tunifname, VxlanInterfaces: vxinterf}
	return GeneralTemplateProcessing(templateFile, "vxlaninterface", data)
}

func processRoutingPolicy(rp *types.K8ssrlRoutingPolicy) string {
	templateFile := path.Join("templates", "switch", "routingpolicy.tmpl")
	return GeneralTemplateProcessing(templateFile, "routingpolicy", rp)
}

func processSrlSubInterface(nodename string, interfacename string, srlsubif *types.K8ssrlsubinterface) string {
	templateFile := path.Join("templates", "switch", "subinterfaces.tmpl")
	data := struct {
		InterfaceName string
		SubInterface  *types.K8ssrlsubinterface
		Target        string
	}{srlsubif.InterfaceRealName, srlsubif, nodename}

	return GeneralTemplateProcessing(templateFile, "subinterface", data)
}
