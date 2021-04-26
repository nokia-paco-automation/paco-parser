package parser

import (
	"os"
	"path/filepath"
	"text/template"

	log "github.com/sirupsen/logrus"
)

// WriteUpfValues function writes the amf values file
func (p *Parser) WriteCnfValues(t *template.Template, dirName, cnfName *string, appc *AppConfig, appIPMap *AppIPMap) error {
	log.Infof("Writing %s values.yaml...", *cnfName)
	fileName := *cnfName + "_values.yaml"
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(fileName)))
	if err != nil {
		return err
	}

	s := struct {
		AppIPMap *AppIPMap
		Appc     *AppConfig
	}{
		AppIPMap: appIPMap,
		Appc:     appc,
	}

	for wlName, workloads := range appc.Networks {
		for switchIndex, switchWorkloads := range workloads {
			for group, groups := range switchWorkloads{
				for nwtype, nwtypes := range groups {
					for nwsubType, netwsubTypes := range nwtypes {
						for idx, networkInfo := range netwsubTypes {
							log.Infof("%d",idx)
							log.Infof("%s %d %d %s %s %d", wlName, switchIndex, group, nwtype, nwsubType, len(networkInfo.Ipv4Addresses))
							for _, allocatedIPinfo := range networkInfo.Ipv4Addresses {
								log.Infof("ipAddress: %s", *allocatedIPinfo.IPAddress)
							}
						}	
					}
				}
			}
		}
	}

	if err := t.ExecuteTemplate(file, *cnfName+".tmpl", s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}
