package templating

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"
	"text/template"

	"github.com/nokia-paco-automation/paco-parser/parser"
	"github.com/nokia-paco-automation/paco-parser/types"
	log "github.com/sirupsen/logrus"
)

type TngRoot struct {
	Leafgrp *TngLeafGroup
	Cnfs    []*TngCnf
}

type TngCnf struct {
	Name      string
	AsNumber  uint32
	Workloads []*TngCnfWorkload
}
type TngCnfWorkload struct {
	LlbPods  []*TngCnfPodRoutes
	LmgPods  []*TngCnfPodRoutes
	Llbvipv4 string
	Llbvipv6 string
	Name     string
}

type TngCnfPodRoutes struct {
	Systemipv4 string
	Systemipv6 string
	Leaf1      *TngCnfPodLeafIPInfo
	Leaf2      *TngCnfPodLeafIPInfo
}

type TngCnfPodLeafIPInfo struct {
	Ipv4 string
	Gwv4 string
	Ipv6 string
	Gwv6 string
}

type TngLeafGroup struct {
	Name    string
	SpineAs uint32
	IpVrfs  []*TngIpVrf
	Leafs   []*TngLeafGroupLeaf
	Loop    *TngLeafGroupLoop
}

type TngLeafGroupLoop struct {
	Paco_itf        string
	Infra_itf       string
	K8s_infra_as    uint32
	Paco_overlay_as uint32
	Infra_ipvrf     string
}

type TngIpVrf struct {
	Leaf1routerid string
	Leaf2routerid string
	Name          string
	SpineUplink   *TngSpineUplink
	InfraBgp      *TngIpVrfInfraBgp
	StandardName  string
	Subnets       []*TngSubnet
	VxlanVni      int
}

type TngIpVrfInfraBgp struct {
	Leaf1_local_address string
	Leaf1_peer_address  string
	Leaf2_local_address string
	Leaf2_peer_address  string
}

type TngSubnet struct {
	Gatewaysv4 []string
	Name       string
	Type       string
	Vlan       int
	Target     string
}

type TngSpineUplink struct {
	Vlan                int
	Leaf1_local_address string
	Leaf2_local_address string
	Leaf1_uplink_peers  []string
	Leaf2_uplink_peers  []string
}

type TngLeafGroupLeaf struct {
	BgpAs               uint32
	Id                  string
	IrbName             string
	LoName              string
	Name                string
	UplinkItfname       string
	VxlName             string
	IpVlanInterfaceList []string
	SriovInterfaceList  []string
}

func ProcessTNG(p *parser.Parser, wr *types.WorkloadResults, ir *types.InfrastructureResult, cg *types.ClientGroupResults, appconfig map[string]*parser.AppConfig, looptngresult *SwitchGoTNGResult) string {
	tng := &TngRoot{
		Leafgrp: &TngLeafGroup{},
		Cnfs:    []*TngCnf{},
	}

	processTNGCnfs(appconfig, ir, wr, tng, p)
	processTNGLeafGroups(p, tng, wr, ir, cg, appconfig, looptngresult)
	processLeafGroupLeafs(tng, p, cg)
	return populateTemplate(tng)
}

func processTNGLeafGroups(p *parser.Parser, tng *TngRoot, wr *types.WorkloadResults, ir *types.InfrastructureResult, cg *types.ClientGroupResults, appconfig map[string]*parser.AppConfig, switchgotngresult *SwitchGoTNGResult) {
	for wlname, wl := range p.Config.Workloads {
		var vni int
		if _, ok := wl["servers"]; !ok {
			continue
		}
		if wl["servers"].Loopbacks == nil {
			if wlname == "multus-mgmt" {
				vni = *wl["dcgw-grp1"].Itfces["itfce"].VlanID
			} else {
				continue
			}
		} else {
			vni = *wl["servers"].Loopbacks["loopback"].Idx
		}

		wlname = strings.TrimPrefix(wlname, "multus-")
		niName := wlname + "-ipvrf-itfce-" + strconv.Itoa(vni)

		loopresult_leaf1_v4, err1 := switchgotngresult.GetLoopEntryFor(vni, "leaf1", "v4")
		loopresult_leaf2_v4, err2 := switchgotngresult.GetLoopEntryFor(vni, "leaf2", "v4")
		var infrabgp *TngIpVrfInfraBgp

		if err1 == nil || err2 == nil {
			infrabgp = &TngIpVrfInfraBgp{
				Leaf1_local_address: loopresult_leaf1_v4.local_address,
				Leaf1_peer_address:  loopresult_leaf1_v4.peer_address,
				Leaf2_local_address: loopresult_leaf2_v4.local_address,
				Leaf2_peer_address:  loopresult_leaf2_v4.peer_address,
			}
		} else {
			infrabgp = nil
		}

		// figure out ROuterIDs
		var leafrids = []string{"", ""}
		for no, node := range []string{"leaf1", "leaf2"} {
			for _, sif := range wr.NetworkInstances[node][vni].SubInterfaces {
				if sif.Kind == "loopback" {
					leafrids[no] = sif.IPv4Prefix
					break
				}
			}
			if leafrids[no] == "" {
				for _, sif := range ir.DefaultNetworkInstances[node].SubInterfaces {
					if sif.Kind == "loopback" {
						leafrids[no] = sif.IPv4Prefix
						break
					}
				}
			}
		}

		tngVrf := &TngIpVrf{
			Leaf1routerid: strings.Split(leafrids[0], "/")[0],
			Leaf2routerid: strings.Split(leafrids[1], "/")[0],
			Name:          niName,
			SpineUplink:   &TngSpineUplink{},
			InfraBgp:      infrabgp,
			StandardName:  niName,
			Subnets:       []*TngSubnet{},
			VxlanVni:      vni,
		}

		// for _,x := range wr.NetworkInstances
		// 	tngsubnet := &TngSubnet{
		// 		//Gatewaysv4: ,
		// 		Name:   name,
		// 		Vlan:   *entry.VlanID,
		// 	}
		// }
		for name, entry := range wl["servers"].Itfces {
			typepart := "macvrf-" + name
			snName := wlname + "-" + typepart + "-" + strconv.Itoa(*entry.VlanID)
			tngsubnet := &TngSubnet{
				Gatewaysv4: []string{},
				Name:       snName,
				Type:       "",
				Vlan:       *entry.VlanID,
				Target:     "",
			}
			for _, node := range []string{"leaf1", "leaf2"} {
				for _, irbif := range wr.IrbSubInterfaces[node] {
					if irbif.VlanID == strconv.Itoa(*entry.VlanID) {
						tngsubnet.Gatewaysv4 = irbif.IPv4Prefix
					}
				}
			}

			if entry.Target == nil {
				tngsubnet.Type = "ipvlan"
			} else {
				tngsubnet.Type = "sriov"
				tngsubnet.Target = *entry.Target
			}
			tngVrf.Subnets = append(tngVrf.Subnets, tngsubnet)
		}

		// tngS := &TngSubnet{
		// 	Gatewaysv4: []string{},
		// 	Name:       niName,
		// 	Type:       "",
		// 	Vlan:       0,
		// 	Target:     "",
		// }

		l1suplinkv4 := switchgotngresult.GetSpineUplinkFor(vni, "leaf1", "v4")
		if l1suplinkv4 != nil {
			tngVrf.SpineUplink.Leaf1_local_address = l1suplinkv4.local_ip
			tngVrf.SpineUplink.Leaf1_uplink_peers = l1suplinkv4.peer_ips
		}
		l2suplinkv4 := switchgotngresult.GetSpineUplinkFor(vni, "leaf2", "v4")
		if l2suplinkv4 != nil {
			tngVrf.SpineUplink.Leaf2_local_address = l2suplinkv4.local_ip
			tngVrf.SpineUplink.Leaf2_uplink_peers = l2suplinkv4.peer_ips
		}

		tngVrf.SpineUplink.Vlan = *wl["dcgw-grp1"].Itfces["itfce"].VlanID

		tng.Leafgrp.IpVrfs = append(tng.Leafgrp.IpVrfs, tngVrf)
		tng.Leafgrp.SpineAs = *p.Config.Workloads["infrastructure"]["dcgw-grp1"].Itfces["itfce"].PeerAS
		tng.Leafgrp.Name = "leaf-grp1"
		tng.Leafgrp.Loop = &TngLeafGroupLoop{}

		infraNIName := ""
		infraVID := 0
		for wlname, workload := range p.Config.Workloads {
			if strings.Contains(strings.ToLower(wlname), "infrastru") {
				infraNIName = wlname
				infraVID = *workload["dcgw-grp1"].Itfces["itfce"].VlanID
				break
			}
		}

		// Process loop infos
		for _, l := range p.Links {
			if *l.Kind == "loop" {
				tng.Leafgrp.Loop.Infra_ipvrf = infraNIName + "-ipvrf-itfce-" + strconv.Itoa(infraVID)
				tng.Leafgrp.Loop.Infra_itf = *l.A.RealName
				tng.Leafgrp.Loop.K8s_infra_as = *p.Config.Infrastructure.Protocols.AsPoolLoop[0]
				tng.Leafgrp.Loop.Paco_itf = *l.B.RealName
				tng.Leafgrp.Loop.Paco_overlay_as = *p.Config.Infrastructure.Protocols.AsPoolLoop[1]
				break
			}
		}
	}
}

func processLeafGroupLeafs(tng *TngRoot, p *parser.Parser, cg *types.ClientGroupResults) {
	for name, leaf := range get_leafs(p) {
		new_leafgroupleaf := &TngLeafGroupLeaf{
			BgpAs:               *p.Config.Infrastructure.Protocols.OverlayAs,
			Id:                  *leaf.MgmtIPv4,
			IrbName:             "irb0",
			LoName:              "lo0",
			Name:                name,
			UplinkItfname:       getUplinkName(name, p),
			VxlName:             "vxlan0",
			SriovInterfaceList:  []string{},
			IpVlanInterfaceList: []string{},
		}

		stringSetIpVlan := map[string]bool{}
		stringSetSriov := map[string]bool{}
		for _, entry := range cg.ClientInterfaces[name]["servers"] {
			if entry.Lag {
				stringSetIpVlan[entry.Name] = true
			} else {
				stringSetSriov[entry.Name] = true
			}
		}
		for k, _ := range stringSetIpVlan {
			new_leafgroupleaf.IpVlanInterfaceList = append(new_leafgroupleaf.IpVlanInterfaceList, k)
		}
		for k, _ := range stringSetSriov {
			new_leafgroupleaf.SriovInterfaceList = append(new_leafgroupleaf.SriovInterfaceList, k)
		}

		tng.Leafgrp.Leafs = append(tng.Leafgrp.Leafs, new_leafgroupleaf)
	}
}

func getUplinkName(leafname string, p *parser.Parser) string {
	result := ""
	for _, x := range p.Config.Topology.Links {
		for _, y := range x.Endpoints {
			data := strings.Split(*y, ":")
			if data[0] == leafname && x.Labels != nil && *x.Labels["kind"] == "dcgw" {
				result = strings.ReplaceAll(data[1], "-", "/")
				result = strings.ReplaceAll(result, "e", "ethernet-")
				return result
			}
		}
	}
	return ""
}

func get_leafs(p *parser.Parser) map[string]*parser.NodeConfig {
	result := map[string]*parser.NodeConfig{}
	for name, ndata := range p.Config.Topology.Nodes {
		if *ndata.Kind == "srl" {
			result[name] = ndata
		}
	}
	return result
}

// func match_label(actual map[string]string, match_labels map[string]string) bool {
// 	for k, v := range match_labels {
// 		_, exists := actual[k]
// 		if !exists || v != actual[k] {
// 			return false
// 		}
// 	}
// 	return true
// }

func processTNGCnfs(appconf map[string]*parser.AppConfig, ir *types.InfrastructureResult, wr *types.WorkloadResults, tng *TngRoot, p *parser.Parser) {

	resultmap := map[string]*TngCnf{}

	tng_cnf_workloads := map[string]map[string]*TngCnfWorkload{} // CNF, Workload

	cnfpodroutes := TngCnfPodRoutes{
		Systemipv4: "",
		Leaf1:      &TngCnfPodLeafIPInfo{},
		Leaf2:      &TngCnfPodLeafIPInfo{},
	}
	_ = tng_cnf_workloads
	_ = cnfpodroutes

	GlobalStaticRoutes := processAppConfSrNhg(appconf, ir, wr)
	podRouteStore := map[string]*TngCnfPodRoutes{}
	for device, switchdata := range GlobalStaticRoutes.Data {
		for wlname, routearr := range switchdata {
			for _, routeentry := range routearr {
				_ = device
				_ = wlname

				if routeentry.IpVersion == "v6" {
					continue
				}

				if _, exists := tng_cnf_workloads[routeentry.CnfName]; !exists {
					tng_cnf_workloads[routeentry.CnfName] = map[string]*TngCnfWorkload{}
				}
				if _, exists := tng_cnf_workloads[routeentry.CnfName][wlname]; !exists {
					newWorkload := &TngCnfWorkload{}
					tng_cnf_workloads[routeentry.CnfName][wlname] = newWorkload
					if _, exists := resultmap[routeentry.CnfName]; !exists {
						asnumber := p.Config.Application["paco"].Cnfs[routeentry.CnfName].Networking.AS
						newCnf := &TngCnf{Name: routeentry.CnfName, AsNumber: *asnumber, Workloads: []*TngCnfWorkload{}}
						resultmap[routeentry.CnfName] = newCnf
						tng.Cnfs = append(tng.Cnfs, newCnf)
					}
					resultmap[routeentry.CnfName].Workloads = append(resultmap[routeentry.CnfName].Workloads, newWorkload)
				}

				fmt.Println(device + " " + wlname + " " + routeentry.RType + " " + strconv.Itoa(routeentry.INPUT_INDEX) + " " + routeentry.CnfName + " " + routeentry.WlName + " " + routeentry.IpVersion + " " + strconv.Itoa(routeentry.VlanID) + " " + routeentry.Prefix + " ")

				if routeentry.RType == "llbbgp" {
					v6_info := findv6Info(routearr, routeentry)

					tng_cnf_workloads[routeentry.CnfName][wlname].Name = wlname
					if routeentry.IpVersion == "v4" {
						tng_cnf_workloads[routeentry.CnfName][wlname].Llbvipv4 = strings.Split(routeentry.Prefix, "/")[0]
					}

					tng_cnf_workloads[routeentry.CnfName][wlname].Llbvipv6 = strings.Split(v6_info.Prefix, "/")[0]

				} else if routeentry.RType == "llb" {

					v6_info := findv6Info(routearr, routeentry)

					if tng_cnf_workloads[routeentry.CnfName][wlname].LlbPods == nil {
						tng_cnf_workloads[routeentry.CnfName][wlname].LlbPods = []*TngCnfPodRoutes{}
					}
					if _, exists := podRouteStore[routeentry.Prefix]; !exists {
						newPodRoute := &TngCnfPodRoutes{
							Systemipv4: strings.Split(routeentry.Prefix, "/")[0],
							Systemipv6: strings.Split(v6_info.Prefix, "/")[0],
							Leaf1:      &TngCnfPodLeafIPInfo{},
							Leaf2:      &TngCnfPodLeafIPInfo{},
						}
						podRouteStore[routeentry.Prefix] = newPodRoute
						tng_cnf_workloads[routeentry.CnfName][wlname].LlbPods = append(tng_cnf_workloads[routeentry.CnfName][wlname].LlbPods, newPodRoute)
					}

					if _, exists := routeentry.NHGroup.Entries[1]; exists {
						podRouteStore[routeentry.Prefix].Leaf1.Ipv4 = routeentry.NHGroup.Entries[1].NHIp
						podRouteStore[routeentry.Prefix].Leaf1.Gwv4 = routeentry.NHGroup.Entries[1].LocalAddr
					}

					if _, exists := v6_info.NHGroup.Entries[1]; exists {
						podRouteStore[routeentry.Prefix].Leaf1.Ipv6 = v6_info.NHGroup.Entries[1].NHIp
						podRouteStore[routeentry.Prefix].Leaf1.Gwv6 = v6_info.NHGroup.Entries[1].LocalAddr
					}

					if _, exists := routeentry.NHGroup.Entries[2]; exists {
						podRouteStore[routeentry.Prefix].Leaf2.Ipv4 = routeentry.NHGroup.Entries[2].NHIp
						podRouteStore[routeentry.Prefix].Leaf2.Gwv4 = routeentry.NHGroup.Entries[2].LocalAddr
					}

					if _, exists := v6_info.NHGroup.Entries[2]; exists {
						podRouteStore[routeentry.Prefix].Leaf2.Ipv6 = v6_info.NHGroup.Entries[2].NHIp
						podRouteStore[routeentry.Prefix].Leaf2.Gwv6 = v6_info.NHGroup.Entries[2].LocalAddr
					}

				} else if routeentry.RType == "lmg" {

					v6_info := findv6Info(routearr, routeentry)

					if tng_cnf_workloads[routeentry.CnfName][wlname].LmgPods == nil {
						tng_cnf_workloads[routeentry.CnfName][wlname].LmgPods = []*TngCnfPodRoutes{}
					}
					if _, exists := podRouteStore[routeentry.Prefix]; !exists {
						newPodRoute := &TngCnfPodRoutes{
							Systemipv4: strings.Split(routeentry.Prefix, "/")[0],
							Systemipv6: strings.Split(v6_info.Prefix, "/")[0],
							Leaf1:      &TngCnfPodLeafIPInfo{},
							Leaf2:      &TngCnfPodLeafIPInfo{},
						}
						podRouteStore[routeentry.Prefix] = newPodRoute
						tng_cnf_workloads[routeentry.CnfName][wlname].LmgPods = append(tng_cnf_workloads[routeentry.CnfName][wlname].LmgPods, newPodRoute)
					}

					if _, exists := routeentry.NHGroup.Entries[1]; exists {
						podRouteStore[routeentry.Prefix].Leaf1.Ipv4 = routeentry.NHGroup.Entries[1].NHIp
						podRouteStore[routeentry.Prefix].Leaf1.Gwv4 = routeentry.NHGroup.Entries[1].LocalAddr
					}
					if _, exists := v6_info.NHGroup.Entries[1]; exists {
						podRouteStore[routeentry.Prefix].Leaf1.Ipv6 = v6_info.NHGroup.Entries[1].NHIp
						podRouteStore[routeentry.Prefix].Leaf1.Gwv6 = v6_info.NHGroup.Entries[1].LocalAddr
					}

					if _, exists := routeentry.NHGroup.Entries[2]; exists {
						podRouteStore[routeentry.Prefix].Leaf2.Ipv4 = routeentry.NHGroup.Entries[2].NHIp
						podRouteStore[routeentry.Prefix].Leaf2.Gwv4 = routeentry.NHGroup.Entries[2].LocalAddr
					}
					if _, exists := v6_info.NHGroup.Entries[2]; exists {
						podRouteStore[routeentry.Prefix].Leaf2.Ipv6 = v6_info.NHGroup.Entries[2].NHIp
						podRouteStore[routeentry.Prefix].Leaf2.Gwv6 = v6_info.NHGroup.Entries[2].LocalAddr
					}
				}
			}
		}
	}
}

func findv6Info(routearr []*types.StaticRouteNHG, r *types.StaticRouteNHG) *types.StaticRouteNHG {
	result := []*types.StaticRouteNHG{}
	for _, x := range routearr {
		if r.CnfName == x.CnfName && r.RType == x.RType && r.TargetLeaf == x.TargetLeaf && r.VlanID == x.VlanID && r.INPUT_INDEX == x.INPUT_INDEX && x.IpVersion == "v6" {
			result = append(result, x)
		}
	}
	if len(result) > 1 {
		log.Fatal("non unique result!")
	}
	return result[0]
}

func populateTemplate(tng *TngRoot) string {
	t := template.Must(template.ParseGlob(path.Join("templates", "tng", "*.tmpl")))
	template.ParseFiles()
	buf := new(bytes.Buffer)
	err := t.ExecuteTemplate(buf, "TNGRoot", tng)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	return buf.String()
}
