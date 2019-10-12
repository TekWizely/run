package main

import (
	"log"
	"os"
	"strings"
)

type scope struct {
	attrs   map[string]string // All keys uppercase. Keys include leading '.'
	vars    map[string]string // Variables
	exports []string          // Exported variables
}

func newScope() *scope {
	return &scope{
		attrs:   map[string]string{},
		vars:    map[string]string{},
		exports: []string{},
	}
}
func (s *scope) GetEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

func (s *scope) GetAttr(key string) (string, bool) {
	val, ok := s.attrs[key]
	return val, ok
}

func (s *scope) PutAttr(key, value string) {
	s.attrs[key] = value
}

func (s *scope) GetVar(key string) (string, bool) {
	val, ok := s.vars[key]
	return val, ok
}

func (s *scope) PutVar(key, value string) {
	s.vars[key] = value
}

func (s *scope) AddExport(key string) {
	s.exports = append(s.exports, key)
}

func (s *scope) GetExports() []string {
	return s.exports
}

func (s *scope) DefaultShell() (string, bool) {
	shell, ok := s.attrs[".SHELL"]
	return shell, ok && len(shell) > 0
}

// ast
//
type ast struct {
	nodes []astNode
}

func (a *ast) add(n astNode) {
	a.nodes = append(a.nodes, n)
}

func (a *ast) addScopeNode(n astScopeNode) {
	a.nodes = append(a.nodes, &astNodeScopeNode{node: n})
}

// newAST
//
func newAST() *ast {
	a := &ast{}
	return a
}

// astNode
//
type astNode interface {
	Apply(r *runfile)
}

// astScopeNode
//
type astScopeNode interface {
	Apply(s *scope)
}

// astScopeValueNode
//
type astScopeValueNode interface {
	Apply(s *scope) string
}

// astCmdNode
//
type astCmdNode interface {
	Apply(r *runfile, c *runCmd)
}

// astNodeScopeNode
//
type astNodeScopeNode struct {
	node astScopeNode
}

func (a *astNodeScopeNode) Apply(r *runfile) {
	a.node.Apply(r.scope)
}

// astCmdNodeScopeNode
//
type astCmdNodeScopeNode struct {
	node astScopeNode
}

func (a *astCmdNodeScopeNode) Apply(r *runfile, c *runCmd) {
	a.node.Apply(c.scope)
}

// astScopeValueNodeList
//
type astScopeValueNodeList struct {
	values []astScopeValueNode
}

func (a *astScopeValueNodeList) Apply(s *scope) string {
	b := &strings.Builder{}
	if a != nil {
		for _, v := range a.values {
			b.WriteString(v.Apply(s))
		}
	}
	return b.String()
}

// newAstScopeValueNodeList
//
func newAstScopeValueNodeList(value []astScopeValueNode) *astScopeValueNodeList {
	return &astScopeValueNodeList{values: value}
}

// newAstScopeValueNodeList1
//
func newAstScopeValueNodeList1(value astScopeValueNode) *astScopeValueNodeList {
	return &astScopeValueNodeList{values: []astScopeValueNode{value}}
}

// astScopeExportList
//
type astScopeExportList struct {
	names []string
}

// newAstScopeExportList1
//
func newAstScopeExportList1(name string) *astScopeExportList {
	return &astScopeExportList{[]string{name}}
}

func (a *astScopeExportList) Apply(s *scope) {
	for _, name := range a.names {
		s.AddExport(name)
	}
}

// astCmd
//
type astCmd struct {
	name   string
	config *astCmdConfig
	script []string
}

func (a *astCmd) Apply(r *runfile) {
	cmd := &runCmd{
		name:   a.name,
		scope:  newScope(),
		script: a.script,
	}
	// Exports
	//
	for _, name := range r.scope.exports {
		cmd.scope.AddExport(name)
	}
	for _, nameList := range a.config.exports {
		for _, name := range nameList.names {
			cmd.scope.AddExport(name)
		}
	}
	// Vars
	// Start with copy of global vars
	//
	for key, value := range r.scope.vars {
		cmd.scope.PutVar(key, value)
	}
	// Config Environment
	//
	for _, varAssignment := range a.config.vars {
		varAssignment.Apply(cmd.scope)
	}
	// Attrs
	//
	for k, v := range r.scope.attrs {
		cmd.scope.PutAttr(k, v)
	}
	// Config
	//
	cmd.config = &runCmdConfig{}
	cmd.config.shell = a.config.shell
	// Config Desc
	//
	for _, desc := range a.config.desc {
		cmd.config.desc = append(cmd.config.desc, desc.Apply(cmd.scope))
	}
	cmd.config.desc = normalizeCmdDesc(cmd.config.desc)
	// Config Usages
	//
	for _, usage := range a.config.usages {
		cmd.config.usages = append(cmd.config.usages, usage.Apply(cmd.scope))
	}
	// Config Opts
	//
	for _, opt := range a.config.opts {
		cmd.config.opts = append(cmd.config.opts, opt.Apply(cmd))
	}
	r.cmds = append(r.cmds, cmd)
}

// astCmdConfig
//
type astCmdConfig struct {
	shell   string
	desc    []astScopeValueNode
	usages  []astScopeValueNode
	opts    []*astCmdOpt
	vars    []astScopeNode
	exports []*astScopeExportList
}

// astCmdOpt
//
type astCmdOpt struct {
	name  string
	short rune
	long  string
	value string
	desc  astScopeValueNode
}

func (a *astCmdOpt) Apply(c *runCmd) *runCmdOpt {
	opt := &runCmdOpt{}
	opt.name = a.name
	opt.short = a.short
	opt.long = a.long
	opt.value = a.value
	opt.desc = a.desc.Apply(c.scope)
	return opt
}

// astScopeAttrAssignment
//
type astScopeAttrAssignment struct {
	name  string
	value astScopeValueNode
}

func (a *astScopeAttrAssignment) Apply(s *scope) {
	s.PutAttr(a.name, a.value.Apply(s))
}

// astScopeVarAssignment
//
type astScopeVarAssignment struct {
	name  string
	value astScopeValueNode
}

func (a *astScopeVarAssignment) Apply(s *scope) {
	s.PutVar(a.name, a.value.Apply(s))
}

// astScopeVarQAssignment
//
type astScopeVarQAssignment struct {
	name  string
	value astScopeValueNode
}

func (a *astScopeVarQAssignment) Apply(s *scope) {
	// Only assign if not already present+non-empty
	//
	if val, ok := s.GetVar(a.name); !ok || len(val) == 0 {
		if val, ok = s.GetEnv(a.name); !ok || len(val) == 0 {
			s.PutVar(a.name, a.value.Apply(s))
		}
	}
}

// astScopeValueRunes
//
type astScopeValueRunes struct {
	value string
}

func (a *astScopeValueRunes) Apply(_ *scope) string {
	return a.value
}

// astScopeValueEsc
//
type astScopeValueEsc struct {
	seq string
}

func (a *astScopeValueEsc) Apply(_ *scope) string {
	return string([]rune(a.seq)[1]) // TODO A bit of a hack
}

// astScopeValueVar
//
type astScopeValueVar struct {
	name string
}

func (a *astScopeValueVar) Apply(s *scope) string {
	if val, ok := s.GetVar(a.name); ok {
		return val
	}
	if val, ok := s.GetEnv(a.name); ok {
		return val
	}
	if val, ok := s.GetAttr(a.name); ok {
		return val
	}
	return ""
}

// astScopeValueShell
//
type astScopeValueShell struct {
	cmd astScopeValueNode
}

func (a *astScopeValueShell) Apply(s *scope) string {
	cmd := a.cmd.Apply(s)
	env := make(map[string]string)
	for _, name := range s.GetExports() {
		if value, ok := s.GetVar(name); ok {
			env[name] = value
		} else {
			log.Println("Warning: exported variable not defined: ", name)
		}
	}
	capturedOutput := &strings.Builder{}
	shell, _ := s.GetAttr(".SHELL")
	executeSubCommand(shell, cmd, env, capturedOutput)
	result := capturedOutput.String()

	// Trim trailing newlines, per std command-substitution behavior
	//
	for result[len(result)-1] == '\n' {
		result = result[0 : len(result)-1]
	}
	return result
}
