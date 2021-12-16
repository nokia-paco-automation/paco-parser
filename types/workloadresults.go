package types

import (
	log "github.com/sirupsen/logrus"
)

type WorkloadResults struct {
	VxlanInterfaces     map[string][]*K8ssrlVxlanInterface          //
	ClientSubInterfaces map[string]map[string][]*K8ssrlsubinterface // nodename, interface -> subinterfaces
	IrbSubInterfaces    map[string][]*K8ssrlirbsubinterface         // nodename -> subinterfaces
	NetworkInstances    map[string]map[int]*K8ssrlNetworkInstance   // nodename, networkInstanceID -> networkinstance
	Counter             map[string]int                              // counter is just for temporary inspection of what is going on in the code
}

func NewWorkloadResults() *WorkloadResults {
	return &WorkloadResults{
		VxlanInterfaces:     map[string][]*K8ssrlVxlanInterface{},
		ClientSubInterfaces: map[string]map[string][]*K8ssrlsubinterface{},
		IrbSubInterfaces:    map[string][]*K8ssrlirbsubinterface{},
		NetworkInstances:    map[string]map[int]*K8ssrlNetworkInstance{},
		Counter:             map[string]int{},
	}
}

func (w *WorkloadResults) AppendVxlanSubInterfaces(nodeName string, vxlanif []*K8ssrlVxlanInterface) {
	log.Debugf("Appending %d VxLAN interfaces to node %s", len(vxlanif), nodeName)
	w.Counter["AppendVxlanSubInterfaces"]++
	w.VxlanInterfaces[nodeName] = append(w.VxlanInterfaces[nodeName], vxlanif...)
}

func (w *WorkloadResults) AppendClientSubInterfaces(nodeName string, ifname string, clientSubInterface []*K8ssrlsubinterface) {
	log.Debugf("Appending %d client sub-interfaces to node %s", len(clientSubInterface), nodeName)
	w.Counter["AppendClientSubInterface"]++
	for _, si := range clientSubInterface {
		if _, ok := w.ClientSubInterfaces[nodeName]; !ok {
			w.ClientSubInterfaces[nodeName] = map[string][]*K8ssrlsubinterface{}
		}
		if _, ok := w.ClientSubInterfaces[nodeName][si.InterfaceShortName]; !ok {
			w.ClientSubInterfaces[nodeName][si.InterfaceShortName] = []*K8ssrlsubinterface{}
		}
		w.ClientSubInterfaces[nodeName][si.InterfaceShortName] = append(w.ClientSubInterfaces[nodeName][si.InterfaceShortName], clientSubInterface...)
	}
}

func (w *WorkloadResults) AppendIrbSubInterfaces(nodeName string, irbSubInterfaces []*K8ssrlirbsubinterface) {
	log.Debugf("Appending %d irb sub-interfaces to node %s", len(irbSubInterfaces), nodeName)
	w.Counter["AppendIrbSubInterface"]++
	w.IrbSubInterfaces[nodeName] = append(w.IrbSubInterfaces[nodeName], irbSubInterfaces...)
}

func (w *WorkloadResults) AppendNetworkInstance(nodeName string, id int, niInfo *K8ssrlNetworkInstance) {
	log.Debugf("Appending Network instance %s with id %d to node %s", niInfo.Name, id, nodeName)
	w.Counter["AppendNetworkInstance"]++
	if _, ok := w.NetworkInstances[nodeName]; !ok {
		w.NetworkInstances[nodeName] = map[int]*K8ssrlNetworkInstance{}
	}
	w.NetworkInstances[nodeName][id] = niInfo
}
