package cmd

import (
	"encoding/json"
	"fmt"

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

		templating.ProcessSwitchTemplates(workloadResults, infrastructureResult, clientGroupResults)

		infrajson, _ := json.MarshalIndent(infrastructureResult, "", "  ")
		fmt.Print(string(infrajson))

		cgjson, _ := json.MarshalIndent(clientGroupResults, "", "  ")
		fmt.Print(string(cgjson))

		wljson, _ := json.MarshalIndent(workloadResults, "", "  ")
		fmt.Print(string(wljson))

		//Write the server yaml files
		p.ParseServerData()

		//Write the values.yaml file for the respective applications in k8s
		p.ParseApplicationData()

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
