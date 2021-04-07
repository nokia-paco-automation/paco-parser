package parser

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	log "github.com/sirupsen/logrus"
	"github.com/stoewer/go-strcase"
)

var (
	goK8sSrlinterfaceTemplate = `
apiVersion: srlinux.henderiw.be/v1alpha1
kind: K8sSrlNokiaInterfacesInterface
metadata:
  name: {{.ResourceName}}
  labels:
    target: {{.Target}}
spec:
  interface:
{{range $index, $element := .Interfaces}}
  - name: {{$element.Name}}
    admin-state: enable
    description: "paco-{{$element.Name}}"
{{if eq $element.Kind "isl" "access"}}
    vlan-tagging: {{$element.VlanTagging}}
{{end}}
{{if ne $element.PortSpeed ""}}
    ethernet:
      port-speed: {{$element.PortSpeed}}
{{if eq $element.LagMember true }}
      aggregate-id: {{$element.LagName}}
{{end}}
{{end}}
{{if eq $element.Lag true}}
    lag:
      lag-type: lacp
      member-speed: {{$element.PortSpeed}}
{{if eq $element.Pxe true}}
      lacp-fallback-mode: static
{{end}}
      lacp:
        interval: FAST
        lacp-mode: ACTIVE
        admin-key: {{$element.AdminKey}}
        system-id-mac: {{$element.SystemMac}}
{{end}}
{{end}}
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
  {{range $index, $element := .TunnelInterfaces}}
    name: {{$element.Name}}
  {{end}}
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
{{range $index, $element := .VxlanInterfaces}}
  - index: {{$element.VlanID}}
    type: {{$element.Kind}}
    ingress:
      vni: {{$element.VlanID}}
    egress:
      source-ip: use-system-ipv4-address
{{end}}
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
  {{ $target := .Target}}
  subinterface:
  {{range $index, $element := .SubInterfaces}}
  - index: {{$element.VlanID}}
  {{if ne $element.Kind "loopback"}}
    type: {{$element.Kind}}
  {{end}}
    admin-state: enable
    description: "paco-{{$element.InterfaceShortName}}-{{$element.VlanID}}-{{$target}}"
  {{if eq $element.Kind "bridged"}}
    vlan:
      encap:
      {{if eq $element.VlanID "0"}}
        untagged
      {{else}}
        single-tagged:
          vlan-id: {{$element.VlanID}}
      {{end}}
  {{else}}
    ipv4:
      address: 
      - ip-prefix: {{$element.IPv4Prefix}}
    ipv6:
      address: 
      - ip-prefix: {{$element.IPv6Prefix}}
  {{end}}
  {{end}}
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
  {{ $target := .Target}}
  subinterface:
{{range $index, $element := .SubInterfaces}}
  - index: {{$element.VlanID}}
    admin-state: enable
    description: "{{$element.Description}}"
{{if ne (len $element.IPv4Prefix) 0 }}
    ipv4:
      address: 
{{range $index, $ipv4prefix := $element.IPv4Prefix}}
      - ip-prefix: {{$ipv4prefix}}
{{if eq $element.AnycastGW true}}
        anycast-gw: true
{{end}}
{{end}}
      arp:
        learn-unsolicited: true
        host-route:
          populate: dynamic
        evpn:
          advertise: dynamic
{{end}}
{{if eq $element.AnycastGW true}}
    anycast-gw:
      virtual-router-id: {{$element.VrID}}
{{end}}
{{if ne (len $element.IPv6Prefix) 0 }}
    ipv6:
      address: 
{{range $index, $ipv6prefix := $element.IPv6Prefix}}
      - ip-prefix: {{$ipv6prefix}}
{{if eq $element.AnycastGW true}}
        anycast-gw: true
{{end}}
{{end}}
      arp:
        learn-unsolicited: true
        host-route:
          populate: dynamic
        evpn:
          advertise: dynamic
{{end}}
{{end}}
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
{{range $index, $element := .NetworkInstance.SubInterfaces}}
    - name: {{$element.InterfaceRealName}}.{{$element.VlanID}}
{{end}}
{{if .NetworkInstance.TunnelInterfaceName}}
    vxlan-interface:
    - name: {{.NetworkInstance.TunnelInterfaceName}}
{{end}}
{{if eq .NetworkInstance.Kind "mac-vrf"}}
    bridge-table:
      mac-duplication:
        admin-state: enable
        action: blackhole
{{end}}
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
      import-reject-all: true
      export-reject-all: true
    group: 
    {{range $index, $element := .ProtocolBgp.PeerGroups}}
    - group-name: {{$element.Name}}
      admin-state: enable
      next-hop-self: true
      {{range $index, $protocol := $element.Protocols}}
      {{$protocol}}:
        admin-state: enable
      {{end}}
    {{end}}
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
    {{range $index, $element := .ProtocolBgp.Neighbors}}
    - peer-address: {{$element.PeerIP}}
      peer-as: {{$element.PeerAS}}
      peer-group: {{$element.PeerGroup}}
      {{if ne $element.LocalAS 0}}
      local-as:
        as-number: {{$element.LocalAS}}
      {{end}}
      {{if ne $element.TransportAddress ""}}
      transport:
        local-address: {{$element.TransportAddress}}
      {{end}}
    {{end}}
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
          id: 1
      evpn:
        ethernet-segments:
          bgp-instance:
            id: 1
            ethernet-segment:
            {{range $index, $element := .ESIs}}
            - esi: {{$element.ESI}}
              admin-state: enable
              interface: {{$element.LagName}}
            {{end}}
`

	goTemplates = map[string]*template.Template{
		"srlInterface":             makek8sTemplate("srlInterface", goK8sSrlinterfaceTemplate),
		"srlSubInterface":          makek8sTemplate("srlSubInterface", goK8sSrlsubinterfaceTemplate),
		"srlIrbSubInterface":       makek8sTemplate("srlIrbSubInterface", goK8sSrlIrbSubInterfaceTemplate),
		"srlTunnelInterface":       makek8sTemplate("srlTunnelInterface", goK8sSrlTunnelInterfaceTemplate),
		"srlVxlanInterface":        makek8sTemplate("srlVxlanInterface", goK8sSrlVxlanInterfaceTemplate),
		"srlNetworkInstance":       makek8sTemplate("srlNetworkInstance", goK8sSrlnetworkinstanceTemplate),
		"srlProtocolsBgp":          makek8sTemplate("srlProtocolsBgp", goK8sSrlprotocolsbgpTemplate),
		"srlSystemNetworkInstance": makek8sTemplate("srlSystemNetworkInstance", goK8sSrlSystemNetworkInstanceTemplate),
	}

	// templateHelperFunctions specifies a set of functions that are supplied as
	// helpers to the templates that are used within this file.
	templateHelperFunctions = template.FuncMap{
		// inc provides a means to add 1 to a number, and is used within templates
		// to check whether the index of an element within a loop is the last one,
		// such that special handling can be provided for it (e.g., not following
		// it with a comma in a list of arguments).
		"inc": func(i int) int {
			return i + 1
		},
		"dec": func(i int) int {
			return i - 1
		},
		"toUpperCamelCase": strcase.UpperCamelCase,
		"toLowerCamelCase": strcase.LowerCamelCase,
		"toKebabCase":      strcase.KebabCase,
		"toLower":          strings.ToLower,
		"toUpper":          strings.ToUpper,
	}
)

// makek8sTemplate generates a template.Template for a particular named source
// template; with a common set of helper functions.
func makek8sTemplate(name, src string) *template.Template {
	return template.Must(template.New(name).Funcs(templateHelperFunctions).Funcs(sprig.TxtFuncMap()).Parse(src))
}

// WriteSrlInterface function writes the k8s srl interface resource
func (p *Parser) WriteSrlInterface(dirName, fileName, resName, target *string, interfaces []*k8ssrlinterface) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName string
		Target       string
		Interfaces   []*k8ssrlinterface
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
func (p *Parser) WriteSrlSubInterface(dirName, fileName, resName, target *string, subinterfaces []*k8ssrlsubinterface) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName  string
		Target        string
		InterfaceName string
		SubInterfaces []*k8ssrlsubinterface
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
func (p *Parser) WriteSrlIrbSubInterface(dirName, fileName, resName, target *string, irbsubinterfaces []*k8ssrlirbsubinterface) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName  string
		Target        string
		InterfaceName string
		SubInterfaces []*k8ssrlirbsubinterface
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
func (p *Parser) WriteSrlTunnelInterface(dirName, fileName, resName, target *string, tunnelinterfaces []*k8ssrlTunnelInterface) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName     string
		Target           string
		TunnelInterfaces []*k8ssrlTunnelInterface
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
func (p *Parser) WriteSrlVxlanInterface(dirName, fileName, resName, target *string, vxlaninterfaces []*k8ssrlVxlanInterface) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName        string
		Target              string
		TunnelInterfaceName string
		VxlanInterfaces     []*k8ssrlVxlanInterface
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
func (p *Parser) WriteSrlNetworkInstance(dirName, fileName, resName, target *string, netwinstance *k8ssrlnetworkinstance) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName    string
		Target          string
		NetworkInstance *k8ssrlnetworkinstance
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
func (p *Parser) WriteSrlProtocolsBgp(dirName, fileName, resName, target *string, protocolsbgp *k8ssrlprotocolsbgp) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName string
		Target       string
		ProtocolBgp  *k8ssrlprotocolsbgp
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

func (p *Parser) WriteSrlSystemNetworkInstance(dirName, fileName, resName, target *string, esis []*k8ssrlESI) error {
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(*fileName)))
	if err != nil {
		return err
	}

	s := struct {
		ResourceName string
		Target       string
		ESIs         []*k8ssrlESI
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
