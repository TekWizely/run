package ast

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goreleaser/fileglob"
	"github.com/tekwizely/run/internal/config"
	"github.com/tekwizely/run/internal/exec"
	"github.com/tekwizely/run/internal/runfile"
	"github.com/tekwizely/run/internal/util"
)

// ParseBytes is a conceit as we need a way for AST to invoke
// parsers for include functionality.
//
var ParseBytes func(runfile []byte) *Ast = nil

// ProcessAST processes an AST into a Runfile.
//
func ProcessAST(ast *Ast) *runfile.Runfile {
	rf := runfile.NewRunfile()
	// Seed defaults
	//
	rf.Scope.PutAttr(".SHELL", config.DefaultShell)
	rf.Scope.PutAttr(".RUN", config.RunBin)
	rf.Scope.PutAttr(".RUNFILE", config.RunfileAbs)
	for _, n := range ast.nodes {
		n.Apply(rf)
	}
	return rf
}

// ProcessAstRunfile process an AST into an existing Runfile
// Assumes Runfile created via ProcessAST
//
func ProcessAstRunfile(ast *Ast, rf *runfile.Runfile) {
	for _, n := range ast.nodes {
		n.Apply(rf)
	}
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

// NewScopeExportList1 is a convenience method for wrapping a single export.
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

// ScopeAssert asserts the test, exiting with message on failure.
//
type ScopeAssert struct {
	Line    int
	Test    ScopeValueNode
	Message ScopeValueNode
}

// Apply applies the node to the scope.
//
func (a *ScopeAssert) Apply(s *runfile.Scope) {
	assert := &runfile.Assert{}
	assert.Line = a.Line
	assert.Test = a.Test.Apply(s)
	assert.Message = strings.TrimSpace(a.Message.Apply(s))
	s.AddAssert(assert)
}

// ScopeInclude includes other runfiles.
//
type ScopeInclude struct {
	FilePattern ScopeValueNode
}

// Apply applies the node to the scope.
//
func (a *ScopeInclude) Apply(r *runfile.Runfile) {
	filePattern := a.FilePattern.Apply(r.Scope)
	var (
		files []string
		err   error
	)
	// We want the absolute file paths for include tracking
	// If pattern is not absolute, assume its relative to dir(Runfile)
	//
	if !filepath.IsAbs(filePattern) {
		filePattern = filepath.Join(filepath.Dir(config.RunfileAbs), filePattern)
	}
	if files, err = fileglob.Glob(filePattern, fileglob.MaybeRootFS); err != nil {
		panic(err)
	} else if len(files) == 0 {
		panic(fmt.Errorf("Include pattern '%s' resulted in no matches", filePattern))
	}
	// NOTE: filenames assumed to be absolute
	// TODO Sort list (path aware) ?
	//
	for _, filename := range files {
		// Have we included this file already?
		//
		if _, included := config.IncludedFiles[filename]; included {
			log.Printf("WARNING: runfile already included: '%s'", filename)
		} else {
			fileBytes, exists, err := util.ReadFileIfExists(filename)
			if exists {
				// Mark file included
				//
				config.IncludedFiles[filename] = struct{}{}
				// Save prefix, restore before leaving
				//
				logPrefix := log.Prefix()
				defer func() { log.SetPrefix(logPrefix) }()
				// Set new prefix so parse errors/line numbers will be relative to the correct file
				// For brevity, use path relative to dir(Runfile) if possible
				//
				var filenameRel string
				if filenameRel, err = filepath.Rel(filepath.Dir(config.RunfileAbs), filename); err == nil && len(filenameRel) > 0 && !strings.HasPrefix(filenameRel, ".") {
					log.SetPrefix(filenameRel + ": ")
				} else {
					log.SetPrefix(filename + ": ")
				}
				// Parse the file
				//
				rfAst := ParseBytes(fileBytes)
				// Process the AST
				//
				ProcessAstRunfile(rfAst, r)
			} else {
				if err == nil {
					panic(fmt.Errorf("Include runfile '%s' not found", filename))
				} else {
					// If path error, just show the wrapped error
					//
					if pathErr, ok := err.(*os.PathError); ok {
						err = pathErr.Unwrap()
					}
					panic(fmt.Errorf("Include runfile '%s': %s", filename, err.Error()))
				}
			}
		}
	}
}

// ScopeBracketString wraps a bracketed string.
//
type ScopeBracketString struct {
	Value ScopeValueNode
}

// NewScopeBracketString is a convenience method.
//
func NewScopeBracketString(value ScopeValueNode) ScopeValueNode {
	return &ScopeBracketString{Value: value}
}

// Apply applies the node to the scope.
//
func (a *ScopeBracketString) Apply(s *runfile.Scope) string {
	return "[ " + a.Value.Apply(s) + " ]"
}

// ScopeDBracketString wraps a double-bracketed string.
//
type ScopeDBracketString struct {
	Value ScopeValueNode
}

// NewScopeDBracketString is a convenience method.
//
func NewScopeDBracketString(value ScopeValueNode) ScopeValueNode {
	return &ScopeDBracketString{Value: value}
}

// Apply applies the node to the scope.
//
func (a *ScopeDBracketString) Apply(s *runfile.Scope) string {
	return "[[ " + a.Value.Apply(s) + " ]]"
}

// ScopeParenString wraps a paren-string.
//
type ScopeParenString struct {
	Value ScopeValueNode
}

// NewScopeParenString is a convenience method.
//
func NewScopeParenString(value ScopeValueNode) ScopeValueNode {
	return &ScopeParenString{Value: value}
}

// Apply applies the node to the scope.
//
func (a *ScopeParenString) Apply(s *runfile.Scope) string {
	return "( " + a.Value.Apply(s) + " )"
}

// ScopeDParenString wraps a double-paren string.
//
type ScopeDParenString struct {
	Value ScopeValueNode
}

// NewScopeDParenString is a convenience method.
//
func NewScopeDParenString(value ScopeValueNode) ScopeValueNode {
	return &ScopeDParenString{Value: value}
}

// Apply applies the node to the scope.
//
func (a *ScopeDParenString) Apply(s *runfile.Scope) string {
	return "(( " + a.Value.Apply(s) + " ))"
}

// Cmd wraps a parsed command.
//
type Cmd struct {
	Name   string
	Config *CmdConfig
	Script []string
	Line   int
}

// Apply applies the node to the runfile.
//
func (a *Cmd) Apply(r *runfile.Runfile) {
	cmd := &runfile.RunCmd{
		Name:   a.Name,
		Scope:  runfile.NewScope(),
		Script: a.Script,
		Line:   a.Line,
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
	// Asserts - Global first, then Command
	//
	for _, assert := range r.Scope.Asserts {
		cmd.Scope.AddAssert(assert)
	}
	for _, assert := range a.Config.Asserts {
		cmd.Scope.AddAssert(assert.Apply(cmd.Scope))
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
	Asserts []*CmdAssert
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

// CmdAssert wraps a command assertion.
//
type CmdAssert struct {
	Line    int
	Test    ScopeValueNode
	Message ScopeValueNode
}

// Apply applies the node to the Scope.
//
func (a *CmdAssert) Apply(s *runfile.Scope) *runfile.Assert {
	assert := &runfile.Assert{}
	assert.Line = a.Line
	assert.Test = a.Test.Apply(s)
	assert.Message = strings.TrimSpace(a.Message.Apply(s))
	return assert
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
			log.Printf("WARNING: exported variable not defined: '%s'", name)
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
