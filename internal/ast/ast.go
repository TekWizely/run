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
	// Seed attributes
	//
	rf.Scope.PutAttr(".SHELL", config.DefaultShell)
	rf.Scope.PutAttr(".RUN", config.RunBin)
	rf.Scope.PutAttr(".RUNFILE", config.RunfileAbs)
	rf.Scope.PutAttr(".RUNFILE.DIR", config.RunfileAbsDir)
	rf.Scope.PutAttr(".SELF", config.CurrentRunfileAbs)
	rf.Scope.PutAttr(".SELF.DIR", config.CurrentRunfileAbsDir)
	for _, n := range ast.nodes {
		n.Apply(rf)
	}
	return rf
}

// ProcessAstRunfile process an AST into an existing Runfile
// Assumes Runfile created via ProcessAST
//
func ProcessAstRunfile(ast *Ast, rf *runfile.Runfile) {
	// Seed attributes
	// Save current values, restore before leaving
	//
	selfRunfileBak, _ := rf.Scope.GetAttr(".SELF")
	selfRunfileDirBak, _ := rf.Scope.GetAttr(".SELF.DIR")
	defer func() {
		rf.Scope.PutAttr(".SELF", selfRunfileBak)
		rf.Scope.PutAttr(".SELF.DIR", selfRunfileDirBak)
	}()
	rf.Scope.PutAttr(".SELF", config.CurrentRunfileAbs)
	rf.Scope.PutAttr(".SELF.DIR", config.CurrentRunfileAbsDir)
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

// ScopeVarExport exports a variable.
//
type ScopeVarExport struct {
	VarName string
}

// NewVarExport is a convenience method for exporting variables.
//
func NewVarExport(varName string) *ScopeVarExport {
	return &ScopeVarExport{VarName: varName}
}

// Apply applies the node to the scope.
//
func (a *ScopeVarExport) Apply(s *runfile.Scope) {
	s.ExportVar(a.VarName)
}

// ScopeAttrExport exports an attribute.
//
type ScopeAttrExport struct {
	AttrName string
	VarName  string
}

// NewAttrExport is a convenience method for exporting attributes.
//
func NewAttrExport(attrName string, varName string) *ScopeAttrExport {
	return &ScopeAttrExport{AttrName: attrName, VarName: varName}
}

// Apply applies the node to the scope.
//
func (a *ScopeAttrExport) Apply(s *runfile.Scope) {
	s.ExportAttr(a.AttrName, a.VarName)
}

// ScopeAssert asserts the test, exiting with message on failure.
//
type ScopeAssert struct {
	Runfile string
	Line    int
	Test    ScopeValueNode
	Message ScopeValueNode
}

// Apply applies the node to the scope.
//
func (a *ScopeAssert) Apply(s *runfile.Scope) {
	assert := &runfile.Assert{}
	assert.Runfile = a.Runfile
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
	// If pattern is not absolute, assume its relative to config.RunfileAbsDir
	//
	if !filepath.IsAbs(filePattern) {
		filePattern = filepath.Join(config.RunfileAbsDir, filePattern)
	}
	// Skip fileglob if pattern does not look like a glob.
	// By checking this ourselves, we hope to gain more control over error reporting,
	// as fileglob currently (as of v1.3.0) conceals the fs.ErrorNotExist condition.
	//
	if fileglob.ContainsMatchers(filePattern) {
		if files, err = fileglob.Glob(filePattern, fileglob.MaybeRootFS); err != nil {
			panic(fmt.Errorf("processing include pattern '%s': %s", filePattern, err))
			// OK for fileglob to result in 0 files, but notify user
			//
		} else if config.ShowNotices && len(files) == 0 {
			log.Printf("NOTICE: include pattern resulted in no matches: %s", filePattern)
		}
	} else {
		// Specific (not-glob) filename expected to exist - Checked in loop below
		//
		files = []string{filePattern}
	}
	// Save log prefix and current runfile values, restore before leaving
	//
	logPrefixBak := log.Prefix()
	currentRunfileBak := config.CurrentRunfile
	currentRunfileAbsBak := config.CurrentRunfileAbs
	currentRunfileAbsDirBak := config.CurrentRunfileAbsDir
	defer func() {
		log.SetPrefix(logPrefixBak)
		config.CurrentRunfile = currentRunfileBak
		config.CurrentRunfileAbs = currentRunfileAbsBak
		config.CurrentRunfileAbsDir = currentRunfileAbsDirBak
	}()
	// NOTE: filenames assumed to be absolute
	// TODO Sort list (path aware) ?
	//
	for _, filename := range files {
		// Have we included this file already?
		//
		if _, included := config.IncludedFiles[filename]; included {
			// Treat as a notice since we safely avoided the (possibly) infinite loop
			//
			if config.ShowNotices {
				log.Printf("NOTICE: runfile already included: '%s'", filename)
			}
		} else {
			fileBytes, exists, err := util.ReadFileIfExists(filename)
			if exists {
				// Mark file included
				//
				config.IncludedFiles[filename] = struct{}{}
				// Set new prefix so parse errors/line numbers will be relative to the correct file
				// For brevity, use path relative to config.RunfileAbsDir if possible
				//
				var filenameRel string
				if filenameRel, err = filepath.Rel(config.RunfileAbsDir, filename); err == nil && len(filenameRel) > 0 && !strings.HasPrefix(filenameRel, ".") {
					log.SetPrefix(filenameRel + ": ")
					config.CurrentRunfile = filenameRel
				} else {
					log.SetPrefix(filename + ": ")
					config.CurrentRunfile = filename
				}
				config.CurrentRunfileAbs = filename
				config.CurrentRunfileAbsDir = filepath.Dir(config.CurrentRunfileAbs)
				// Parse the file
				//
				rfAst := ParseBytes(fileBytes)
				// Process the AST
				//
				ProcessAstRunfile(rfAst, r)
				// Restore values here too for consistency
				//
				log.SetPrefix(logPrefixBak)
				config.CurrentRunfile = currentRunfileBak
				config.CurrentRunfileAbs = currentRunfileAbsBak
				config.CurrentRunfileAbsDir = currentRunfileAbsDirBak
			} else {
				//
				// We're about to panic, assume prior defer will restore values before exiting
				//

				if err == nil {
					panic(fmt.Errorf("include runfile not found: '%s'", filename))
				} else {
					// If path error, just show the wrapped error
					//
					if pathErr, ok := err.(*os.PathError); ok {
						err = pathErr.Unwrap()
					}
					panic(fmt.Errorf("include runfile '%s': %s", filename, err.Error()))
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
	Name    string
	Config  *CmdConfig
	Script  []string
	Runfile string
	Line    int
}

// Apply applies the node to the runfile.
//
func (a *Cmd) Apply(r *runfile.Runfile) {
	cmd := &runfile.RunCmd{
		Name:    a.Name,
		Scope:   runfile.NewScope(),
		Script:  a.Script,
		Runfile: a.Runfile,
		Line:    a.Line,
	}
	// Exports
	//
	for _, export := range r.Scope.GetVarExports() {
		cmd.Scope.ExportVar(export.VarName)
	}
	for _, export := range r.Scope.GetAttrExports() {
		cmd.Scope.ExportAttr(export.AttrName, export.VarName)
	}
	for _, export := range a.Config.VarExports {
		cmd.Scope.ExportVar(export.VarName)
	}
	for _, export := range a.Config.AttrExports {
		cmd.Scope.ExportAttr(export.AttrName, export.VarName)
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
	Shell       string
	Desc        []ScopeValueNode
	Usages      []ScopeValueNode
	Opts        []*CmdOpt
	Vars        []scopeNode
	VarExports  []*ScopeVarExport
	AttrExports []*ScopeAttrExport
	Asserts     []*CmdAssert
}

// CmdOpt wraps a command option.
//
type CmdOpt struct {
	Name     string
	Required bool
	Default  ScopeValueNode
	Short    rune
	Long     string
	Example  string
	Desc     ScopeValueNode
}

// Apply applies the node to the command.
//
func (a *CmdOpt) Apply(c *runfile.RunCmd) *runfile.RunCmdOpt {
	opt := &runfile.RunCmdOpt{}
	opt.Name = a.Name
	opt.Required = a.Required
	opt.HasDefault = a.Default != nil
	if opt.HasDefault {
		opt.Default = a.Default.Apply(c.Scope)
	}
	opt.Short = a.Short
	opt.Long = a.Long
	opt.Example = a.Example
	opt.Desc = a.Desc.Apply(c.Scope)
	return opt
}

// CmdAssert wraps a command assertion.
//
type CmdAssert struct {
	Runfile string
	Line    int
	Test    ScopeValueNode
	Message ScopeValueNode
}

// Apply applies the node to the Scope.
//
func (a *CmdAssert) Apply(s *runfile.Scope) *runfile.Assert {
	assert := &runfile.Assert{}
	assert.Runfile = a.Runfile
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
	for _, export := range s.GetVarExports() {
		if value, ok := s.GetVar(export.VarName); ok {
			env[export.VarName] = value
		} else {
			log.Printf("WARNING: exported variable not defined: '%s'", export.VarName)
		}
	}
	for _, export := range s.GetAttrExports() {
		if value, ok := s.GetAttr(export.AttrName); ok {
			env[export.VarName] = value
		} else {
			log.Printf("WARNING: exported attribute not defined: '%s'", export.AttrName)
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
