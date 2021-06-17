package parser

import (
	"os"
	"path/filepath"
	"text/template"

	log "github.com/sirupsen/logrus"
)

// RenderStatefulSet function writes the StatefulSet file
func (p *Parser) RenderStatefulSet(t *template.Template, dirName *string, values *Values) error {
	log.Infof("Render statefulset file...")
	fileName := "statefulSet.yaml"
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(fileName)))
	if err != nil {
		return err
	}

	s := struct {
		Values *Values
	}{
		Values: values,
	}

	if err := t.ExecuteTemplate(file, "StatefulSet.tmpl", s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

// RenderToActiveConfigMap function writes the ActiveConfigMap file
func (p *Parser) RenderToActiveConfigMap(t *template.Template, dirName *string, values *Values) error {
	log.Infof("Render network attachement file...")
	fileName := "toActiveConfigMap.yaml"
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(fileName)))
	if err != nil {
		return err
	}

	s := struct {
		Values *Values
	}{
		Values: values,
	}

	if err := t.ExecuteTemplate(file, "ToActive_ConfigMap.tmpl", s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}

// RenderNetworkAttachement function writes the network Attachement file
func (p *Parser) RenderNetworkAttachement(t *template.Template, dirName *string, values *Values) error {
	log.Infof("Render network attachement file...")
	fileName := "networkAttachementDefinition.yaml"
	file, err := os.Create(filepath.Join(*dirName, filepath.Base(fileName)))
	if err != nil {
		return err
	}

	s := struct {
		Values *Values
	}{
		Values: values,
	}

	if err := t.ExecuteTemplate(file, "networkAttachmentDefinition.tmpl", s); err != nil {
		log.Error(err)
	}
	file.Close()
	return nil
}
