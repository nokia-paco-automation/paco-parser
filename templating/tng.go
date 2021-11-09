package templating

import (
	"bytes"
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
	Systemip string
	Leaf1    *TngCnfPodLeafIPInfo
	Leaf2    *TngCnfPodLeafIPInfo
}

type TngCnfPodLeafIPInfo struct {
	Ip string
	Gw string
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
	BgpAs         uint32
	Id            string
	IrbName       string
	LoName        string
	Name          string
	UplinkItfname string
	VxlName       string
}

func ProcessTNG(p *parser.Parser, wr *types.WorkloadResults, ir *types.InfrastructureResult, cg *types.ClientGroupResults, appconfig map[string]*parser.AppConfig, looptngresult *SwitchGoTNGResult) string {
	tng := &TngRoot{
		Leafgrp: &TngLeafGroup{},
		Cnfs:    []*TngCnf{},
	}

	processTNGCnfs(appconfig, ir, wr, tng, p)
	processTNGLeafGroups(p, tng, wr, ir, cg, appconfig, looptngresult)
	processLeafGroupLeafs(tng, p)
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
			tngsubnet := &TngSubnet{
				Gatewaysv4: []string{},
				Name:       name,
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
		tng.Leafgrp.SpineAs = 4259845498 // TODO
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

func processLeafGroupLeafs(tng *TngRoot, p *parser.Parser) {
	for name, leaf := range get_leafs(p) {
		new_leafgroupleaf := &TngLeafGroupLeaf{
			BgpAs:         *leaf.AS,
			Id:            *leaf.MgmtIPv4,
			IrbName:       "irb0",
			LoName:        "lo0",
			Name:          name,
			UplinkItfname: getUplinkName(name, p),
			VxlName:       "vxlan0",
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
		Systemip: "",
		Leaf1:    &TngCnfPodLeafIPInfo{},
		Leaf2:    &TngCnfPodLeafIPInfo{},
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

				if routeentry.RType == "llbbgp" {
					tng_cnf_workloads[routeentry.CnfName][wlname].Name = wlname
					if routeentry.IpVersion == "v4" {
						tng_cnf_workloads[routeentry.CnfName][wlname].Llbvipv4 = routeentry.Prefix
					}
					if routeentry.IpVersion == "v6" {
						tng_cnf_workloads[routeentry.CnfName][wlname].Llbvipv6 = routeentry.Prefix
					}

				} else if routeentry.RType == "llb" {
					if tng_cnf_workloads[routeentry.CnfName][wlname].LlbPods == nil {
						tng_cnf_workloads[routeentry.CnfName][wlname].LlbPods = []*TngCnfPodRoutes{}
					}
					if _, exists := podRouteStore[routeentry.Prefix]; !exists {
						newPodRoute := &TngCnfPodRoutes{
							Systemip: routeentry.Prefix,
							Leaf1:    &TngCnfPodLeafIPInfo{},
							Leaf2:    &TngCnfPodLeafIPInfo{},
						}
						podRouteStore[routeentry.Prefix] = newPodRoute
						tng_cnf_workloads[routeentry.CnfName][wlname].LlbPods = append(tng_cnf_workloads[routeentry.CnfName][wlname].LlbPods, newPodRoute)
					}

					if _, exists := routeentry.NHGroup.Entries[1]; exists {
						podRouteStore[routeentry.Prefix].Leaf1.Ip = routeentry.NHGroup.Entries[1].LocalAddr
						podRouteStore[routeentry.Prefix].Leaf1.Gw = routeentry.NHGroup.Entries[1].NHIp
					}

					if _, exists := routeentry.NHGroup.Entries[2]; exists {
						podRouteStore[routeentry.Prefix].Leaf2.Ip = routeentry.NHGroup.Entries[2].LocalAddr
						podRouteStore[routeentry.Prefix].Leaf2.Gw = routeentry.NHGroup.Entries[2].NHIp
					}

				} else if routeentry.RType == "lmg" {
					if tng_cnf_workloads[routeentry.CnfName][wlname].LmgPods == nil {
						tng_cnf_workloads[routeentry.CnfName][wlname].LmgPods = []*TngCnfPodRoutes{}
					}
					if _, exists := podRouteStore[routeentry.Prefix]; !exists {
						newPodRoute := &TngCnfPodRoutes{
							Systemip: routeentry.Prefix,
							Leaf1:    &TngCnfPodLeafIPInfo{},
							Leaf2:    &TngCnfPodLeafIPInfo{},
						}
						podRouteStore[routeentry.Prefix] = newPodRoute
						tng_cnf_workloads[routeentry.CnfName][wlname].LmgPods = append(tng_cnf_workloads[routeentry.CnfName][wlname].LmgPods, newPodRoute)
					}

					if _, exists := routeentry.NHGroup.Entries[1]; exists {
						podRouteStore[routeentry.Prefix].Leaf1.Ip = routeentry.NHGroup.Entries[1].LocalAddr
						podRouteStore[routeentry.Prefix].Leaf1.Gw = routeentry.NHGroup.Entries[1].NHIp
					}

					if _, exists := routeentry.NHGroup.Entries[2]; exists {
						podRouteStore[routeentry.Prefix].Leaf2.Ip = routeentry.NHGroup.Entries[2].LocalAddr
						podRouteStore[routeentry.Prefix].Leaf2.Gw = routeentry.NHGroup.Entries[2].NHIp
					}

				}
			}
		}
	}
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
