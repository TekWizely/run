package config

import (
	"errors"
	"io"
	"log"
	"reflect"
	"runtime"
)

// Command is an abastraction for a command, allowing us to mix runfile commands and custom comments (help, list, etc).
//
type Command struct {
	Name   string
	Title  string
	Help   func()
	Run    func()
	Rename func(string) // Rename Command to script Name in 'main' mode
}

// Me stores the script name we consider the runfile to be running as.
//
var Me string

// ErrOut is where logs and errors are sent to (generally stderr).
//
var ErrOut io.Writer

// ErrShell is an Error message for missing .SHELL attribute
//
var ErrShell = errors.New(".SHELL not defined")

// CommandList stores a list of commands.
//
var CommandList []*Command

// CommandMap stores a map of commands, keyed by the command name (lowercased)
//
var CommandMap = make(map[string]*Command)

// EnableFnTrace shows parser/lexer fn call/stack
//
var EnableFnTrace = false

// ShowScriptFiles shows Command/sub-shell filenames
//
var ShowScriptFiles = false

// TraceFn logs lexer transitions
//
func TraceFn(msg string, i interface{}) {
	if EnableFnTrace {
		fnName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
		log.Println(msg, ":", fnName)
	}
}
