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
	Name     string
	AsNumber uint32
	LlbPods  int
	LmgPods  int
	Multus   []*struct {
		Name string
	}
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
	Cidrv4 string
	Cidrv6 string
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

	processTNGCnfs(p, tng)
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
			StandardName: "", // TODO
			Subnets:      []*TngSubnet{},
			VxlanVni:     vni,
		}

		for name, entry := range wl["servers"].Itfces {
			tngcidrv4 := ""
			sep := ""
			for _, cidr := range entry.Ipv4Cidr {
				tngcidrv4 = tngcidrv4 + sep + *cidr // TODO just a single CIDR is expected ... What shall we do!
				sep = " "
			}
			tngcidrv6 := ""
			sep = ""
			for _, cidr := range entry.Ipv6Cidr {
				tngcidrv6 = tngcidrv6 + sep + *cidr // TODO just a single CIDR is expected ... What shall we do!
				sep = " "
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

func processTNGCnfs(p *parser.Parser, tng *TngRoot) {
	for cnfname, cnf := range p.Config.Application["paco"].Cnfs {

		if cnf.Networking.AS == nil {
			continue
		}

		new_cnf := &TngCnf{}
		tng.Cnfs = append(tng.Cnfs, new_cnf)

		new_cnf.AsNumber = *cnf.Networking.AS
		if val, ok := p.Config.AppNetworkIndexes["itfce"][cnfname]["llb"]; ok {
			new_cnf.LlbPods = *val
		}
		if val, ok := p.Config.AppNetworkIndexes["itfce"][cnfname]["lmg"]; ok {
			new_cnf.LmgPods = *val
		}
		new_cnf.Name = cnfname
		new_cnf.Multus = append(new_cnf.Multus, &struct{ Name string }{Name: "FOO"}) // TODO
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
