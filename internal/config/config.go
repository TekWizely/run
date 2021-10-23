package config

import (
	"errors"
	"io"
	"log"
	"reflect"
	"runtime"
)

// Command is an abstraction for a command, allowing us to mix runfile commands and custom comments (help, list, etc).
//
type Command struct {
	Name   string
	Title  string
	Help   func()
	Run    func() int
	Rename func(string) // Rename Command to script Name in 'main' mode
}

// DefaultShell specifies which shell to use for command scripts and sub-shells if none explicitly defined.
//
const DefaultShell = "sh"

// Me stores the script name we consider the runfile to be running as.
//
var Me string

// ShebangMode treats the Runfile as the executable
//
var ShebangMode bool

// MainMode extends ShebangMode by auto-invoking the main command
//
var MainMode bool

// ErrOut is where logs and errors are sent to (generally stderr).
//
var ErrOut io.Writer

// ErrShell is an Error message for missing '.SHELL' attribute
//
var ErrShell = errors.New(".SHELL not defined")

// CommandList stores a list of commands.
//
var CommandList []*Command

// CommandMap stores a map of commands, keyed by the command name (lower-cased)
//
var CommandMap = make(map[string]*Command)

// RunBin holds the absolute path to the run command in use.
//
var RunBin string

// Runfile holds the (possibly relative) path to the current Runfile.
//
var Runfile string

// RunfileAbs holds the absolute path to the current Runfile.
//
var RunfileAbs string

// RunfileIsLoaded is true if the runfile has been successfully loaded
//
var RunfileIsLoaded bool

// RunfileIsDefault is true if the current Runfile is the default "Runfile"
//
var RunfileIsDefault bool

// EnableFnTrace shows parser/lexer fn call/stack
//
var EnableFnTrace = false

// ShowScriptTmpDir shows the directory where Command/sub-shell scripts are stored
//
var ShowScriptTmpDir = false

// ShowCmdShells shows the command shell in the command's help screen
//
var ShowCmdShells = false

// EnableRunfileOverride indicates if $RUNFILE env var or '-r | --runfile' arguments are supported in the current mode.
//
var EnableRunfileOverride = true

// TraceFn logs lexer transitions
//
func TraceFn(msg string, i interface{}) {
	//goland:noinspection GoBoolExpressions
	if EnableFnTrace {
		fnName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
		log.Println(msg, ":", fnName)
	}
}
