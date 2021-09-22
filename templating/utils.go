package templating

import (
	"bytes"
	"log"
	"net"
	"text/template"

	"github.com/nokia-paco-automation/paco-parser/types"
)

func initDoubleMap(data map[string]map[string]string, key1 string, key2 string) {
	if _, ok := data[key1]; !ok {
		data[key1] = map[string]string{}
	}
	if _, ok := data[key1][key2]; !ok {
		data[key1][key2] = ""
	}
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
	appIp, _, err := net.ParseCIDR(ipv4 + "/32")
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
					//fmt.Printf("MATCH: Ipv4: %s, Net %s\n", ipv4, irbnet.String())
					return irbsubif
				}
			}
		}
	}
	log.Fatalln("not found!")
	return nil
}

func findNetworkInstanceOfSRLInterface(networkinstances map[string]map[int]*types.K8ssrlNetworkInstance, ipv4 string) *NetworkInstanceLookupResult {
	appIp, _, err := net.ParseCIDR(ipv4 + "/32")
	if err != nil {
		log.Fatalln("Not a valid IP.")
	}
	for nodename, networkinstances := range networkinstances {
		for _, ni := range networkinstances {
			for _, subif := range ni.SubInterfaces {
				if subif.IPv4Prefix == "" {
					continue
				}
				_, accessnet, err := net.ParseCIDR(subif.IPv4Prefix)
				if err != nil {
					log.Fatalln("IP Parsing error")
				}
				if accessnet.Contains(appIp) {
					//fmt.Printf("MATCH: Ipv4: %s, Net %s\n", ipv4, accessnet.String())
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

func GeneralTemplateProcessing(templateFile string, templateName string, data interface{}) string {
	t := template.Must(template.ParseFiles(templateFile))
	buf := new(bytes.Buffer)
	err := t.ExecuteTemplate(buf, templateName, data)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	return buf.String()
}

func incrementIP(ip net.IP) (result net.IP) {
	result = make([]byte, len(ip)) // start off with a nice empty ip of proper length

	carry := true
	for i := len(ip) - 1; i >= 0; i-- {
		result[i] = ip[i]
		if carry {
			result[i]++
			if result[i] != 0 {
				carry = false
			}
		}
	}
	return
}
