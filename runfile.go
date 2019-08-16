package main

// runfile
//
type runfile struct {
	attrs map[string]string  // All keys uppercase. Keys include leading '.'
	env   map[string]string  // Shell variables
	cmds  map[string]*runCmd // key = cmd
}

func (r *runfile) HasCommand(c string) bool {
	_, ok := r.cmds[c]
	return ok
}
func (r *runfile) DefaultShell() (string, bool) {
	shell, ok := r.attrs[".SHELL"]
	return shell, ok && len(shell) > 0
}

// processAST
//
func processAST(ast *ast) *runfile {
	rf := &runfile{
		attrs: make(map[string]string),
		env:   make(map[string]string),
		cmds:  make(map[string]*runCmd),
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
	config *runCmdConfig
	attrs  map[string]string
	env    map[string]string
	name   string
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
func (c *runCmd) EnableUsage() bool {
	return len(c.config.usages) > 0 || len(c.config.opts) > 0
}
