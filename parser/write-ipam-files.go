package parser

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	log "github.com/sirupsen/logrus"
)

func (p *Parser) WriteApplicationDeploymentIPAM(dirName *string) {

	csvDataIPv4 := make([][]string, 0)
	csvDataIPv6 := make([][]string, 0)
	csvRowStart := []string{"wlName", "NetworkType", "Subnet", "VlanID", "Gateway", "IP Address", "Application", "Uages"}
	csvDataIPv4 = append(csvDataIPv4, csvRowStart)
	csvDataIPv6 = append(csvDataIPv6, csvRowStart)

	for wlName, netwInfo := range p.DeploymentIPAM {
		//fmt.Printf("Workload name: %s \n", wlName)
		for netwType, subnetInfo := range netwInfo {
			//fmt.Printf("Network Type: %s \n", netwType)
			for subnet, ipamInfo := range subnetInfo {
				ipType := ip4or6(subnet)

				csvRowPrep := make([]string, 0)
				csvRowPrep = append(csvRowPrep, wlName)
				csvRowPrep = append(csvRowPrep, netwType)
				csvRowPrep = append(csvRowPrep, subnet)

				if ipamInfo.Gateway != nil {
					if ipamInfo.VlanID != nil {
						//fmt.Printf("Subnet: %s, IpGateway: %s, VlanID %d\n", subnet, *ipamInfo.Gateway, *ipamInfo.VlanID)
						csvRowPrep = append(csvRowPrep, strconv.Itoa(*ipamInfo.VlanID))
						csvRowPrep = append(csvRowPrep, *ipamInfo.Gateway)
					} else {
						//fmt.Printf("Subnet: %s, IpGateway: %s\n", subnet, *ipamInfo.Gateway)
						csvRowPrep = append(csvRowPrep, "") // empty vlan
						csvRowPrep = append(csvRowPrep, *ipamInfo.Gateway)
					}
				} else {
					//fmt.Printf("Subnet: %s:\n", subnet)
					if ipamInfo.VlanID != nil {
						csvRowPrep = append(csvRowPrep, strconv.Itoa(*ipamInfo.VlanID))
					} else {
						csvRowPrep = append(csvRowPrep, "") // empty vlan
					}
					csvRowPrep = append(csvRowPrep, "") // empty gateway
				}
				//fmt.Printf("Subnet: %s, VlanId: %d, Ipv4Gateway: %s, Ipv6Gateway: %s \n", subnet, *ipamInfo.VlanID, ipamInfo.Ipv4Gateway.String(), ipamInfo.Ipv6Gateway.String())
				if len(ipamInfo.AllocatedIPs) == 0 {
					//log.Info("Empty csvRow: %v", csvRowPrep)
					csvRow := make([]string, 0)
					csvRow = append(csvRow, csvRowPrep...)
					csvRow = append(csvRow, "")
					csvRow = append(csvRow, "unused")
					csvRow = append(csvRow, "unused")
					switch ipType {
					case "ipv4":
						csvDataIPv4 = append(csvDataIPv4, csvRowPrep)
					case "ipv6":
						csvDataIPv6 = append(csvDataIPv6, csvRowPrep)
					default:
						log.Error("Wrong type")
					}
				}
				for _, ipAlloc := range ipamInfo.AllocatedIPs {
					csvRow := make([]string, 0)
					csvRow = append(csvRow, csvRowPrep...)
					csvRow = append(csvRow, *ipAlloc.IPAddress)
					csvRow = append(csvRow, *ipAlloc.Application)
					csvRow = append(csvRow, *ipAlloc.Usage)

					switch ipType {
					case "ipv4":
						csvDataIPv4 = append(csvDataIPv4, csvRow)
					case "ipv6":
						csvDataIPv6 = append(csvDataIPv6, csvRow)
					default:
						log.Error("Wrong type")
					}

					//fmt.Printf("Allocated IP: %s, App: %s, Usage: %s \n", *ipAlloc.IPAddress, *ipAlloc.Application, *ipAlloc.Usage)
					//log.Info("csvRow: %v", csvRow)
				}

			}
		}
	}
	log.Info("Dumping CSV Data...")
	log.Infof("length ipv4 csvData: %d", len(csvDataIPv4))
	if len(csvDataIPv4) > 1 {
		p.writeIpamFile(dirName, StringPtr("ipv4"), &csvDataIPv4)
	}
	log.Infof("length ipv6 csvData: %d", len(csvDataIPv6))
	if len(csvDataIPv6) > 1 {
		p.writeIpamFile(dirName, StringPtr("ipv6"), &csvDataIPv6)
	}

	log.Infof("Writing application ipam csv file...")

}

func (p *Parser) writeIpamFile(dirName, version *string, csvData *[][]string) {
	p.CreateDirectory(*dirName, 0777)
	fileName := fmt.Sprintf("paco-ipam-%s.csv", *version)
	csvFile, err := os.Create(filepath.Join(*dirName, filepath.Base(fileName)))
	if err != nil {
		log.WithError(err).Infof("Cannot create file")
	}

	csvwriter := csv.NewWriter(csvFile)

	for _, row := range *csvData {
		//log.Info("csvRow: %v", row)
		err = csvwriter.Write(row)
		if err != nil {
			log.WithError(err).Infof("Cannot write csv file")
		}
	}
	csvwriter.Flush()
	csvFile.Close()
}
