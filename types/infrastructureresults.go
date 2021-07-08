package types

import (
	log "github.com/sirupsen/logrus"
)

type InfrastructureResult struct {
	IslInterfaces           map[string][]*K8ssrlinterface
	IslSubInterfaces        map[string]map[string][]*K8ssrlsubinterface
	SystemSubInterfaces     map[string][]*K8ssrlsubinterface
	DefaultNetworkInstances map[string]*K8ssrlNetworkInstance
	DefaultProtocolBGP      map[string]*K8ssrlprotocolsbgp
	RoutingPolicy           *K8ssrlRoutingPolicy
	SystemInterfaces        []*K8ssrlinterface
	Counter                 map[string]int
	TunnelInterfaces        []*K8ssrlTunnelInterface
}

func NewInfrastructureResult() *InfrastructureResult {
	return &InfrastructureResult{
		IslInterfaces:           map[string][]*K8ssrlinterface{},
		IslSubInterfaces:        map[string]map[string][]*K8ssrlsubinterface{},
		SystemSubInterfaces:     map[string][]*K8ssrlsubinterface{},
		DefaultNetworkInstances: map[string]*K8ssrlNetworkInstance{},
		DefaultProtocolBGP:      map[string]*K8ssrlprotocolsbgp{},
		RoutingPolicy:           nil,
		SystemInterfaces:        []*K8ssrlinterface{},
		Counter:                 map[string]int{}, // counter is just for temporary inspection of what is going on in the code
	}
}

func (i *InfrastructureResult) AppendIslInterfaces(nodeName string, islinterfaces []*K8ssrlinterface) {
	log.Debugf("Appending %d ISL Interfaces for %s", len(islinterfaces), nodeName)
	i.Counter["AppendIslInterfaces"]++
	i.IslInterfaces[nodeName] = append(i.IslInterfaces[nodeName], islinterfaces...)
}
func (i *InfrastructureResult) SetRoutingPolicy(routingPolicy *K8ssrlRoutingPolicy) {
	log.Debugf("Setting RoutingPolicy")
	i.Counter["SetRoutingPolicy"]++
	i.RoutingPolicy = routingPolicy
}
func (i *InfrastructureResult) AppendSystemInterface(systemInterfaces []*K8ssrlinterface) {
	log.Debugf("Appending %d system interfaces", len(systemInterfaces))
	i.Counter["AppendSystemInterface"]++
	i.SystemInterfaces = append(i.SystemInterfaces, systemInterfaces...)
}
func (i *InfrastructureResult) AppendTunnelInterfaces(tunnelInterfaces []*K8ssrlTunnelInterface) {
	log.Debugf("Appending %d tunnel interfaces", len(tunnelInterfaces))
	i.Counter["AppendTunnelInterfaces"]++
	i.TunnelInterfaces = append(i.TunnelInterfaces, tunnelInterfaces...)
}
func (i *InfrastructureResult) AppendIslSubInterfaces(nodeName string, islsubinterfaces []*K8ssrlsubinterface) {
	log.Debugf("Appending %d ISL sub-interfaces to %s", len(islsubinterfaces), nodeName)
	i.Counter["AppendIslSubInterfaces"]++
	for _, si := range islsubinterfaces {
		if _, ok := i.IslSubInterfaces[nodeName][si.InterfaceShortName]; !ok {
			i.IslSubInterfaces[nodeName] = map[string][]*K8ssrlsubinterface{}
		}
		i.IslSubInterfaces[nodeName][si.InterfaceRealName] = append(i.IslSubInterfaces[nodeName][si.InterfaceRealName], islsubinterfaces...)
	}
}
func (i *InfrastructureResult) AppendSystemSubInterfaces(nodeName string, systemsubinterfaces []*K8ssrlsubinterface) {
	log.Debugf("Appending %d system sub-interfaces to %s", len(systemsubinterfaces), nodeName)
	i.Counter["AppendSystemSubInterfaces"]++
	i.SystemSubInterfaces[nodeName] = append(i.SystemSubInterfaces[nodeName], systemsubinterfaces...)
}
func (i *InfrastructureResult) AppendDefaultNetworkInstance(nodeName string, defaultNetworkInstance *K8ssrlNetworkInstance) {
	log.Debugf("Appending default Network Instance to %s", nodeName)
	i.Counter["AppendDefaultNetworkInstance"]++
	i.DefaultNetworkInstances[nodeName] = defaultNetworkInstance
}
func (i *InfrastructureResult) SetDefaultProtocolBgp(nodeName string, defaultProtocolBgp *K8ssrlprotocolsbgp) {
	log.Debugf("Setting default protocol BGP to %s", nodeName)
	i.Counter["SetDefaultProtocolBgp"]++
	i.DefaultProtocolBGP[nodeName] = defaultProtocolBgp
}
