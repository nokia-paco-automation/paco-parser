package templating

import (
	"bytes"
	"path"
	"text/template"

	"github.com/nokia-paco-automation/paco-parser/types"

	log "github.com/sirupsen/logrus"
)

func ProcessSwitchTemplates(wr *types.WorkloadResults, ir *types.InfrastructureResult, cg *types.ClientGroupResults) {
	log.Infof("ProcessingSwitchTemplates")
	for nodename, nodeInterfaces := range ir.IslInterfaces {
		processInterfaces(nodeInterfaces)
		_ = nodename
	}

}

func processInterfaces(islinterfaces []*types.K8ssrlinterface) {
	buf := new(bytes.Buffer)
	t := template.Must(template.New("interfaces.tmpl").ParseFiles(path.Join("templates", "switch", "interfaces.tmpl")))
	err := t.ExecuteTemplate(buf, "interfaces.tmpl", islinterfaces)
	if err != nil {
		log.Infof("%+v", err)
	}

	log.Info(buf.String())
}
