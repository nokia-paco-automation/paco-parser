package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/nokia-paco-automation/paco-parser/types"
	log "github.com/sirupsen/logrus"
	"github.com/stoewer/go-strcase"
)

var (
	goK8sKustomizeTemplate = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
{{- range $index, $element := .Resources}}
- {{$element}}
{{- end}}
`
	goK8sSrlinterfaceTemplate = `
apiVersion: srlinux.henderiw.be/v1alpha1
kind: K8sSrlNokiaInterfacesInterface
metadata:
  name: {{.ResourceName}}
  labels:
    target: {{.Target}}
spec:
  interface:
{{- range $index, $element := .Interfaces}}
  - name: {{$element.Name}}
    admin-state: enable
    description: "paco-{{$element.Name}}"
{{- if $element.VlanTagging}}
    vlan-tagging: {{$element.VlanTagging}}
{{- end}}
{{- if ne $element.PortSpeed ""}}
{{- if not $element.Lag}}
    ethernet:
      port-speed: {{$element.PortSpeed}}
{{- if eq $element.LagMember true }}
      aggregate-id: {{$element.LagName}}
{{- end}}
{{- end}}
{{- end}}
{{- if eq $element.Lag true}}
    lag:
      lag-type: lacp
      member-speed: {{$element.PortSpeed}}
{{- if eq $element.Pxe true}}
      lacp-fallback-mode: static
{{- end}}
      lacp:
        interval: FAST
        lacp-mode: ACTIVE
        admin-key: {{$element.AdminKey}}
        system-id-mac: {{$element.SystemMac}}
{{- end}}
{{- end}}
`
	goK8sSrlTunnelInterfaceTemplate = `
apiVersion: srlinux.henderiw.be/v1alpha1
kind: K8sSrlNokiaTunnelInterfacesTunnelInterface
metadata:
  name: {{.ResourceName}}
  labels:
    target: {{.Target}}
spec:
  tunnel-interface:
{{- range $index, $element := .TunnelInterfaces}}
  - name: {{$element.Name}}
{{- end}}
`

	goK8sSrlVxlanInterfaceTemplate = `
apiVersion: srlinux.henderiw.be/v1alpha1
kind: K8sSrlNokiaTunnelInterfacesTunnelInterfaceVxlanInterface
metadata:
  name: {{.ResourceName}}
  labels:
    target: {{.Target}}
spec:
  tunnel-interface-name: {{.TunnelInterfaceName}}
  vxlan-interface:
{{- range $index, $element := .VxlanInterfaces}}
  - index: {{$element.VlanID}}
    type: {{$element.Kind}}
    ingress:
{{- if eq $element.VlanID "0"}}
      vni: 1
{{- else}}
      vni: {{$element.VlanID}}
{{- end}}
    egress:
      source-ip: use-system-ipv4-address
{{- end}}
`

	goK8sSrlsubinterfaceTemplate = `
apiVersion: srlinux.henderiw.be/v1alpha1
kind: K8sSrlNokiaInterfacesInterfaceSubinterface
metadata:
  name: {{.ResourceName}}
  labels:
    target: {{.Target}}
spec:
  interface-name: {{.InterfaceName}}
{{- $target := .Target}}
  subinterface:
{{- range $index, $element := .SubInterfaces}}
  - index: {{$element.VlanID}}
{{- if ne $element.Kind "loopback"}}
    type: {{$element.Kind}}
{{- end}}
    admin-state: enable
    description: "paco-{{$element.InterfaceShortName}}-{{$element.VlanID}}-{{$target}}"
{{- if $element.VlanTagging}}
    vlan:
      encap:
{{- if eq $element.VlanID "0"}}
        untagged: {}
{{- else}}
        single-tagged:
          vlan-id: "{{$element.VlanID}}"
{{- end}}
{{- end}}
{{- if eq $element.Kind "routed" "loopback"}}
    ipv4:
      address: 
      - ip-prefix: {{$element.IPv4Prefix}}
    ipv6:
      address: 
      - ip-prefix: {{$element.IPv6Prefix}}
{{- end}}
{{- end}}
`

	goK8sSrlIrbSubInterfaceTemplate = `
apiVersion: srlinux.henderiw.be/v1alpha1
kind: K8sSrlNokiaInterfacesInterfaceSubinterface
metadata:
  name: {{.ResourceName}}
  labels:
    target: {{.Target}}
spec:
  interface-name: {{.InterfaceName}}
{{- $target := .Target}}
  subinterface:
{{- range $index, $element := .SubInterfaces}}
  - index: {{$element.VlanID}}
    admin-state: enable
    description: "{{$element.Description}}"
{{- if ne (len $element.IPv4Prefix) 0 }}
    ipv4:
      address: 
{{- range $index, $ipv4prefix := $element.IPv4Prefix}}
      - ip-prefix: {{$ipv4prefix}}
{{- if eq $element.AnycastGW true}}
        anycast-gw: true
{{- end}}
{{- end}}
      arp:
        learn-unsolicited: true
        host-route:
          populate:
          - route-type: dynamic
        evpn:
          advertise: 
          - route-type: dynamic
{{- end}}
{{- if eq $element.AnycastGW true}}
    anycast-gw:
      virtual-router-id: {{$element.VrID}}
{{- end}}
{{- if ne (len $element.IPv6Prefix) 0 }}
    ipv6:
      address: 
{{- range $index, $ipv6prefix := $element.IPv6Prefix}}
      - ip-prefix: {{$ipv6prefix}}
{{- if eq $element.AnycastGW true}}
        anycast-gw: true
{{- end}}
{{- end}}
      neighbor-discovery:
        learn-unsolicited: both
        host-route:
          populate:
          - route-type: dynamic
        evpn:
          advertise: 
          - route-type: dynamic
{{- end}}
{{- end}}
`

	goK8sSrlnetworkinstanceTemplate = `
apiVersion: srlinux.henderiw.be/v1alpha1
kind: K8sSrlNokiaNetworkInstanceNetworkInstance
metadata:
  name: {{.ResourceName}}
  labels:
    target: {{.Target}}
spec:
  network-instance:
  - name: {{.NetworkInstance.Name}}
    type: {{.NetworkInstance.Kind}}
    admin-state: enable
    description: paco-{{.NetworkInstance.Name}}
    interface:
{{- range $index, $element := .NetworkInstance.SubInterfaces}}
    - name: {{$element.InterfaceRealName}}.{{$element.VlanID}}
{{- end}}
{{- if .NetworkInstance.TunnelInterfaceName}}
    vxlan-interface:
    - name: {{.NetworkInstance.TunnelInterfaceName}}
{{- end}}
{{- if eq .NetworkInstance.Kind "mac-vrf"}}
    bridge-table:
      mac-duplication:
        admin-state: enable
        action: blackhole
{{- end}}
`

	goK8sSrlprotocolsbgpTemplate = `
apiVersion: srlinux.henderiw.be/v1alpha1
kind: K8sSrlNokiaNetworkInstanceNetworkInstanceProtocolsBgp
metadata:
  name: {{.ResourceName}}
  labels:
    target: {{.Target}}
spec:
  network-instance-name: {{.ProtocolBgp.NetworkInstanceName}}
  bgp:
    admin-state: enable
    autonomous-system: {{.ProtocolBgp.AS}}
    router-id: {{.ProtocolBgp.RouterID}}
    ebgp-default-policy:
      import-reject-all: false
      export-reject-all: false
    group: 
{{- range $index, $element := .ProtocolBgp.PeerGroups}}
    - group-name: {{$element.Name}}
{{- if $element.PolicyName }}
      export-policy: {{$element.PolicyName}}
{{- end}}
      admin-state: enable
      next-hop-self: true
{{- range $index, $protocol := $element.Protocols}}
      {{$protocol}}:
        admin-state: enable
{{- end}}
{{- end}}
    ipv4-unicast:
      admin-state: enable
      multipath:
        allow-multiple-as: true
        max-paths-level-1: 64
        max-paths-level-2: 64
    ipv6-unicast:
      admin-state: enable
      multipath:
        allow-multiple-as: true
        max-paths-level-1: 64
        max-paths-level-2: 64
    neighbor:
{{- range $index, $element := .ProtocolBgp.Neighbors}}
    - peer-address: {{$element.PeerIP}}
      peer-as: {{$element.PeerAS}}
      peer-group: {{$element.PeerGroup}}
{{- if ne $element.LocalAS 0}}
      local-as:
      - as-number: {{$element.LocalAS}}
{{- end}}
{{- if ne $element.TransportAddress ""}}
      transport:
        local-address: {{$element.TransportAddress}}
{{- end}}
{{- end}}
`
	goK8sSrlSystemNetworkInstanceTemplate = `
apiVersion: srlinux.henderiw.be/v1alpha1
kind: K8sSrlNokiaSystemSystemNetworkInstance
metadata:
  name: {{.ResourceName}}
  labels:
    target: {{.Target}}
spec:
  network-instance:
    protocols:
      bgp-vpn:
        bgp-instance:
        - id: 1
      evpn:
        ethernet-segments:
          bgp-instance:
          - id: "1"
            ethernet-segment:
{{- range $index, $element := .ESIs}}
            - name: {{$element.ESI}}
              admin-state: enable
              interface: {{$element.LagName}}
{{- end}}
`
	goK8sSrlNetworkInstanceBgpVpnTemplate = `
apiVersion: srlinux.henderiw.be/v1alpha1
kind: K8sSrlNokiaNetworkInstanceNetworkInstanceProtocolsBgpVpn
metadata:
  name: {{.ResourceName}}
  labels:
    target: {{.Target}}
spec:
  network-instance-name: {{.NetworkInstanceProtocol.Name}}
  bgp-vpn:
    bgp-instance:
    - id: 1
      route-target:
        export-rt: {{.NetworkInstanceProtocol.RouteTarget}}
        import-rt: {{.NetworkInstanceProtocol.RouteTarget}}
`

	goK8sSrlNetworkInstanceBgpEvpnTemplate = `
apiVersion: srlinux.henderiw.be/v1alpha1
kind: K8sSrlNokiaNetworkInstanceNetworkInstanceProtocolsBgpevpn
metadata:
  name: {{.ResourceName}}
  labels:
    target: {{.Target}}
spec:
  network-instance-name: {{.NetworkInstanceProtocol.Name}}
  bgp-evpn:
    bgp-instance:
    - id: "1"
      admin-state: enable
      ecmp: 8
      evi: {{.NetworkInstanceProtocol.Evi}}
      vxlan-interface: {{.NetworkInstanceProtocol.TunnelInterfaceName}}
`

	goK8sSrlNetworkInstanceLinuxTemplate = `
apiVersion: srlinux.henderiw.be/v1alpha1
kind: K8sSrlNokiaNetworkInstanceNetworkInstanceProtocolsLinux
metadata:
  name: {{.ResourceName}}
  labels:
    target: {{.Target}}
spec:
  network-instance-name: {{.NetworkInstanceProtocol.Name}}
  linux:
    export-neighbors: true
    export-routes: true
    import-routes: true
`

	goK8sSrlRoutingPoliciesTemplate = `
apiVersion: srlinux.henderiw.be/v1alpha1
kind: K8sSrlNokiaRoutingPolicyRoutingPolicy
metadata:
  name: {{.ResourceName}}
  labels:
    target: {{.Target}}
spec:
  routing-policy:
    prefix-set:
{{- if .RoutingPolicy.IPv4Prefix}}
    - name: {{.RoutingPolicy.IPv4PrefixSetName}}
      prefix: 
      - ip-prefix: {{.RoutingPolicy.IPv4Prefix}}
        mask-length-range: 32..32
{{- end }}
{{- if .RoutingPolicy.IPv6Prefix}}
    - name: {{.RoutingPolicy.IPv6PrefixSetName}}
      prefix: 
      - ip-prefix: {{.RoutingPolicy.IPv6Prefix}}
        mask-length-range: 128..128
{{- end }}
    policy:
    - name: {{.RoutingPolicy.Name}}
      statement:
{{- if .RoutingPolicy.IPv4Prefix}}
      - sequence-id: 10
        match:
          prefix-set: {{.RoutingPolicy.IPv4PrefixSetName}}
        action:
          accept: {}
{{- end }}
{{- if .RoutingPolicy.IPv6Prefix}}
      - sequence-id: 20
        match:
          prefix-set: {{.RoutingPolicy.IPv6PrefixSetName}}
        action:
          accept: {}
{{- end }}
`

	goTemplates = map[string]*template.Template{
		"kustomize":                 makek8sTemplate("kustomize", goK8sKustomizeTemplate),
		"srlInterface":              makek8sTemplate("srlInterface", goK8sSrlinterfaceTemplate),
		"srlSubInterface":           makek8sTemplate("srlSubInterface", goK8sSrlsubinterfaceTemplate),
		"srlIrbSubInterface":        makek8sTemplate("srlIrbSubInterface", goK8sSrlIrbSubInterfaceTemplate),
		"srlTunnelInterface":        makek8sTemplate("srlTunnelInterface", goK8sSrlTunnelInterfaceTemplate),
		"srlVxlanInterface":         makek8sTemplate("srlVxlanInterface", goK8sSrlVxlanInterfaceTemplate),
		"srlNetworkInstance":        makek8sTemplate("srlNetworkInstance", goK8sSrlnetworkinstanceTemplate),
		"srlProtocolsBgp":           makek8sTemplate("srlProtocolsBgp", goK8sSrlprotocolsbgpTemplate),
		"srlSystemNetworkInstance":  makek8sTemplate("srlSystemNetworkInstance", goK8sSrlSystemNetworkInstanceTemplate),
		"srlNetworkInstanceBgpVpn":  makek8sTemplate("srlNetworkInstanceBgpVpn", goK8sSrlNetworkInstanceBgpVpnTemplate),
		"srlNetworkInstanceBgpEvpn": makek8sTemplate("srlNetworkInstanceBgpEvpn", goK8sSrlNetworkInstanceBgpEvpnTemplate),
		"srlNetworkInstanceLinux":   makek8sTemplate("srlNetworkInstanceLinux", goK8sSrlNetworkInstanceLinuxTemplate),
		"srlRoutingPolicy":          makek8sTemplate("srlRoutingPolicy", goK8sSrlRoutingPoliciesTemplate),
	}

	// templateHelperFunctions specifies a set of functions that are supplied as
	// helpers to the templates that are used within this file.
	templateHelperFunctions = template.FuncMap{
		// inc provides a means to add 1 to a number, and is used within templates
		// to check whether the index of an element within a loop is the last one,
		// such that special handling can be provided for it (e.g., not following
		// it with a comma in a list of arguments).
		"inc":  func(i int) int { return i + 1 },
		"dec":  func(i int) int { return i - 1 },
		"mul":  func(p1 int, p2 int) int { return p1 * p2 },
		"mul3": func(p1, p2, p3 int) int { return p1 * p2 * p3 },
		"boolValue": func(b bool) int {
			if b {
				return 1
			} else {
				return 0
			}
		},
		"toUpperCamelCase": strcase.UpperCamelCase,
		"toLowerCamelCase": strcase.LowerCamelCase,
		"toKebabCase":      strcase.KebabCase,
		"toLower":          strings.ToLower,
		"toUpper":          strings.ToUpper,
		"mod":              func(i, j int) bool { return i%j == 0 },
		"deref":            func(s *string) string { return *s },
		"derefInt":         func(i *int) int { return *i },
		"list2string": func(s []*string) string {
			var str string
			for i, v := range s {
				if i < len(s)-1 {
					str = str + fmt.Sprintf("\"%s\", ", *v)
				} else {
					str = str + fmt.Sprintf("\"%s\"", *v)
				}
			}
			return str
		},
		"rtCommExpr": func(vrfUpId, lmgs int, wlShortname string) string {
			// if we come here there should be at least 1 element
			rtCommExpr := fmt.Sprintf("[rt-lmg%d-%d-%s]", 1, vrfUpId+1, wlShortname)
			for i := 2; i <= lmgs; i++ {
				rtCommExpr += fmt.Sprintf(" OR [rt-lmg%d-%d-%s]", i, vrfUpId+i, wlShortname)
			}
			return rtCommExpr
		},
		"lastmap": func(s string, x map[string][]*string) bool {
			i := 0
			for k := range x {
				if k == s {
					if i == len(x)-1 {
						return true
					}
				}
				i++
			}
			return false
		},
		"interfaceCounterMap": func() map[int]int {
           return map[int]int{}
		},
		"interfaceCounterMapIncrement": func(key int, counter map[int]int) string{
			if _,exists := counter[key]; !exists {
        	    counter[key] = 0
			}
			counter[key] = counter[key] + 1
			return ""
		},
		"interfaceCounterMapGet": func(key int, counter map[int]int) int{
        	return counter[key]
		},
		"pad": func(mnc int) string{
			count := 0
			mnc_temp := mnc
			for mnc_temp != 0 {
				mnc_temp /= 10
				count = count + 1
			}
			if count == 2{
				return "0" + strconv.Itoa(mnc)
			} 
			return strconv.Itoa(mnc)
		},
	}
)

// makek8sTemplate generates a template.Template for a particular named source
// template; with a common set of helper functions.
func makek8sTemplate(name, src string) *template.Template {
	return template.Must(template.New(name).Funcs(templateHelperFunctions).Funcs(sprig.TxtFuncMap()).Parse(src))
}

// WriteKustomize function writes the kustomize resource file
func (p *Parser) WriteKustomize(dirName, fileName *string, resources []string) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		Resources []string
	}{
		Resources: resources,
	}

	if err := goTemplates["kustomize"].Execute(file, s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

// WriteSrlInterface function writes the k8s srl interface resource
func (p *Parser) WriteSrlInterface(dirName, fileName, resName, target *string, interfaces []*types.K8ssrlinterface) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName string
		Target       string
		Interfaces   []*types.K8ssrlinterface
	}{
		ResourceName: *resName,
		Target:       *target,
		Interfaces:   interfaces,
	}

	if err := goTemplates["srlInterface"].Execute(file, s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

// WriteSrlInterface function writes the k8s srl subinterface resource
func (p *Parser) WriteSrlSubInterface(dirName, fileName, resName, target *string, subinterfaces []*types.K8ssrlsubinterface) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName  string
		Target        string
		InterfaceName string
		SubInterfaces []*types.K8ssrlsubinterface
	}{
		ResourceName:  *resName,
		Target:        *target,
		InterfaceName: subinterfaces[0].InterfaceRealName, // if we come here there will be 1 element in the list so we pick the first, since interfacename will always be the same
		SubInterfaces: subinterfaces,
	}

	if err := goTemplates["srlSubInterface"].Execute(file, s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

// WriteSrlIrbSubInterface function writes the k8s srl subinterface resource
func (p *Parser) WriteSrlIrbSubInterface(dirName, fileName, resName, target *string, irbsubinterfaces []*types.K8ssrlirbsubinterface) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName  string
		Target        string
		InterfaceName string
		SubInterfaces []*types.K8ssrlirbsubinterface
	}{
		ResourceName:  *resName,
		Target:        *target,
		InterfaceName: irbsubinterfaces[0].InterfaceRealName, // if we come here there will be 1 element in the list so we pick the first, since interfacename will always be the same
		SubInterfaces: irbsubinterfaces,
	}

	if err := goTemplates["srlIrbSubInterface"].Execute(file, s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

// WriteSrlTunnelInterface function writes the k8s srl tunnel-interface resource
func (p *Parser) WriteSrlTunnelInterface(dirName, fileName, resName, target *string, tunnelinterfaces []*types.K8ssrlTunnelInterface) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName     string
		Target           string
		TunnelInterfaces []*types.K8ssrlTunnelInterface
	}{
		ResourceName:     *resName,
		Target:           *target,
		TunnelInterfaces: tunnelinterfaces,
	}

	if err := goTemplates["srlTunnelInterface"].Execute(file, s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

// WriteSrlVxlanInterface function writes the k8s srl vxlan interface within the tunnelinterface resource
func (p *Parser) WriteSrlVxlanInterface(dirName, fileName, resName, target *string, vxlaninterfaces []*types.K8ssrlVxlanInterface) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName        string
		Target              string
		TunnelInterfaceName string
		VxlanInterfaces     []*types.K8ssrlVxlanInterface
	}{
		ResourceName:        *resName,
		Target:              *target,
		TunnelInterfaceName: vxlaninterfaces[0].TunnelInterfaceName, // if we come here there will be 1 element in the list so we pick the first, since interfacename will always be the same
		VxlanInterfaces:     vxlaninterfaces,
	}

	if err := goTemplates["srlVxlanInterface"].Execute(file, s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

// WriteSrlNetworkInstance function writes the k8s srl network-instance resource
func (p *Parser) WriteSrlNetworkInstance(dirName, fileName, resName, target *string, netwinstance *types.K8ssrlNetworkInstance) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName    string
		Target          string
		NetworkInstance *types.K8ssrlNetworkInstance
	}{
		ResourceName:    *resName,
		Target:          *target,
		NetworkInstance: netwinstance,
	}

	if err := goTemplates["srlNetworkInstance"].Execute(file, s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

// WriteSrlNetworkInstance function writes the k8s srl protocols bgp resource
func (p *Parser) WriteSrlProtocolsBgp(dirName, fileName, resName, target *string, protocolsbgp *types.K8ssrlprotocolsbgp) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName string
		Target       string
		ProtocolBgp  *types.K8ssrlprotocolsbgp
	}{
		ResourceName: *resName,
		Target:       *target,
		ProtocolBgp:  protocolsbgp,
	}

	if err := goTemplates["srlProtocolsBgp"].Execute(file, s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

func (p *Parser) WriteSrlSystemNetworkInstance(dirName, fileName, resName, target *string, esis []*types.K8ssrlESI) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName string
		Target       string
		ESIs         []*types.K8ssrlESI
	}{
		ResourceName: *resName,
		Target:       *target,
		ESIs:         esis,
	}

	if err := goTemplates["srlSystemNetworkInstance"].Execute(file, s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

// WriteSrlNetworkInstanceBgpVpn function writes the k8s srl network-instance bgpvpn protocol resource
func (p *Parser) WriteSrlNetworkInstanceBgpVpn(dirName, fileName, resName, target *string, netwInstanceProtocol *types.K8ssrlNetworkInstance) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName            string
		Target                  string
		NetworkInstanceProtocol *types.K8ssrlNetworkInstance
	}{
		ResourceName:            *resName,
		Target:                  *target,
		NetworkInstanceProtocol: netwInstanceProtocol,
	}

	if err := goTemplates["srlNetworkInstanceBgpVpn"].Execute(file, s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

// WriteSrlNetworkInstanceBgpEvpn function writes the k8s srl network-instance bgpevpn protocol resource
func (p *Parser) WriteSrlNetworkInstanceBgpEvpn(dirName, fileName, resName, target *string, netwInstanceProtocol *types.K8ssrlNetworkInstance) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName            string
		Target                  string
		NetworkInstanceProtocol *types.K8ssrlNetworkInstance
	}{
		ResourceName:            *resName,
		Target:                  *target,
		NetworkInstanceProtocol: netwInstanceProtocol,
	}

	if err := goTemplates["srlNetworkInstanceBgpEvpn"].Execute(file, s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

// WriteSrlNetworkInstanceLinux function writes the k8s srl network-instance bgpevpn protocol resource
func (p *Parser) WriteSrlNetworkInstanceLinux(dirName, fileName, resName, target *string, netwInstanceProtocol *types.K8ssrlNetworkInstance) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName            string
		Target                  string
		NetworkInstanceProtocol *types.K8ssrlNetworkInstance
	}{
		ResourceName:            *resName,
		Target:                  *target,
		NetworkInstanceProtocol: netwInstanceProtocol,
	}

	if err := goTemplates["srlNetworkInstanceLinux"].Execute(file, s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

// WriteSrlNetworkInstanceLinux function writes the k8s srl network-instance bgpevpn protocol resource
func (p *Parser) WriteSrlRoutingPolicy(dirName, fileName, resName, target *string, routingPolicy *types.K8ssrlRoutingPolicy) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName  string
		Target        string
		RoutingPolicy *types.K8ssrlRoutingPolicy
	}{
		ResourceName:  *resName,
		Target:        *target,
		RoutingPolicy: routingPolicy,
	}

	if err := goTemplates["srlRoutingPolicy"].Execute(file, s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}
