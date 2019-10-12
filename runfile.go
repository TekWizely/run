package main

// runfile
//
type runfile struct {
	scope *scope
	cmds  []*runCmd
}

func (r *runfile) DefaultShell() (string, bool) {
	shell, ok := r.scope.attrs[".SHELL"]
	return shell, ok && len(shell) > 0
}

// processAST
//
func processAST(ast *ast) *runfile {
	rf := &runfile{
		scope: newScope(),
		cmds:  []*runCmd{},
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
	scope  *scope
	script []string
}

func (c *runCmd) Title() string {
	if len(c.config.desc) > 0 {
		return c.config.desc[0]
	}
	return ""
}
func (c *runCmd) Shell() string {
	return defaultIfEmpty(c.config.shell, c.scope.attrs[".SHELL"])
}
func (c *runCmd) EnableHelp() bool {
	return len(c.config.desc) > 0 || len(c.config.usages) > 0 || len(c.config.opts) > 0
}
