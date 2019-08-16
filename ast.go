package main

import (
	"os"
	"strings"
)

// ast
//
type ast struct {
	nodes    []astNode
	commands map[string]struct{} // Simulate a Set
}

func (a *ast) add(n astNode) {
	a.nodes = append(a.nodes, n)
	// Command?
	//
	if c, ok := n.(*astCmd); ok {
		a.commands[c.name] = struct{}{}
	}
}

// newAST
//
func newAST() *ast {
	a := &ast{}
	a.commands = make(map[string]struct{})
	return a
}

// astNode
//
type astNode interface {
	Apply(r *runfile)
}

// astValueElement
//
type astValueElement interface {
	Apply(r *runfile) string
}

// astValue
//
type astValue struct {
	value []astValueElement
}

// Apply
//
func (a *astValue) Apply(r *runfile) string {
	b := &strings.Builder{}
	if a != nil {
		for _, v := range a.value {
			b.WriteString(v.Apply(r))
		}
	}
	return b.String()
}

// newAstValue
//
func newAstValue(value []astValueElement) *astValue {
	return &astValue{value: value}
}

// newAstValue1
//
func newAstValue1(value astValueElement) *astValue {
	return &astValue{value: []astValueElement{value}}
}

// astCmd
//
type astCmd struct {
	name   string
	config *astCmdConfig
	script []string
}

func (a *astCmd) Apply(r *runfile) {
	cmd := &runCmd{name: a.name, script: a.script}
	// attrs
	//
	cmd.attrs = make(map[string]string)
	for k, v := range r.attrs {
		cmd.attrs[k] = v
	}
	// env
	//
	cmd.env = make(map[string]string)
	for k, v := range r.env {
		cmd.env[k] = v
	}
	// Config
	//
	cmd.config = &runCmdConfig{}
	cmd.config.shell = a.config.shell
	// Config Desc
	//
	for _, desc := range a.config.desc {
		cmd.config.desc = append(cmd.config.desc, desc.Apply(r))
	}
	// Config Usages
	//
	for _, usage := range a.config.usages {
		cmd.config.usages = append(cmd.config.usages, usage.Apply(r))
	}
	// Config Opts
	//
	for _, opt := range a.config.opts {
		cmd.config.opts = append(cmd.config.opts, opt.Apply(r))
	}
	r.cmds[cmd.name] = cmd
}

// astCmdConfig
//
type astCmdConfig struct {
	shell  string
	desc   []*astValue
	usages []*astValue
	opts   []*astCmdOpt
}

type astCmdOpt struct {
	name  string
	short rune
	long  string
	value string
	desc  *astValue
}

func (a *astCmdOpt) Apply(r *runfile) *runCmdOpt {
	opt := &runCmdOpt{}
	opt.name = a.name
	opt.short = a.short
	opt.long = a.long
	opt.value = a.value
	opt.desc = a.desc.Apply(r)
	return opt
}

// astAttrAssignment
//
type astAttrAssignment struct {
	name  string
	value *astValue
}

func (a *astAttrAssignment) Apply(r *runfile) {
	r.attrs[a.name] = a.value.Apply(r)
}

// astEnvAssignment
//
type astEnvAssignment struct {
	name  string
	value *astValue
}

func (a *astEnvAssignment) Apply(r *runfile) {
	r.env[a.name] = a.value.Apply(r)
}

// astEnvQAssignment
//
type astEnvQAssignment struct {
	name  string
	value *astValue
}

func (a *astEnvQAssignment) Apply(r *runfile) {
	// Only assign if not already present+non-empty
	//
	if val, ok := r.env[a.name]; !ok || len(val) == 0 {
		if val, ok = os.LookupEnv(a.name); !ok || len(val) == 0 {
			r.env[a.name] = a.value.Apply(r)
		}
	}
}

// astValueRunes
//
type astValueRunes struct {
	runes string
}

// newAstValueRunes
//
func newAstValueRunes(runes string) *astValueRunes {
	return &astValueRunes{runes: runes}
}

func (a *astValueRunes) Apply(r *runfile) string {
	return a.runes
}

// astValueEsc
//
type astValueEsc struct {
	seq string
}

func (a *astValueEsc) Apply(r *runfile) string {
	return string([]rune(a.seq)[1]) // TODO A bit of a hack
}

// astValueVar
//
type astValueVar struct {
	varName string
}

func (a *astValueVar) Apply(r *runfile) string {
	if val, ok := r.env[a.varName]; ok {
		return val
	}
	if val, ok := os.LookupEnv(a.varName); ok {
		return val
	}
	if val, ok := r.attrs[a.varName]; ok {
		return val
	}
	return ""
}

// astValueShell
//
type astValueShell struct {
	cmd *astValue
}

func (a *astValueShell) Apply(r *runfile) string {
	cmd := a.cmd.Apply(r)
	b := &strings.Builder{}
	executeSubCommand(r.attrs[".SHELL"], cmd, r.env, b)
	result := b.String()

	// Trim trailing newlines
	//
	for result[len(result)-1] == '\n' {
		result = result[0 : len(result)-1]
	}
	return result
}
