package main

// runfile
//
type runfile struct {
	attrs   map[string]string // All keys uppercase. Keys include leading '.'
	vars    map[string]string // Runfile variables
	exports []string          // Exported variables
	cmds    []*runCmd
}

func (r *runfile) DefaultShell() (string, bool) {
	shell, ok := r.attrs[".SHELL"]
	return shell, ok && len(shell) > 0
}

// processAST
//
func processAST(ast *ast) *runfile {
	rf := &runfile{
		attrs:   map[string]string{},
		vars:    map[string]string{},
		exports: []string{},
		cmds:    []*runCmd{},
	}
	for _, node := range ast.nodes {
		node.Apply(rf)
	}
	return rf
}

// runCmdOpt
//
type runCmdOpt struct {
	name  string
	short rune
	long  string
	value string
	desc  string
}

// runCmdConfig
//
type runCmdConfig struct {
	shell  string
	desc   []string
	usages []string
	opts   []*runCmdOpt
}

// runCmd
//
type runCmd struct {
	name   string
	config *runCmdConfig
	attrs  map[string]string
	env    map[string]string
	script []string
}

func (c *runCmd) Title() string {
	if len(c.config.desc) > 0 {
		return c.config.desc[0]
	}
	return ""
}
func (c *runCmd) Shell() string {
	return defaultIfEmpty(c.config.shell, c.attrs[".SHELL"])
}
func (c *runCmd) EnableHelp() bool {
	return len(c.config.desc) > 0 || len(c.config.usages) > 0 || len(c.config.opts) > 0
}
