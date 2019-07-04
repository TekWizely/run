package main

type ast struct {
	commands map[string]*script
}

func (a *ast) Commands() []string {
	keys := make([]string, len(a.commands))
	i := 0
	for k := range a.commands {
		keys[i] = k
		i++
	}
	return keys
}

func (a *ast) Command(command string) *script {
	return a.commands[command]
}

type script struct {
	_    string
	text []string
}

func newAST() *ast {
	a := &ast{}
	a.commands = make(map[string]*script)
	return a
}
