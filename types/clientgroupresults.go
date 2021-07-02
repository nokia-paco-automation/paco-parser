package types

import (
	log "github.com/sirupsen/logrus"
)

type ClientGroupResults struct {
	ClientInterfaces map[string]map[string][]*K8ssrlinterface // node, clientgroup -> interface
	Esis             map[string][]*K8ssrlESI
	Counter          map[string]int // counter is just for temporary inspection of what is going on in the code
}

func NewClientGroupResults() *ClientGroupResults {
	return &ClientGroupResults{
		ClientInterfaces: map[string]map[string][]*K8ssrlinterface{},
		Esis:             map[string][]*K8ssrlESI{},
		Counter:          map[string]int{},
	}
}

func (c *ClientGroupResults) AppendEsis(cgName string, esis []*K8ssrlESI) {
	log.Debugf("Appending %d ESIs to Clientgroup %s", len(esis), cgName)
	c.Counter["AppendEsis"]++
	c.Esis[cgName] = append(c.Esis[cgName], esis...)
}

func (c *ClientGroupResults) AppendClientInterfaces(nodeName string, cgName string, clientInterfaces []*K8ssrlinterface) {
	log.Debugf("Appending %d interface to Clientgroup %s for node %s", len(clientInterfaces), cgName, nodeName)
	c.Counter["AppendClientInterfaces"]++
	for _, ci := range clientInterfaces {
		if _, ok := c.ClientInterfaces[nodeName][ci.Name]; !ok {
			c.ClientInterfaces[nodeName] = map[string][]*K8ssrlinterface{}
		}
		c.ClientInterfaces[nodeName][ci.Name] = append(c.ClientInterfaces[nodeName][ci.Name], clientInterfaces...)
	}
}
