package main

import (
	"log"
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

// astCmdNode
//
type astCmdNode interface {
	Apply(r *runfile, c *runCmd)
}

// astValueElement
//
type astValueElement interface {
	Apply(r *runfile) string
}

// astCmdValueElement
//
type astCmdValueElement interface {
	Apply(r *runfile, c *runCmd) string
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

// astExport
//
type astExport struct {
	names []string
}

func (a *astExport) Apply(r *runfile) {
	r.exports = append(r.exports, a.names...)
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
	// Exports - copy vars into cmd env
	// Setup cmd env first so its accessible from other cmd configs
	// Copy global exports first
	//
	cmd.env = make(map[string]string)
	for _, name := range r.exports {
		if value, ok := r.vars[name]; ok {
			cmd.env[name] = value
		} else {
			panic("Unknown variable for export: " + name)
		}
	}
	// Copy cmd exports
	// NOTE: There could be dupes but should be safe
	//
	for _, exports := range a.config.exports {
		for _, name := range exports.names {
			if value, ok := r.vars[name]; ok {
				cmd.env[name] = value
			} else {
				panic("Unknown variable for export: " + name)
			}
		}
	}
	// Config Environment
	//
	for _, value := range a.config.env {
		value.Apply(r, cmd)
	}

	// attrs
	//
	cmd.attrs = make(map[string]string)
	for k, v := range r.attrs {
		cmd.attrs[k] = v
	}
	// Config
	//
	cmd.config = &runCmdConfig{}
	cmd.config.shell = a.config.shell
	// Config Desc
	//
	for _, desc := range a.config.desc {
		cmd.config.desc = append(cmd.config.desc, desc.Apply(r, cmd))
	}
	cmd.config.desc = normalizeCmdDesc(cmd.config.desc)
	// Config Usages
	//
	for _, usage := range a.config.usages {
		cmd.config.usages = append(cmd.config.usages, usage.Apply(r, cmd))
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
	shell   string
	desc    []*astCmdValue
	usages  []*astCmdValue
	opts    []*astCmdOpt
	exports []*astCmdExport
	env     []astCmdNode
}

// astCmdOpt
//
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

// astCmdExport
//
type astCmdExport struct {
	names []string
}

// astCmdValue
//
type astCmdValue struct {
	value []astCmdValueElement
}

// Apply
//
func (a *astCmdValue) Apply(r *runfile, c *runCmd) string {
	b := &strings.Builder{}
	if a != nil {
		for _, v := range a.value {
			b.WriteString(v.Apply(r, c))
		}
	}
	return b.String()
}

// newAstCmdValue
//
func newAstCmdValue(value []astCmdValueElement) *astCmdValue {
	return &astCmdValue{value: value}
}

// newAstCmdValue1
//
func newAstCmdValue1(value astCmdValueElement) *astCmdValue {
	return &astCmdValue{value: []astCmdValueElement{value}}
}

// astCmdAstValue wraps and astValue in an astCmdValue
//
type astCmdAstValue struct {
	value *astValue
}

// Apply
//
func (a *astCmdAstValue) Apply(r *runfile, c *runCmd) string {
	return a.value.Apply(r)
}

// newAstCmdAstValue1
//
func newAstCmdAstValue1(value astValueElement) *astCmdValue {
	return &astCmdValue{value: []astCmdValueElement{&astCmdAstValue{value: newAstValue1(value)}}}
}

// astCmdEnvAssignment
//
type astCmdEnvAssignment struct {
	name  string
	value *astValue
}

func (a *astCmdEnvAssignment) Apply(r *runfile, c *runCmd) {
	c.env[a.name] = a.value.Apply(r)
}

// astCmdEnvQAssignment
//
type astCmdEnvQAssignment struct {
	name  string
	value *astValue
}

func (a *astCmdEnvQAssignment) Apply(r *runfile, c *runCmd) {
	// Only assign if not already present+non-empty
	//
	if val, ok := r.vars[a.name]; !ok || len(val) == 0 {
		c.env[a.name] = a.value.Apply(r)
	}
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

// astVarAssignment
//
type astVarAssignment struct {
	name  string
	value *astValue
}

func (a *astVarAssignment) Apply(r *runfile) {
	r.vars[a.name] = a.value.Apply(r)
}

// astVarQAssignment
//
type astVarQAssignment struct {
	name  string
	value *astValue
}

func (a *astVarQAssignment) Apply(r *runfile) {
	// Only assign if not already present+non-empty
	//
	if val, ok := r.vars[a.name]; !ok || len(val) == 0 {
		if val, ok = os.LookupEnv(a.name); !ok || len(val) == 0 {
			r.vars[a.name] = a.value.Apply(r)
		}
	}
}

// astValueRunes
//
type astValueRunes struct {
	value string
}

// newAstValueRunes
//
func newAstValueRunes(value string) *astValueRunes {
	return &astValueRunes{value: value}
}

func (a *astValueRunes) Apply(r *runfile) string {
	return a.value
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
	name string
}

func (a *astValueVar) Apply(r *runfile) string {
	if val, ok := r.vars[a.name]; ok {
		return val
	}
	if val, ok := os.LookupEnv(a.name); ok {
		return val
	}
	if val, ok := r.attrs[a.name]; ok {
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
	env := make(map[string]string)
	for _, name := range r.exports {
		if value, ok := r.vars[name]; ok {
			env[name] = value
		} else {
			log.Println("Warning: exported variable not defined: ", name)
		}
	}
	capturedOutput := &strings.Builder{}
	executeSubCommand(r.attrs[".SHELL"], cmd, env, capturedOutput)
	result := capturedOutput.String()

	// Trim trailing newlines, per std command-substitution behavior
	//
	for result[len(result)-1] == '\n' {
		result = result[0 : len(result)-1]
	}
	return result
}
