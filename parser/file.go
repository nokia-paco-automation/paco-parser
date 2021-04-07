package parser

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ConfigFile type is a struct which defines parameters of the config file
type ConfigFile struct {
	fullName *string // file name with extension
	name     *string // file name without extension
}

// GetConfig parses the configuration file into p.Config structure
// as well as populates the ConfigFile structure with the config-file related information
func (p *Parser) GetConfig(file *string) error {
	log.Infof("Getting config information from %s file...", *file)

	yamlFile, err := ioutil.ReadFile(*file)
	if err != nil {
		return err
	}
	log.Debug(fmt.Sprintf("Config file contents:\n%s\n", yamlFile))

	err = yaml.Unmarshal(yamlFile, p.Config)
	if err != nil {
		return err
	}

	s := strings.Split(*file, "/")
	f := s[len(s)-1]
	filename := strings.Split(f, ".")
	p.ConfigFile = &ConfigFile{
		fullName: file,
		name:     &filename[0],
	}
	p.NextAS = p.Config.Infrastructure.Protocols.AsPool[0] 
	*p.NextAS++
	return nil
}

// CreateDirectory creates a directory
func (p *Parser) CreateDirectory(path string, perm os.FileMode) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, perm)
	}
}
