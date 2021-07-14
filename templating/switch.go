package templating

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"path"
	"strings"

	"text/template"

	"github.com/nokia-paco-automation/paco-parser/parser"
	"github.com/nokia-paco-automation/paco-parser/types"

	log "github.com/sirupsen/logrus"
)

func ProcessSwitchTemplates(wr *types.WorkloadResults, ir *types.InfrastructureResult, cg *types.ClientGroupResults, n map[string]*parser.Node, appConfig map[string]*parser.AppConfig) map[string]string {
	log.Infof("ProcessingSwitchTemplates")

	templatenodes := map[string]*TemplateNode{}

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
	}

	// process subinterfaces on a per interface basis
	for nodename, syssubifs := range ir.SystemSubInterfaces {
		for _, syssubif := range syssubifs {
			conf := processSrlSubInterfaces(nodename, syssubif.InterfaceRealName, syssubif)
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
		for ifname, clientsubifs := range clientinterfaces {
			for _, clientsubif := range clientsubifs {
				conf := processSrlSubInterfaces(nodename, ifname, clientsubif)
				templatenodes[nodename].AddSubInterface(ifname, clientsubif.VlanID, conf)
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
				conf := processSrlSubInterfaces(nodename, ifname, srlsubif)
				templatenodes[nodename].AddSubInterface(srlsubif.InterfaceRealName, srlsubif.VlanID, conf)
				conf = processBfdInterface(srlsubif)
				templatenodes[nodename].AddBfd(conf)
			}
		}
	}

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

	// networkinstance
	for nodename, data := range wr.NetworkInstances {
		for _, networkinstance := range data {
			conf := processNetworkInstance(networkinstance)
			templatenodes[nodename].AddNetworkInstance(networkinstance.Name, conf)
		}
	}

	// bgp
	for nodename, bgp := range ir.DefaultProtocolBGP {
		conf := processBgp(bgp)
		templatenodes[nodename].AddBgp(bgp.NetworkInstanceName, conf)
	}

	// // static routes
	// for appname, appentries := range appLbIpResults.BgpIp {
	// 	for workloadname, workloadenrties := range appentries {
	// 		for index, ipInfo := range workloadenrties {
	// 			fmt.Printf("Loopback - app: %s, workload: %s, index: %d, IP: %s\n", appname, workloadname, index, ipInfo.ToString())

	// 		}
	// 	}
	// }

	// for appname, appentries := range appLbIpResults.LinkIp {
	// 	for workloadname, workloadenrties := range appentries {
	// 		for index, ipInfo := range workloadenrties {
	// 			fmt.Printf("LINK - app: %s, workload: %s, index: %d, IP: %s\n", appname, workloadname, index, ipInfo.ToString())
	// 			destination := ipInfo.Ipv4 + "/32"
	// 			resultingIrbSubIf := findRelatedIRBv4(wr.IrbSubInterfaces, destination)
	// 			niLookupResult := findNetworkInstanceOfIrb(wr.NetworkInstances, resultingIrbSubIf)
	// 			fmt.Printf("Node: %s, NI: %s\n", niLookupResult.nodename, niLookupResult.networkInstance.Name)
	// 			nhgroup := "TO BE FIGURED OUT!"
	// 			conf := processStaticRoute(destination, nhgroup)
	// 			templatenodes[niLookupResult.nodename].AddStaticRoute(niLookupResult.networkInstance.Name, conf)
	// 		}
	// 	}
	// }

	gsr := processAppConf(appConfig)

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

func processNextHopGroup(nhg *types.NHGroup) string {
	nhgj := NewJsonMerger()

	templateFile := path.Join("templates", "switch", "nhgroup.tmpl")
	nhgj.Merge([]byte(generalTemplateProcessing(templateFile, "nhgroup", nhg)))

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
	return generalTemplateProcessing(templateFile, "nhgroupentry", nhge)
}

func BGPPeerMapToString(peerinfo map[int]*parser.BGPPeerInfo) string {
	result := ""
	counter := 0
	for _, peer := range peerinfo {
		if counter > 0 {
			result += " | "
		}
		counter++
		result += fmt.Sprintf("Peer: %s, AS: %d", *peer.IP, *peer.AS)
	}

	return result
}

type GlobalStaticRoutes struct {
	Data map[string]map[string][]*types.StaticRouteNHG // nodename, networkinstance -> []*staticrouteNHG
}

func NewGlobalStaticRoutes() *GlobalStaticRoutes {
	return &GlobalStaticRoutes{
		Data: map[string]map[string][]*types.StaticRouteNHG{},
	}
}

func (gsr *GlobalStaticRoutes) addEntry(nodename string, networkinstance string, sr *types.StaticRouteNHG) {
	if _, ok := gsr.Data[nodename]; !ok {
		gsr.Data[nodename] = map[string][]*types.StaticRouteNHG{}
	}
	if _, ok := gsr.Data[nodename][networkinstance]; !ok {
		gsr.Data[nodename][networkinstance] = []*types.StaticRouteNHG{}
	}
	gsr.Data[nodename][networkinstance] = append(gsr.Data[nodename][networkinstance], sr)
}

func processAppConf(appconf map[string]*parser.AppConfig) *GlobalStaticRoutes {

	output := strings.Builder{}

	globalStaticRoutes := NewGlobalStaticRoutes()

	sr := types.NewStaticRouteNHG("66..6.6")
	sr.SetNHGroupName("FooBARGroup")
	sr.AddNHGroupEntry(&types.NHGroupEntry{
		Index:     5,
		NHIp:      "6.6.6.8",
		LocalAddr: "1.2.3.4",
	})
	sr.AddNHGroupEntry(&types.NHGroupEntry{
		Index:     3,
		NHIp:      "3.3.3.3",
		LocalAddr: "25.25.25.25",
	})
	globalStaticRoutes.addEntry("leaf1", "external-macvrf-ipvlan-1400", sr)

	for cnfName, cnf := range appconf {
		if cnfName != "upf" && cnfName != "smf" {
			continue
		}
		for wlName, workloads := range cnf.Networks {

			llbLoopbackInfoArr := workloads[0][0]["loopback"]["llbLbk"]

			for llbLoopb, foo := range llbLoopbackInfoArr {
				output.WriteString(fmt.Sprintf("llblbk - WLName: %s, BGPAddr: %s ,BGPPeers: [ %s ]\n", wlName, *foo.IPv4BGPAddress, BGPPeerMapToString(foo.IPv4BGPPeers)))

				_ = llbLoopb
				// sr := types.NewStaticRouteNHG(*foo.IPv4BGPAddress)
				// sr.SetNHGroupName(fmt.Sprintf("llb%d-%s-%s-%s", llbLoopb, "InterfaceName", "Target", *foo.NetworkShortName))

			}

			for x := 1; x < len(workloads[0]); x++ {
				llbInterfInfoArr := workloads[0][x]["itfce"]["intIP"]

				for llbInterfInfoIndex, interfInfo := range llbInterfInfoArr {
					output.WriteString(fmt.Sprintf("intIP - WLName: %s, VLANID: %d, Target: %s, BGPPeers: [ %s ]\n", wlName, *interfInfo.VlanID, *interfInfo.Target, BGPPeerMapToString(interfInfo.IPv4BGPPeers)))

					_ = llbInterfInfoIndex
					// sr := types.NewStaticRouteNHG("66..6.6")
					// sr.SetNHGroupName(fmt.Sprintf("llb%d-%s", llbInterfInfoIndex, *interfInfo.NetworkShortName))
					// for _, IPv4Peer := range interfInfo.IPv4BGPPeers {
					// 	nhgentry := &types.NHGroupEntry{
					// 		Index:     *interfInfo.VlanID,
					// 		NHIp:      *interfInfo.Ipv4Addresses[0].IPAddress,
					// 		LocalAddr: *IPv4Peer.IP,
					// 	}
					// 	sr.AddNHGroupEntry(nhgentry)
					// }

					// globalStaticRoutes.addEntry(*interfInfo.Target, "networkInstance", sr)
				}
			}
			// for switchIndex, switchWorkloads := range workloads {
			// 	for group, groups := range switchWorkloads {
			// 		for nwtype, nwtypes := range groups {
			// 			for nwsubType, netwsubTypes := range nwtypes {
			// 				for idx, networkInfo := range netwsubTypes {
			// 					prefix := fmt.Sprintf("CNFName: %s, WLName: %s, SwIndex: %d, Group: %d, NWType: %s, NWSubType: %s, Index %d, Length: %d", cnfName, wlName, switchIndex, group, nwtype, nwsubType, idx, len(networkInfo.Ipv4Addresses))
			// 					for _, allocatedIPinfo := range networkInfo.Ipv4Addresses {
			// 						output.WriteString(prefix + fmt.Sprintf(" IP: %s\n", *allocatedIPinfo.IPAddress))
			// 					}
			// 				}
			// 			}
			// 		}
			// 	}
			// }
		}
	}

	fmt.Println(output.String())
	return globalStaticRoutes
}

func processStaticRoute(nhg *types.StaticRouteNHG) string {
	templateFile := path.Join("templates", "switch", "staticroute.tmpl")
	parameter := struct {
		Prefix      string
		Nhgroupname string
	}{Prefix: nhg.Prefix, Nhgroupname: nhg.NHGroup.Name}
	return generalTemplateProcessing(templateFile, "staticroute", parameter)
}

type NetworkInstanceLookupResult struct {
	nodename        string
	networkInstance *types.K8ssrlNetworkInstance
}

func findNetworkInstanceOfIrb(networkinstances map[string]map[int]*types.K8ssrlNetworkInstance, irbif *types.K8ssrlirbsubinterface) *NetworkInstanceLookupResult {
	for nodename, networkinstances := range networkinstances {
		for _, ni := range networkinstances {
			for _, subif := range ni.SubInterfaces {
				if subif.InterfaceRealName == irbif.InterfaceRealName && subif.VlanID == irbif.VlanID {
					return &NetworkInstanceLookupResult{
						nodename:        nodename,
						networkInstance: ni,
					}
				}
			}
		}
	}
	log.Fatalln("No Networkinstance found!")
	return nil
}

func findRelatedIRBv4(irbsubinterface map[string][]*types.K8ssrlirbsubinterface, ipv4 string) *types.K8ssrlirbsubinterface {
	appIp, _, err := net.ParseCIDR(ipv4)
	if err != nil {
		log.Fatalln("Not a valid IP.")
	}
	for _, irbsubifs := range irbsubinterface {
		for _, irbsubif := range irbsubifs {
			//fmt.Printf("Node: %s, ifname: %s, IPv4: %s, IPv6: %s\n", nodename, irbsubif.InterfaceRealName, irbsubif.IPv4Prefix, irbsubif.IPv6Prefix)
			for _, entry := range irbsubif.IPv4Prefix {
				_, irbnet, err := net.ParseCIDR(entry)
				if err != nil {
					log.Fatalln("IP Parsing error")
				}
				if irbnet.Contains(appIp) {
					fmt.Printf("MATCH: Ipv4: %s, Net %s\n", ipv4, irbnet.String())
					return irbsubif
				}
			}
		}
	}
	log.Fatalln("not found!")
	return nil
}

func processBfdInterface(interf *types.K8ssrlsubinterface) string {
	templateFile := path.Join("templates", "switch", "bfd.tmpl")
	return generalTemplateProcessing(templateFile, "bfd", interf.InterfaceRealName+"."+interf.VlanID)
}

func processBfdIrb(irbsubinterf *types.K8ssrlirbsubinterface) string {
	templateFile := path.Join("templates", "switch", "bfd.tmpl")
	return generalTemplateProcessing(templateFile, "bfd", irbsubinterf.InterfaceRealName+"."+irbsubinterf.VlanID)
}

func processEsi(esi *types.K8ssrlESI) string {
	templateFile := path.Join("templates", "switch", "esi.tmpl")
	return generalTemplateProcessing(templateFile, "esi", esi)
}

func processBgp(bgp *types.K8ssrlprotocolsbgp) string {
	templateFile := path.Join("templates", "switch", "bgp.tmpl")
	return generalTemplateProcessing(templateFile, "bgp", bgp)
}

func processNetworkInstance(networkinstance *types.K8ssrlNetworkInstance) string {
	templateFile := path.Join("templates", "switch", "networkinstance.tmpl")
	return generalTemplateProcessing(templateFile, "networkinstance", networkinstance)
}

func processIrbSubInterfaces(irbsubif *types.K8ssrlirbsubinterface) string {
	templateFile := path.Join("templates", "switch", "irbinterface.tmpl")
	return generalTemplateProcessing(templateFile, "irbinterface", irbsubif)
}

func processInterface(nodename string, islinterfaces *types.K8ssrlinterface) string {
	templateFile := path.Join("templates", "switch", "srlinterfaces.tmpl")
	return generalTemplateProcessing(templateFile, "srlinterface", islinterfaces)
}

func processVxlanInterfaces(tunifname string, vxinterf []*types.K8ssrlVxlanInterface) string {
	templateFile := path.Join("templates", "switch", "vxlanInterfaces.tmpl")
	data := struct {
		TunnelInterfaceName string
		VxlanInterfaces     []*types.K8ssrlVxlanInterface
	}{TunnelInterfaceName: tunifname, VxlanInterfaces: vxinterf}
	return generalTemplateProcessing(templateFile, "vxlaninterface", data)
}

func processRoutingPolicy(rp *types.K8ssrlRoutingPolicy) string {
	templateFile := path.Join("templates", "switch", "routingpolicy.tmpl")
	return generalTemplateProcessing(templateFile, "routingpolicy", rp)
}

func processSrlSubInterfaces(nodename string, interfacename string, srlsubif *types.K8ssrlsubinterface) string {
	templateFile := path.Join("templates", "switch", "subinterfaces.tmpl")
	data := struct {
		InterfaceName string
		SubInterface  *types.K8ssrlsubinterface
		Target        string
	}{srlsubif.InterfaceRealName, srlsubif, nodename}

	return generalTemplateProcessing(templateFile, "subinterface", data)
}

func generalTemplateProcessing(templateFile string, templateName string, data interface{}) string {
	t := template.Must(template.ParseFiles(templateFile))
	buf := new(bytes.Buffer)
	err := t.ExecuteTemplate(buf, templateName, data)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	return buf.String()
}
