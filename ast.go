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
func (a *ast) hasCommand(c string) bool {
	_, ok := a.commands[c]
	return ok
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
	Resolve(r *runfile)
}

// astCmd
//
type astCmd struct {
	name   string
	shell  string
	script []string
}

func (a *astCmd) Resolve(r *runfile) {
	cmd := &runCmd{name: a.name, shell: a.shell, script: a.script}
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
	r.cmds[cmd.name] = cmd
}

// astAttrAssignment
//
type astAttrAssignment struct {
	name  string
	value []astValueElement
}

func (a *astAttrAssignment) Resolve(r *runfile) {
	b := &strings.Builder{}
	for _, v := range a.value {
		b.WriteString(v.Resolve(r))
	}
	r.attrs[a.name] = b.String()
}

// astEnvAssignment
//
type astEnvAssignment struct {
	name  string
	value []astValueElement
}

func (a *astEnvAssignment) Resolve(r *runfile) {
	b := &strings.Builder{}
	for _, v := range a.value {
		b.WriteString(v.Resolve(r))
	}
	r.env[a.name] = b.String()
}

// astValueElement
//
type astValueElement interface {
	Resolve(r *runfile) string
}

// astValueRunes
//
type astValueRunes struct {
	runes string
}

func (a *astValueRunes) Resolve(r *runfile) string {
	return a.runes
}

// astValueEsc
//
type astValueEsc struct {
	seq string
}

func (a *astValueEsc) Resolve(r *runfile) string {
	return string([]rune(a.seq)[1]) // TODO A bit of a hack
}

// astValueVar
//
type astValueVar struct {
	varName string
}

func (a *astValueVar) Resolve(r *runfile) string {
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
	cmd []astValueElement
}

func (a *astValueShell) Resolve(r *runfile) string {
	b := &strings.Builder{}
	for _, v := range a.cmd {
		b.WriteString(v.Resolve(r))
	}
	cmd := b.String()

	b.Reset()

	executeSubCommand(r.attrs[".SHELL"], cmd, r.env, b)

	result := b.String()

	// Trim trailing newlines
	//
	for result[len(result)-1] == '\n' {
		result = result[0 : len(result)-1]
	}
	return result
}
