package boundedlifetime

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin(linterName, New)
}

type plugin struct{}

func New(any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{newAnalyzer()}, nil
}

func (p *plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
