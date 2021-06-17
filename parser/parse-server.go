package parser

import (
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func (p *Parser) ParseServerData() {
	log.Infof("Rendering Server Data into k8s .yaml files...")
	// Create directory where the Server manifest files will be stored
	dirName := filepath.Join(*p.BaseServerDir)
	p.CreateDirectory(dirName, 0777)

	// Parse the application templates
	t := ParseTemplates("./templates/server")

	// identify the server nodes and its respective leaf/tor switches
	p.InitializeSwitchToServerInformation()

	// show the result of the parsing
	p.showSwitchToServerInformation()

	// render the config map for SRIOV
	p.RenderSriovConfigMap(t, &dirName, p.ClientServer2NetworkLinks)

}

func (p *Parser) showSwitchToServerInformation() {
	for linkName, linkInfo := range p.ClientServer2NetworkLinks["sriov"] {
		for numaID, numaInfo := range linkInfo {
			for switchName, switchInfo := range numaInfo {
				for _, pfName := range switchInfo {
					log.Infof("SRIOV linkName, NumaID, switchName, pfName: %s, %d, %s, %s", linkName, numaID, switchName, *pfName)
				}
			}
		}
	}
	for linkName, linkInfo := range p.ClientServer2NetworkLinks["sriov"] {
		for numaID := range linkInfo {
			log.Infof("IPVLAN linkName, NumaID: %s, %d", linkName, numaID)
		}
	}
}

func (p *Parser) InitializeSwitchToServerInformation() {
	for _, link := range p.Links {
		// if a link has sriov enabled it should be taken into account for further processing to check
		// numa and clientName handling
		if link.Sriov != nil && *link.Sriov {
			log.Infof("Sriov CLientName %s, Numa %d", *link.ClientName, *link.Numa)
			p.initializeClientServer2NetworkLinks(StringPtr("sriov"), link.ClientName, link.Numa)

			// check if link A is linux the other end B should be a switch
			if *link.A.Node.Kind == "linux" {

				p.updateClientServer2NetworkLinks(StringPtr("sriov"), link.ClientName, link.Numa, link.A.PeerNode.ShortName, link.A.RealName)

			}
			// check if link A is linux the other end B should be a switch
			if *link.B.Node.Kind == "linux" {

				p.updateClientServer2NetworkLinks(StringPtr("sriov"), link.ClientName, link.Numa, link.B.PeerNode.ShortName, link.B.RealName)
			}
		}
		if link.IPVlan != nil && *link.IPVlan {
			log.Infof("IPVLAN  CLientName %s, Numa %d", *link.ClientName, *link.Numa)
			p.initializeClientServer2NetworkLinks(StringPtr("ipvlan"), link.ClientName, link.Numa)
		}

	}
}

func (p *Parser) initializeClientServer2NetworkLinks(netwType, clientName *string, numa *int) {
	if _, ok := p.ClientServer2NetworkLinks[*netwType]; !ok {
		p.ClientServer2NetworkLinks[*netwType] = make(map[string]map[int]map[string][]*string)
		p.ClientServer2NetworkLinks[*netwType][*clientName] = make(map[int]map[string][]*string)
		p.ClientServer2NetworkLinks[*netwType][*clientName][*numa] = make(map[string][]*string)
	} else {
		if _, ok := p.ClientServer2NetworkLinks[*netwType][*clientName]; !ok {
			p.ClientServer2NetworkLinks[*netwType][*clientName] = make(map[int]map[string][]*string)
			p.ClientServer2NetworkLinks[*netwType][*clientName][*numa] = make(map[string][]*string)
		} else {
			if _, ok := p.ClientServer2NetworkLinks[*netwType][*clientName][*numa]; !ok {
				p.ClientServer2NetworkLinks[*netwType][*clientName][*numa] = make(map[string][]*string)
			}
		}
	}
}

func (p *Parser) updateClientServer2NetworkLinks(netwType, clientName *string, numa *int, switchName, serverItfceName *string) {
	if _, ok := p.ClientServer2NetworkLinks[*netwType][*clientName][*numa][*switchName]; !ok {
		p.ClientServer2NetworkLinks[*netwType][*clientName][*numa][*switchName] = make([]*string, 0)
	}
	// check if the pfName is already added to the list or not, if not add it
	found := false
	var pfName *string
	for _, pfName = range p.ClientServer2NetworkLinks[*netwType][*clientName][*numa][*switchName] {
		if *pfName == *serverItfceName {
			found = true
		}
	}

	if !found {
		p.ClientServer2NetworkLinks[*netwType][*clientName][*numa][*switchName] = append(p.ClientServer2NetworkLinks[*netwType][*clientName][*numa][*switchName], serverItfceName)
	}
}
