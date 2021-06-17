package parser

import (
	"os"
	"path/filepath"
	"text/template"

	log "github.com/sirupsen/logrus"
)

// RenderSriovConfigMap function writes the sriov config map to be used for the cluster
func (p *Parser) RenderSriovConfigMap(t *template.Template, dirName *string, sriovc map[string]map[string]map[int]map[string][]*string) error {
	log.Info("Writing server k8s .yaml files...")
	fileName := "sriovdp_cm.yaml"
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(fileName)))
	if err != nil {
		return err
	}

	s := struct {
		Sriovc map[string]map[string]map[int]map[string][]*string
	}{
		Sriovc: sriovc,
	}

	if err := t.ExecuteTemplate(file, "sriov.tmpl", s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}
