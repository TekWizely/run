package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"runtime"
	"strings"
)

type command struct {
	name   string
	title  string
	help   func()
	run    func()
	rename func(string) // Rename command to script name in 'main' mode
}

const (
	runfileDefault = "Runfile"
)

var (
	me          string
	inputFile   string
	shebangMode bool
	mainMode    bool
	errOut      io.Writer
	errShell    = errors.New(".SHELL not defined")
	commandList []*command
	commandMap  = make(map[string]*command)
)
var (
	enableFnTrace   = false // Show parser/lexer fn call/stack
	hidePanic       = false // Hide full trace on panics
	showScriptFiles = false // Show command/sub-shell filenames
)

// showUsage exits with error code 2.
//
func showUsage() {
	pad := strings.Repeat(" ", len(me)-1)
	fmt.Fprintf(errOut, "Usage:\n")
	fmt.Fprintf(errOut, "       %s -h | --help\n", me)
	fmt.Fprintf(errOut, "       %s (show help)\n", pad)
	fmt.Fprintf(errOut, "  or   %s [-r runfile] list\n", me)
	fmt.Fprintf(errOut, "       %s (list commands)\n", pad)
	fmt.Fprintf(errOut, "  or   %s [-r runfile] help <command>\n", me)
	fmt.Fprintf(errOut, "       %s (show help for <command>)\n", pad)
	fmt.Fprintf(errOut, "  or   %s [-r runfile] <command> [option ...]\n", me)
	fmt.Fprintf(errOut, "       %s (run <command>)\n", pad)
	fmt.Fprintln(errOut, "Options:")
	fmt.Fprintln(errOut, "  -h, --help")
	fmt.Fprintln(errOut, "        Show help screen")
	fmt.Fprintln(errOut, "  -r, --runfile <file>")
	fmt.Fprintf(errOut, "        Specify runfile (default='%s')\n", runfileDefault)
	fmt.Fprintln(errOut, "Note:")
	fmt.Fprintln(errOut, "  Options accept '-' | '--'")
	fmt.Fprintln(errOut, "  Values can be given as:")
	fmt.Fprintln(errOut, "        -o value | -o=value")
	fmt.Fprintln(errOut, "  For boolean options:")
	fmt.Fprintln(errOut, "        -f | -f=true | -f=false")
	fmt.Fprintln(errOut, "  Short options cannot be combined")
	// flag.PrintDefaults()
	os.Exit(2)
}

// main
//
func main() {
	errOut = os.Stderr
	me = path.Base(os.Args[0])
	// Configure logging
	//
	log.SetFlags(0)
	log.SetPrefix(path.Base(os.Args[0]) + ": ")
	// Capture panics as log messages
	//
	if hidePanic {
		defer func() {
			if r := recover(); r != nil {
				log.Fatal(r)
			}
		}()
	}
	// Shebang?
	//
	var shebangFile string
	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "shebang") {
		os.Args = append(os.Args[:1], os.Args[2:]...)
		if len(os.Args) > 1 {
			shebangFile = os.Args[1]
			os.Args = append(os.Args[:1], os.Args[2:]...)
		}
		shebangMode = len(shebangFile) > 0 && path.Base(shebangFile) != runfileDefault
	}
	// In shebang mode, we defer parsing args until we know if we are in "main" mode
	//
	if shebangMode {
		me = path.Base(shebangFile) // Script name = executable name for help
		inputFile = shebangFile     // shebang file = runfile
	} else {
		parseArgs()
	}
	// Read file into memory
	//
	fileBytes, err := readFile(inputFile)
	if err != nil {
		log.Printf("Error reading file '%s': %s\n", inputFile, err.Error())
		showUsage() // exits
	}
	// Parse the file
	//
	rfAst := parse(lex(fileBytes))
	rf := processAST(rfAst)
	// Setup Commands
	//
	listCmd := &command{
		name:   "list",
		title:  "(builtin) List available commands",
		help:   func() { listCommands() },
		run:    func() { listCommands() },
		rename: func(_ string) {},
	}
	helpCmd := &command{
		name:   "help",
		title:  "(builtin) Show help for a command",
		help:   showUsage,
		run:    func() { runHelp(rf) },
		rename: func(_ string) {},
	}
	commandMap["list"] = listCmd
	commandMap["help"] = helpCmd
	commandList = append(commandList, listCmd, helpCmd)
	rfCmdIndex := len(commandList)
	for name, rfcmd := range rf.cmds {
		name = strings.ToLower(name) // normalize
		if _, ok := commandMap[name]; ok {
			panic("Duplicate command: " + name)
		}
		cmd := &command{
			name:   rfcmd.name,
			title:  rfcmd.Title(),
			help:   func(c *runCmd) func() { return func() { showCmdHelp(c) } }(rfcmd),
			run:    func(c *runCmd) func() { return func() { runCommand(c) } }(rfcmd),
			rename: func(c *runCmd) func(string) { return func(s string) { c.name = s } }(rfcmd),
		}
		commandMap[name] = cmd
		commandList = append(commandList, cmd)
	}
	// In shebang mode, if only 1 command defined, named "main", default to it directly
	//
	mainMode = shebangMode &&
		len(commandList) == (rfCmdIndex+1) &&
		strings.EqualFold(commandList[rfCmdIndex].name, "main")
	// Determine which command to run
	//
	var cmdName string
	if mainMode {
		// In main mode, we defer parsing args to the command
		//
		os.Args = os.Args[1:] // Discard 'me'
		cmdName = "main"
		commandList[rfCmdIndex].rename(me) // Print help as script name
	} else {
		// If we deferred parsing args, now is the time
		//
		if shebangMode {
			parseArgs()
		}
		if len(os.Args) > 0 {
			cmdName, os.Args = os.Args[0], os.Args[1:]
		} else {
			// Default = first command in command list
			//
			cmdName = commandList[0].name
		}
	}
	// Run command, if present, else error
	//
	cmdName = strings.ToLower(cmdName) // normalize
	if cmd, ok := commandMap[cmdName]; ok {
		cmd.run()
	} else {
		log.Printf("command not found: %s", cmdName)
		listCommands()
		os.Exit(2)
	}
}

func parseArgs() {
	var showHelp bool
	flag.CommandLine.SetOutput(errOut)
	flag.CommandLine.Usage = showUsage // Invoked if error parsing args
	flag.BoolVar(&showHelp, "help", false, "")
	flag.BoolVar(&showHelp, "h", false, "")
	// No -r/--runfile support in shebang mode
	//
	if !shebangMode {
		flag.StringVar(&inputFile, "runfile", runfileDefault, "")
		flag.StringVar(&inputFile, "r", runfileDefault, "")
	}
	flag.Parse()
	os.Args = flag.Args()
	// Help?
	//
	if showHelp {
		showUsage()
	}
}

// Returns contents of file at specified path as a byte array
//
func readFile(path string) ([]byte, error) {
	var (
		err   error
		stat  os.FileInfo
		file  *os.File
		bytes []byte
	)

	// Stat the file
	//
	if stat, err = os.Stat(path); err == nil {
		// Confirm file is regular
		//
		if !stat.Mode().IsRegular() {
			return nil, errors.New("File not found")
		}
		// Open the file
		//
		if file, err = os.Open(path); err == nil {
			// Close file before we exit
			//
			defer file.Close()
			// Read file into memory
			//
			if bytes, err = ioutil.ReadAll(file); err == nil {
				return bytes, nil
			}
		}
	}
	// If we get here, we have an error
	//
	return nil, err
}

// traceFn
//
func traceFn(msg string, i interface{}) {
	if enableFnTrace {
		fnName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
		log.Println(msg, ":", fnName)
	}
}
