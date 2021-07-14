package templating

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/nokia-paco-automation/paco-parser/types"
)

func mergeJson(configsnippets map[string]map[string]string) {
	results := map[string]string{}
	for nodename, nodedata := range configsnippets {
		results[nodename] = "{}"
		fmt.Printf("Node: %s\n", nodename)
		for snippetname, data := range nodedata {
			results[nodename] = runCommand(results[nodename], "{"+data+"}")
			fmt.Printf("%s %d\n", snippetname, len(results[nodename]))
		}
	}
	for x, y := range configsnippets["leaf1"] {
		f, _ := os.Create("/tmp/leaf1/" + x)
		f.WriteString("{" + y + "}")
		f.Close()
	}

	for x, y := range results {
		fmt.Printf("%s: %+v", x, string(y))
	}
	//fmt.Printf("%+v\n", results)
}

func runCommand(data1 string, data2 string) string {
	file1, err := ioutil.TempFile(os.TempDir(), "json-paco-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(file1.Name())

	file2, err := ioutil.TempFile(os.TempDir(), "json-paco-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(file2.Name())

	_, err = file1.WriteString(data1)
	if err != nil {
		log.Fatalf("%v", err)
	}

	_, err = file2.WriteString(data2)
	if err != nil {
		log.Fatalf("%v", err)
	}

	cmd := exec.Command("jq", "-s", ".[0] * .[1]", file1.Name(), file2.Name())
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		log.Fatalf("%v", err)
	}
	return out.String()
}

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
	appIp, _, err := net.ParseCIDR(ipv4)
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
					fmt.Printf("MATCH: Ipv4: %s, Net %s\n", ipv4, irbnet.String())
					return irbsubif
				}
			}
		}
	}
	log.Fatalln("not found!")
	return nil
}
