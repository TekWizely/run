package ast

import (
	"log"
	"strings"

	"github.com/tekwizely/run/internal/config"
	"github.com/tekwizely/run/internal/exec"
	"github.com/tekwizely/run/internal/runfile"
)

// ProcessAST processes an AST into a Runfile.
//
func ProcessAST(ast *Ast) *runfile.Runfile {
	rf := runfile.NewRunfile()
	for _, n := range ast.nodes {
		n.Apply(rf)
	}
	return rf
}

// Ast is the root ast container.
//
type Ast struct {
	nodes []node
}

// Add adds a root level node to the ast.
//
func (a *Ast) Add(n node) {
	a.nodes = append(a.nodes, n)
}

// AddScopeNode adds a Scope Node to the ast, wrapping it.
//
func (a *Ast) AddScopeNode(n scopeNode) {
	a.nodes = append(a.nodes, &nodeScopeNode{node: n})
}

// NewAST is a convenience method.
//
func NewAST() *Ast {
	a := &Ast{}
	return a
}

// node
//
type node interface {
	Apply(r *runfile.Runfile)
}

// scopeNode
//
type scopeNode interface {
	Apply(s *runfile.Scope)
}

// ScopeValueNode is a scope node that results in a string value.
//
type ScopeValueNode interface {
	Apply(s *runfile.Scope) string
}

// nodeScopeNode
//
type nodeScopeNode struct {
	node scopeNode
}

// Apply applies the node to the runfile.
//
func (a *nodeScopeNode) Apply(r *runfile.Runfile) {
	a.node.Apply(r.Scope)
}

// ScopeValueNodeList builds a single string from a list of value nodes.
//
type ScopeValueNodeList struct {
	Values []ScopeValueNode
}

// Apply applies the node to the scope.
//
func (a *ScopeValueNodeList) Apply(s *runfile.Scope) string {
	b := &strings.Builder{}
	if a != nil {
		for _, v := range a.Values {
			b.WriteString(v.Apply(s))
		}
	}
	return b.String()
}

// NewScopeValueNodeList is a convenience method.
//
func NewScopeValueNodeList(value []ScopeValueNode) *ScopeValueNodeList {
	return &ScopeValueNodeList{Values: value}
}

// NewScopeValueNodeList1 is a convenience method for wrapping a single value node.
//
func NewScopeValueNodeList1(value ScopeValueNode) *ScopeValueNodeList {
	return &ScopeValueNodeList{Values: []ScopeValueNode{value}}
}

// ScopeExportList contains a list of exported vars.
//
type ScopeExportList struct {
	Names []string
}

// NewScopeExportList1 is a convience method for wrapping a single export.
//
func NewScopeExportList1(name string) *ScopeExportList {
	return &ScopeExportList{[]string{name}}
}

// Apply applies the node to the scope.
//
func (a *ScopeExportList) Apply(s *runfile.Scope) {
	for _, name := range a.Names {
		s.AddExport(name)
	}
}

// Cmd wraps a parsed command.
//
type Cmd struct {
	Name   string
	Config *CmdConfig
	Script []string
}

// Apply applies the node to the runfile.
//
func (a *Cmd) Apply(r *runfile.Runfile) {
	cmd := &runfile.RunCmd{
		Name:   a.Name,
		Scope:  runfile.NewScope(),
		Script: a.Script,
	}
	// Exports
	//
	for _, name := range r.Scope.GetExports() {
		cmd.Scope.AddExport(name)
	}
	for _, nameList := range a.Config.Exports {
		for _, name := range nameList.Names {
			cmd.Scope.AddExport(name)
		}
	}
	// Vars
	// Start with copy of global vars
	//
	for key, value := range r.Scope.Vars {
		cmd.Scope.PutVar(key, value)
	}
	// Attrs
	//
	for k, v := range r.Scope.Attrs {
		cmd.Scope.PutAttr(k, v)
	}
	// Config Environment
	//
	for _, varAssignment := range a.Config.Vars {
		varAssignment.Apply(cmd.Scope)
	}
	// Config
	//
	cmd.Config = &runfile.RunCmdConfig{}
	// .SHELL
	//
	cmd.Config.Shell = a.Config.Shell
	cmd.Scope.PutAttr(".SHELL", cmd.Shell())
	// Config Desc
	//
	for _, desc := range a.Config.Desc {
		cmd.Config.Desc = append(cmd.Config.Desc, desc.Apply(cmd.Scope))
	}
	cmd.Config.Desc = runfile.NormalizeCmdDesc(cmd.Config.Desc)
	// Config Usages
	//
	for _, usage := range a.Config.Usages {
		cmd.Config.Usages = append(cmd.Config.Usages, usage.Apply(cmd.Scope))
	}
	// Config Opts
	//
	for _, opt := range a.Config.Opts {
		cmd.Config.Opts = append(cmd.Config.Opts, opt.Apply(cmd))
	}
	r.Cmds = append(r.Cmds, cmd)
}

// CmdConfig wraps a command config.
//
type CmdConfig struct {
	Shell   string
	Desc    []ScopeValueNode
	Usages  []ScopeValueNode
	Opts    []*CmdOpt
	Vars    []scopeNode
	Exports []*ScopeExportList
}

// CmdOpt wraps a command option.
//
type CmdOpt struct {
	Name  string
	Short rune
	Long  string
	Value string
	Desc  ScopeValueNode
}

// Apply applies the node to the command.
//
func (a *CmdOpt) Apply(c *runfile.RunCmd) *runfile.RunCmdOpt {
	opt := &runfile.RunCmdOpt{}
	opt.Name = a.Name
	opt.Short = a.Short
	opt.Long = a.Long
	opt.Value = a.Value
	opt.Desc = a.Desc.Apply(c.Scope)
	return opt
}

// ScopeAttrAssignment wraps an attribute assignment.
//
type ScopeAttrAssignment struct {
	Name  string
	Value ScopeValueNode
}

// Apply applies the node to the scope.
//
func (a *ScopeAttrAssignment) Apply(s *runfile.Scope) {
	s.PutAttr(a.Name, a.Value.Apply(s))
}

// ScopeVarAssignment wraps a variable assignment.
//
type ScopeVarAssignment struct {
	Name  string
	Value ScopeValueNode
}

// Apply applies the node to the scope.
//
func (a *ScopeVarAssignment) Apply(s *runfile.Scope) {
	s.PutVar(a.Name, a.Value.Apply(s))
}

// ScopeVarQAssignment wraps a variable Q-Assignment.
//
type ScopeVarQAssignment struct {
	Name  string
	Value ScopeValueNode
}

// Apply applies the node to the scope.
//
func (a *ScopeVarQAssignment) Apply(s *runfile.Scope) {
	// Only assign if not already present+non-empty
	//
	if val, ok := s.GetVar(a.Name); !ok || len(val) == 0 {
		// Use the Env value, if present+non-empty, else the assignment value
		//
		if val, ok = s.GetEnv(a.Name); !ok || len(val) == 0 {
			val = a.Value.Apply(s)
		}
		s.PutVar(a.Name, val)
	}
}

// ScopeValueRunes wraps a simple string as a value.
//
type ScopeValueRunes struct {
	Value string
}

// Apply applies the node to the scope, returning the value.
//
func (a *ScopeValueRunes) Apply(_ *runfile.Scope) string {
	return a.Value
}

// ScopeValueEsc wraps an escape sequence.
//
type ScopeValueEsc struct {
	Seq string
}

// Apply applies the node to the scope, returning the value.
//
func (a *ScopeValueEsc) Apply(_ *runfile.Scope) string {
	return string([]rune(a.Seq)[1]) // TODO A bit of a hack
}

// ScopeValueVar wraps a variable reference.
//
type ScopeValueVar struct {
	Name string
}

// Apply applies the node to the scope, returning the value.
//
func (a *ScopeValueVar) Apply(s *runfile.Scope) string {
	if val, ok := s.GetVar(a.Name); ok {
		return val
	}
	if val, ok := s.GetEnv(a.Name); ok {
		return val
	}
	if val, ok := s.GetAttr(a.Name); ok {
		return val
	}
	return ""
}

// ScopeValueShell wraps a command substitution string.
//
type ScopeValueShell struct {
	Cmd ScopeValueNode
}

// Apply applies the node to the scope, returning the value.
//
func (a *ScopeValueShell) Apply(s *runfile.Scope) string {
	cmd := a.Cmd.Apply(s)
	env := make(map[string]string)
	for _, name := range s.GetExports() {
		if value, ok := s.GetVar(name); ok {
			env[name] = value
		} else {
			log.Println("Warning: exported variable not defined: ", name)
		}
	}
	capturedOutput := &strings.Builder{}
	shell, ok := s.GetAttr(".SHELL")
	if !ok || len(shell) == 0 {
		shell = config.DefaultShell
	}
	exec.ExecuteSubCommand(shell, cmd, env, capturedOutput)
	result := capturedOutput.String()

	// Trim trailing newlines, per std command-substitution behavior
	//
	for result[len(result)-1] == '\n' {
		result = result[0 : len(result)-1]
	}
	return result
}
