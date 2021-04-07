package parser

import (
	log "github.com/sirupsen/logrus"
)

type Parser struct {
	BaseDir      *string
	ConfigFile   *ConfigFile
	Config       *Config
	Nodes        map[string]*Node
	Links        []*Link
	IPAM         map[string]*Ipam
	NextAS       *uint32
	Workloads    map[string]*Workload
	ClientGroups map[string]*ClientGroup
	//Dir        *parserDirectory

	debug bool
}

type ParserOption func(p *Parser)

// WithDebug initializes the debug flag
func WithDebug(d bool) ParserOption {
	return func(p *Parser) {
		p.debug = d
	}
}

// WithConfigFile initializes and marshals the config file
func WithConfigFile(file *string) ParserOption {
	return func(p *Parser) {
		if *file == "" {
			return
		}
		if err := p.GetConfig(file); err != nil {
			log.Fatalf("failed to read topology file: %v", err)
		}
	}
}

// WithOutput initializes the output variable
func WithOutput(o *string) ParserOption {
	return func(p *Parser) {
		p.BaseDir = StringPtr(*o + "/" + "kutomize")
	}
}

// NewParser function defines a new parser
func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{
		BaseDir:      new(string),
		Config:       new(Config),
		ConfigFile:   new(ConfigFile),
		Nodes:        make(map[string]*Node),
		Links:        make([]*Link, 0),
		IPAM:         make(map[string]*Ipam),
		Workloads:    make(map[string]*Workload),
		ClientGroups: make(map[string]*ClientGroup),
		NextAS:       new(uint32),
	}
	for _, o := range opts {
		o(p)
	}
	return p
}
