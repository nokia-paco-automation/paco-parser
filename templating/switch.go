package templating

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strconv"

	"strings"
	"text/template"

	"github.com/nokia-paco-automation/paco-parser/parser"
	"github.com/nokia-paco-automation/paco-parser/types"

	log "github.com/sirupsen/logrus"
)

func ProcessSwitchTemplates(wr *types.WorkloadResults, ir *types.InfrastructureResult, cg *types.ClientGroupResults, n map[string]*parser.Node) {
	log.Infof("ProcessingSwitchTemplates")

	configsnippets := map[string]map[string]string{}
	//interfaces := map[string]interface{}{}

	// routing policy & systeminterfaces
	for nodename, node := range n {
		if *node.Kind != "srl" {
			continue
		}
		// routing policies
		initDoubleMap(configsnippets, nodename, "routingpolicy")
		configsnippets[nodename]["routingpolicy"] = processRoutingPolicy(ir.RoutingPolicy)

		// system interfaces
		initDoubleMap(configsnippets, nodename, "interfaces-sys")
		configsnippets[nodename]["interfaces-sys"] = processInterfaces(nodename, ir.SystemInterfaces)

		// systemni
		for cgname, esis := range cg.Esis {
			initDoubleMap(configsnippets, nodename, "esi-"+cgname)
			configsnippets[nodename]["systemni-"+cgname] = processEsi(esis)
		}
	}

	// Clientinterfaces
	for nodename, data := range cg.ClientInterfaces {
		for cgname, interfs := range data {
			initDoubleMap(configsnippets, nodename, "interfaces-client")
			interfaceconf := processInterfaces(nodename, interfs)
			configsnippets[nodename]["interfaces-clientgroup-"+cgname] = interfaceconf
		}
	}

	// Infrastructure Interfaces
	for nodename, nodeInterfaces := range ir.IslInterfaces {
		initDoubleMap(configsnippets, nodename, "interfaces-isl")
		interfaceconf := processInterfaces(nodename, nodeInterfaces)
		configsnippets[nodename]["interfaces-isl"] = interfaceconf
	}

	//vxlaninterfaces
	for nodename, vxinterf := range wr.VxlanInterfaces {
		initDoubleMap(configsnippets, nodename, "vxlaninterfaces")
		configsnippets[nodename]["vxlaninterfaces"] = processVxlanInterfaces(ir.TunnelInterfaces[0].Name, vxinterf)
	}

	// process subinterfaces on a per interface basis
	for nodename, srlifs := range ir.IslSubInterfaces {
		for name, srlsubif := range srlifs {
			initDoubleMap(configsnippets, nodename, "srlsubinterface_"+strings.ReplaceAll(name, "/", "-"))
			configsnippets[nodename]["srlsubinterface_"+strings.ReplaceAll(name, "/", "-")] = processSrlSubInterfaces(srlsubif, nodename)
		}
	}

	// irbsub interfaces
	for nodename, irbsubif := range wr.IrbSubInterfaces {
		initDoubleMap(configsnippets, nodename, "irbsubinterfaces")
		configsnippets[nodename]["irbsubinterfaces"] = processIrbSubInterfaces(irbsubif)
	}

	// networkinstance
	for nodename, data := range wr.NetworkInstances {
		for niid, networkinstance := range data {
			initDoubleMap(configsnippets, nodename, "networkinstance-"+strconv.Itoa(niid))
			configsnippets[nodename]["networkinstance-"+strconv.Itoa(niid)] = processNetworkInstance(networkinstance)
		}
	}
	// bgp
	for nodename, bgp := range ir.DefaultProtocolBGP {
		initDoubleMap(configsnippets, nodename, "bgp-"+bgp.NetworkInstanceName)
		configsnippets[nodename]["bgp-"+bgp.NetworkInstanceName] = processBgp(bgp)
	}

	for nodename, data := range configsnippets {
		for partname, conf := range data {
			//fmt.Printf("%s, %s: %s", nodename, partname, conf)
			f, _ := os.Create("/tmp/conf/" + nodename + "-" + partname)
			f.WriteString(conf)
			f.Close()
		}
	}

	fmt.Println()
}

func processEsi(esi []*types.K8ssrlESI) string {
	buf := new(bytes.Buffer)
	t := template.Must(template.ParseFiles(path.Join("templates", "switch", "systemni.tmpl")))

	err := t.ExecuteTemplate(buf, "systemni", esi)
	if err != nil {
		log.Infof("%+v", err)
	}
	//fmt.Println(buf.String())
	return buf.String()
}

func processBgp(bgp *types.K8ssrlprotocolsbgp) string {
	buf := new(bytes.Buffer)
	t := template.Must(template.ParseFiles(path.Join("templates", "switch", "bgp.tmpl")))

	err := t.ExecuteTemplate(buf, "bgp", bgp)
	if err != nil {
		log.Infof("%+v", err)
	}
	//fmt.Println(buf.String())
	return buf.String()
}

func processNetworkInstance(networkinstance *types.K8ssrlNetworkInstance) string {
	buf := new(bytes.Buffer)
	t := template.Must(template.ParseFiles(path.Join("templates", "switch", "networkinstance.tmpl")))

	err := t.ExecuteTemplate(buf, "networkinstance", networkinstance)
	if err != nil {
		log.Infof("%+v", err)
	}
	//fmt.Println(buf.String())
	return buf.String()
}

func processIrbSubInterfaces(irbsubif []*types.K8ssrlirbsubinterface) string {
	buf := new(bytes.Buffer)
	t := template.Must(template.ParseFiles(path.Join("templates", "switch", "irbinterfaces.tmpl")))
	err := t.ExecuteTemplate(buf, "irbinterfaces", struct {
		SubInterfaces []*types.K8ssrlirbsubinterface
		InterfaceName string
	}{SubInterfaces: irbsubif, InterfaceName: irbsubif[0].InterfaceRealName})
	if err != nil {
		log.Fatalf("%+v", err)
	}
	return buf.String()
}

func processInterfaces(nodename string, islinterfaces []*types.K8ssrlinterface) string {
	buf := new(bytes.Buffer)
	t := template.Must(template.ParseFiles(path.Join("templates", "switch", "srlinterfaces.tmpl")))

	err := t.ExecuteTemplate(buf, "srlinterfaces", islinterfaces)
	if err != nil {
		log.Infof("%+v", err)
	}
	//fmt.Println(buf.String())
	return buf.String()
}

// func processTunnelInterfaces(tuninterfs []*types.K8ssrlTunnelInterface) string {
// 	buf := new(bytes.Buffer)
// 	t := template.Must(template.New("tunnelinterfaces.tmpl").ParseFiles(path.Join("templates", "switch", "tunnelinterfaces.tmpl")))
// 	err := t.ExecuteTemplate(buf, "tunnelinterfaces.tmpl", tuninterfs)
// 	if err != nil {
// 		log.Infof("%+v", err)
// 	}
// 	return buf.String()
// }

func processVxlanInterfaces(tunifname string, vxinterf []*types.K8ssrlVxlanInterface) string {
	buf := new(bytes.Buffer)
	t := template.Must(template.New("vxlanInterfaces.tmpl").ParseFiles(path.Join("templates", "switch", "vxlanInterfaces.tmpl")))
	err := t.ExecuteTemplate(buf, "vxlanInterfaces.tmpl", struct {
		TunnelInterfaceName string
		VxlanInterfaces     []*types.K8ssrlVxlanInterface
	}{TunnelInterfaceName: tunifname, VxlanInterfaces: vxinterf})
	if err != nil {
		log.Infof("%+v", err)
	}

	return buf.String()
}

func processRoutingPolicy(rp *types.K8ssrlRoutingPolicy) string {
	buf := new(bytes.Buffer)
	t := template.Must(template.New("routingpolicy.tmpl").ParseFiles(path.Join("templates", "switch", "routingpolicy.tmpl")))
	err := t.ExecuteTemplate(buf, "routingpolicy.tmpl", rp)
	if err != nil {
		log.Infof("%+v", err)
	}
	return buf.String()
}

func processSrlSubInterfaces(srlsubifs []*types.K8ssrlsubinterface, nodename string) string {
	t := template.Must(template.ParseFiles(path.Join("templates", "switch", "subinterfaces.tmpl")))
	buf := new(bytes.Buffer)
	err := t.ExecuteTemplate(buf, "subinterfaces", struct {
		InterfaceName string
		SubInterfaces []*types.K8ssrlsubinterface
		Target        string
	}{srlsubifs[0].InterfaceRealName, srlsubifs, nodename})
	if err != nil {
		log.Fatalf("%+v", err)
	}

	return buf.String()
}
