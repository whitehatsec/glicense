package output

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/hpapaxen/glicense/config"
	"github.com/hpapaxen/glicense/license"
	"github.com/hpapaxen/glicense/module"
)

const unknown = "unknown"

type CSVOutput struct {
	Path    string
	Config  *config.Config
	modules map[*module.Module]interface{}
	sync.Mutex
}

func (c *CSVOutput) Start(m *module.Module) {

}

func (c *CSVOutput) Update(
	m *module.Module,
	t license.StatusType, msg string) {

}

func (c *CSVOutput) Finish(
	m *module.Module,
	l *license.License,
	err error) {
	c.Lock()
	defer c.Unlock()

	if c.modules == nil {
		c.modules = make(map[*module.Module]interface{})
	}

	c.modules[m] = l
	if err != nil {
		c.modules[m] = err
	}
}

func (c *CSVOutput) Close() error {
	f, err := os.OpenFile(c.Path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	// Sort the modules by name
	keys := make([]string, 0, len(c.modules))
	index := map[string]*module.Module{}
	for m := range c.modules {
		keys = append(keys, m.Path)
		index[m.Path] = m
	}
	sort.Strings(keys)

	_, err = f.WriteString("Dependency," +
		"Version," +
		"SPDX ID," +
		"License," +
		"Allowed," +
		"Indirect" +
		"\n")
	if err != nil {
		return err
	}

	for _, k := range keys {
		m := index[k]

		line := c.formatLine(m)

		_, err = f.WriteString(line)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *CSVOutput) formatLine(m *module.Module) string {
	raw := c.modules[m]

	spdx := unknown
	licenseVal := unknown
	allowed := unknown
	switch t := raw.(type) {
	case *license.License:
		spdx = t.SPDX
		licenseVal = t.String()
		allowed = c.setAllowed(t)
	case error:
		licenseVal = fmt.Sprintf("ERROR: %s", t)
		licenseVal = strings.ReplaceAll(licenseVal, "\n", "")
		allowed = "no"
	}

	return fmt.Sprintf("%s,%s,%s,%s,%s,%t\n",
		m.Path,
		m.Version,
		spdx,
		licenseVal,
		allowed,
		m.Indirect)
}

func (c *CSVOutput) setAllowed(l *license.License) string {
	if c.Config == nil {
		return "unknown"
	}
	switch c.Config.Allowed(l) {
	case config.StateAllowed:
		return "Yes"
	case config.StateDenied:
		return "False"
	default:
		return "unknown"
	}
}
