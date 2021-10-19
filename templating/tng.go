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
	Leafs   []*TngLeaf
}

type TngIpVrf struct {
	LoopbackCIDR string
	Name         string
	SpineUplink  *TngSpineUplink
	StandardName string
	Subnets      []*TngSubnet
	VxlanVni     int
}

type TngSubnet struct {
	Cidrv4 []string
	Cidrv6 []string
	Name   string
	Type   string
	Vlan   int
	Target string
}

type TngSpineUplink struct {
	Subnetv4 string
	Subnetv6 string
	Vlan     int
}

type TngLeaf struct {
	BgpAs         uint32
	Id            string
	IrbName       string
	LoName        string
	Name          string
	UplinkLagName string
	VxlName       string
}

func ProcessTNG(p *parser.Parser, wr *types.WorkloadResults, ir *types.InfrastructureResult, cg *types.ClientGroupResults, appconfig map[string]*parser.AppConfig) string {
	tng := &TngRoot{
		Leafgrp: &TngLeafGroup{},
		Cnfs:    []*TngCnf{},
	}

	processTNGCnfs(appconfig, ir, wr, tng, p)
	processTNGLeafGroups(p, tng, wr, ir, cg, appconfig)
	return populateTemplate(tng)
}

func processTNGLeafGroups(p *parser.Parser, tng *TngRoot, wr *types.WorkloadResults, ir *types.InfrastructureResult, cg *types.ClientGroupResults, appconfig map[string]*parser.AppConfig) {
	for wlname, wl := range p.Config.Workloads {

		if _, ok := wl["servers"]; !ok {
			continue
		}
		if wl["servers"].Loopbacks == nil {
			continue
		}

		vni := *wl["servers"].Loopbacks["loopback"].Idx

		wlname = strings.TrimPrefix(wlname, "multus-")
		niName := wlname + "-ipvrf-itfce-" + strconv.Itoa(vni)

		tngVrf := &TngIpVrf{
			LoopbackCIDR: *wl["servers"].Loopbacks["loopback"].Ipv4Cidr[0],
			Name:         niName,
			SpineUplink:  &TngSpineUplink{},
			StandardName: niName,
			Subnets:      []*TngSubnet{},
			VxlanVni:     vni,
		}

		for name, entry := range wl["servers"].Itfces {
			tngcidrv4 := []string{}
			for _, cidr := range entry.Ipv4Cidr {
				tngcidrv4 = append(tngcidrv4, *cidr)
			}
			tngcidrv6 := []string{}
			for _, cidr := range entry.Ipv6Cidr {
				tngcidrv6 = append(tngcidrv6, *cidr)
			}

			tngsubnet := &TngSubnet{
				Cidrv4: tngcidrv4,
				Cidrv6: tngcidrv6,
				Name:   name,
				Vlan:   *entry.VlanID,
			}

			if entry.Target == nil {
				tngsubnet.Type = "ipvrf"
			} else {
				tngsubnet.Type = "sriov"
				tngsubnet.Target = *entry.Target
			}
			tngVrf.Subnets = append(tngVrf.Subnets, tngsubnet)
		}

		tngVrf.SpineUplink.Subnetv4 = *wl["dcgw-grp1"].Itfces["itfce"].Ipv4Cidr[0]
		tngVrf.SpineUplink.Subnetv6 = *wl["dcgw-grp1"].Itfces["itfce"].Ipv6Cidr[0]
		tngVrf.SpineUplink.Vlan = *wl["dcgw-grp1"].Itfces["itfce"].VlanID

		tng.Leafgrp.IpVrfs = append(tng.Leafgrp.IpVrfs, tngVrf)
		tng.Leafgrp.SpineAs = 0 // TODO
		tng.Leafgrp.Name = "leaf-grp1"
	}
}

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
