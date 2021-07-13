package templating

import (
	"bytes"
	"encoding/json"
	"path"

	"text/template"

	"github.com/nokia-paco-automation/paco-parser/parser"
	"github.com/nokia-paco-automation/paco-parser/types"

	log "github.com/sirupsen/logrus"
)

func ProcessSwitchTemplates(wr *types.WorkloadResults, ir *types.InfrastructureResult, cg *types.ClientGroupResults, n map[string]*parser.Node) map[string]string {
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
			}
		}
	}

	// irbsub interfaces
	for nodename, irbsubifs := range wr.IrbSubInterfaces {
		for _, irbsubif := range irbsubifs {
			conf := processIrbSubInterfaces(irbsubif)
			templatenodes[nodename].AddSubInterface(irbsubif.InterfaceRealName, irbsubif.VlanID, conf)
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

	result := map[string]string{}

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
		result[name] = string(indentresult)
	}

	return result
}

func processEsi(esi *types.K8ssrlESI) string {
	buf := new(bytes.Buffer)
	t := template.Must(template.ParseFiles(path.Join("templates", "switch", "esi.tmpl")))

	err := t.ExecuteTemplate(buf, "esi", esi)
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

func processIrbSubInterfaces(irbsubif *types.K8ssrlirbsubinterface) string {
	buf := new(bytes.Buffer)
	t := template.Must(template.ParseFiles(path.Join("templates", "switch", "irbinterface.tmpl")))
	err := t.ExecuteTemplate(buf, "irbinterface", irbsubif)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	return buf.String()
}

func processInterface(nodename string, islinterfaces *types.K8ssrlinterface) string {
	buf := new(bytes.Buffer)
	t := template.Must(template.ParseFiles(path.Join("templates", "switch", "srlinterfaces.tmpl")))

	err := t.ExecuteTemplate(buf, "srlinterface", islinterfaces)
	if err != nil {
		log.Infof("%+v", err)
	}
	return buf.String()
}

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

func processSrlSubInterfaces(nodename string, interfacename string, srlsubif *types.K8ssrlsubinterface) string {
	t := template.Must(template.ParseFiles(path.Join("templates", "switch", "subinterfaces.tmpl")))
	buf := new(bytes.Buffer)
	err := t.ExecuteTemplate(buf, "subinterface", struct {
		InterfaceName string
		SubInterface  *types.K8ssrlsubinterface
		Target        string
	}{srlsubif.InterfaceRealName, srlsubif, nodename})
	if err != nil {
		log.Fatalf("%+v", err)
	}

	return buf.String()
}
