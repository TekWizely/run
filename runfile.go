package main

// runfile
//
type runfile struct {
	attrs map[string]string  // All keys uppercase. Keys include leading '.'
	env   map[string]string  // Shell variables
	cmds  map[string]*runCmd // key = cmd.name
}

// processAST
//
func processAST(ast *ast) *runfile {
	runfile := &runfile{
		attrs: make(map[string]string),
		env:   make(map[string]string),
		cmds:  make(map[string]*runCmd),
	}
	for _, node := range ast.nodes {
		node.Resolve(runfile)
	}
	return runfile
}

// runCmd
//
type runCmd struct {
	attrs  map[string]string
	env    map[string]string
	name   string
	shell  string
	script []string
}
