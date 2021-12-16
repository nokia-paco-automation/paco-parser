package cmd

import (
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/nokia-paco-automation/paco-parser/parser"
	"github.com/nokia-paco-automation/paco-parser/templating"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// parseCmd represents the parse command
var parseCmd = &cobra.Command{
	Use:          "parse",
	Short:        "parse a paco deployment file",
	Long:         "parse a paco deployment by means of the deployment definition file",
	Aliases:      []string{"dep"},
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if err = configSet(); err != nil {
			return err
		}
		opts := []parser.ParserOption{
			parser.WithDebug(debug),
			parser.WithConfigFile(&config),
			parser.WithOutput(&output),
		}
		p := parser.NewParser(opts...)

		setFlags(p.Config)

		// initialize IPAM for the inter switch links (isl) links and elements
		netwInfo := &parser.NetworkInfo{
			Kind:                  parser.StringPtr("isl"),
			AddressingSchema:      p.Config.Infrastructure.AddressingSchema,
			Ipv4Cidr:              p.Config.Infrastructure.Networks["isl"].Ipv4Cidr,
			Ipv4ItfcePrefixLength: p.Config.Infrastructure.Networks["isl"].Ipv4ItfcePrefixLength,
			Ipv6Cidr:              p.Config.Infrastructure.Networks["isl"].Ipv6Cidr,
			Ipv6ItfcePrefixLength: p.Config.Infrastructure.Networks["isl"].Ipv6ItfcePrefixLength,
		}

		p.IPAM["isl"], err = parser.NewIPAM(netwInfo)
		if err != nil {
			log.Fatal(err)
		}
		// initialize IPAM for the loopbacks of the network elements
		netwInfo = &parser.NetworkInfo{
			Kind:                  parser.StringPtr("loopback"),
			AddressingSchema:      p.Config.Infrastructure.AddressingSchema,
			Ipv4Cidr:              p.Config.Infrastructure.Networks["loopback"].Ipv4Cidr,
			Ipv4ItfcePrefixLength: p.Config.Infrastructure.Networks["loopback"].Ipv4ItfcePrefixLength,
			Ipv6Cidr:              p.Config.Infrastructure.Networks["loopback"].Ipv6Cidr,
			Ipv6ItfcePrefixLength: p.Config.Infrastructure.Networks["loopback"].Ipv6ItfcePrefixLength,
		}
		p.IPAM["loopback"], err = parser.NewIPAM(netwInfo)
		if err != nil {
			log.Fatal(err)
		}

		// initialize IPAM for the loop of the network elements
		netwInfo = &parser.NetworkInfo{
			Kind:                  parser.StringPtr("loop"),
			AddressingSchema:      p.Config.Infrastructure.AddressingSchema,
			Ipv4Cidr:              p.Config.Infrastructure.Networks["loop"].Ipv4Cidr,
			Ipv4ItfcePrefixLength: p.Config.Infrastructure.Networks["loop"].Ipv4ItfcePrefixLength,
			Ipv6Cidr:              p.Config.Infrastructure.Networks["loop"].Ipv6Cidr,
			Ipv6ItfcePrefixLength: p.Config.Infrastructure.Networks["loop"].Ipv6ItfcePrefixLength,
		}
		p.IPAM["loop"], err = parser.NewIPAM(netwInfo)
		if err != nil {
			log.Fatal(err)
		}

		// Parse the topology part of the configuration
		if err = p.ParseTopology(); err != nil {
			return err
		}
		//p.ShowTopology()

		if err = p.ParseClientGroup(); err != nil {
			return err
		}
		//p.ShowClientGroup()

		if err := p.InitializeIPAMWorkloads(); err != nil {
			log.Fatal(err)
		}

		// Parse the workload part of the configuration
		/*
			if err = p.ParseWorkload(); err != nil {
				return err
			}
			p.ShowWorkload()
		*/

		// Write the switch configuration in K8s
		p.WriteBase()
		// holds a structure with all directories that are used by kustomize
		var kdirs []string

		kd, infrastructureResult := p.WriteInfrastructure()
		kdirs = append(kdirs, kd...)

		kd, clientGroupResults := p.WriteClientsGroups()
		kdirs = append(kdirs, kd...)

		kd, workloadResults := p.WriteWorkloads()
		kdirs = append(kdirs, kd...)

		p.WriteFinalBase(kdirs)

		_ = infrastructureResult
		_ = clientGroupResults
		_ = workloadResults

		// infrajson, _ := json.MarshalIndent(infrastructureResult, "", "  ")
		// fmt.Print(string(infrajson))
		// f, err := os.Create("/tmp/infrastructureResult")
		// f.WriteString(string(infrajson))
		// f.Close()

		// cgjson, _ := json.MarshalIndent(clientGroupResults, "", "  ")
		// fmt.Print(string(cgjson))
		// f, err = os.Create("/tmp/clientGroupResults")
		// f.WriteString(string(cgjson))
		// f.Close()

		// wljson, _ := json.MarshalIndent(workloadResults, "", "  ")
		// fmt.Print(string(wljson))
		// f, err = os.Create("/tmp/workloadResults")
		// f.WriteString(string(wljson))
		// f.Close()

		//Write the server yaml files
		p.ParseServerData()

		//Write the values.yaml file for the respective applications in k8s
		appConfig := p.ParseApplicationData()
		_ = appConfig

		srl_configs, looptngresult := templating.ProcessSwitchTemplates(workloadResults, infrastructureResult, clientGroupResults, p.Nodes, appConfig, p.Config.Application["paco"].Global.Multus, p.Config, p)
		writeSwitchConfigs(srl_configs)

		_ = looptngresult
		//writeTngFile(templating.ProcessTNG(p, workloadResults, infrastructureResult, clientGroupResults, appConfig, looptngresult))

		writeNokiaYml(p.Config)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)
}

// setFlags provides an override capability from the commandline
func setFlags(conf *parser.Config) {
	if name != "" {
		conf.Name = &name
	}
}

func writeSwitchConfigs(srl_configs map[string]string) {
	confdir := path.Join(output, "switch-full")
	os.MkdirAll(confdir, 0777)
	for devicename, config := range srl_configs {
		f, err := os.Create(path.Join(confdir, devicename+".json"))
		if err != nil {
			log.Fatalf("%v", err)
		}
		f.WriteString(config)
		f.Close()
	}
}

type NokiaYmlXtraData struct {
	Nodes     map[string]*NokiaYmlNode
	Config    *parser.Config
	Workloads map[string]*NokiaYmlWorkload
}

type NokiaYmlInterface struct {
	Name      string
	Sriov     bool
	Ipvlan    bool
	Eth_names []string
}

type NokiaYmlNode struct {
	Name       string
	Nodetype   string
	Mgmt_ip    string
	Interfaces map[string]*NokiaYmlInterface
}

type NokiaYmlWorkload struct {
	Name string
	VID  int
}

func (nyn *NokiaYmlNode) add_bond_subinterface(bond_name string, subinterface_name string, sriov bool, ipvlan bool) {
	if _, exists := nyn.Interfaces[bond_name]; !exists {
		nyn.Interfaces[bond_name] = &NokiaYmlInterface{
			Name:      bond_name,
			Sriov:     sriov,
			Ipvlan:    ipvlan,
			Eth_names: []string{},
		}
	}
	nyn.Interfaces[bond_name].Eth_names = append(nyn.Interfaces[bond_name].Eth_names, subinterface_name)
}

func SkipLinkConditionCheck(nodeA string, nodeB string) bool {
	SkipCombinations := [][]string{
		{"leaf", "infra"},
		{"infra", "infra"},
		{"infra", "dcgw"},
		{"leaf", "dcgw"},
	}
	for _, data1 := range SkipCombinations {
		if strings.Contains(nodeA, data1[0]) && strings.Contains(nodeB, data1[1]) {
			return true
		}
		if strings.Contains(nodeB, data1[0]) && strings.Contains(nodeA, data1[1]) {
			return true
		}
	}
	return false
}

func writeNokiaYml(config *parser.Config) {
	nokia_yml_xtra_data := &NokiaYmlXtraData{
		Nodes:     map[string]*NokiaYmlNode{},
		Config:    config,
		Workloads: map[string]*NokiaYmlWorkload{},
	}

	for nodename, nodedata := range config.Topology.Nodes {
		node := &NokiaYmlNode{
			Name:       nodename,
			Mgmt_ip:    *nodedata.MgmtIPv4,
			Interfaces: map[string]*NokiaYmlInterface{},
		}

		nokia_yml_xtra_data.Nodes[nodename] = node
		switch {
		case strings.Contains(nodename, "worker"):
			node.Nodetype = "worker"
		case strings.Contains(nodename, "master"):
			node.Nodetype = "master"
		case strings.Contains(nodename, "leaf"):
			node.Nodetype = "leaf"
		}
	}

	for _, lc := range config.Topology.Links {
		if SkipLinkConditionCheck(*lc.Endpoints[0], *lc.Endpoints[1]) {
			continue
		}
		for _, ep := range lc.Endpoints {
			ep_info := strings.Split(*ep, ":")
			if _, exists := nokia_yml_xtra_data.Nodes[ep_info[0]]; !exists {
				continue
			}
			if nokia_yml_xtra_data.Nodes[ep_info[0]].Nodetype != "leaf" {
				sriov, err := strconv.ParseBool(*lc.Labels["sriov"])
				if err != nil {
					log.Fatal(err)
				}
				ipvlan, err := strconv.ParseBool(*lc.Labels["ipvlan"])
				if err != nil {
					log.Fatal(err)
				}

				bond_name := *lc.Labels["client-name"]
				nokia_yml_xtra_data.Nodes[ep_info[0]].add_bond_subinterface(bond_name, ep_info[1], sriov, ipvlan)
			}
		}
	}

	for wl_name, wl_data := range config.Workloads {
		if _, exists := wl_data["servers"].Itfces["ipvlan"]; !exists {
			continue
		}

		nokia_yml_xtra_data.Workloads[wl_name] = &NokiaYmlWorkload{
			Name: wl_name,
			VID:  *wl_data["servers"].Itfces["ipvlan"].VlanID,
		}
	}

	templateFile := path.Join("templates", "Nokia.yml.tmpl")
	data := templating.GeneralTemplateProcessing(templateFile, "nokia", nokia_yml_xtra_data)
	conf := path.Join(output, "Nokia.yml")
	f, err := os.Create(path.Join(conf))
	if err != nil {
		log.Fatalf("%v", err)
	}
	f.WriteString(data)
	f.Close()
}

func writeTngFile(data string) {
	conf := path.Join(output, "tng.yml")
	f, err := os.Create(path.Join(conf))
	if err != nil {
		log.Fatalf("%v", err)
	}
	f.WriteString(data)
	f.Close()
}
