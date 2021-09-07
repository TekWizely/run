package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tekwizely/run/internal/ast"
	"github.com/tekwizely/run/internal/config"
	"github.com/tekwizely/run/internal/lexer"
	"github.com/tekwizely/run/internal/parser"
	"github.com/tekwizely/run/internal/runfile"
)

const (
	runfileDefault = "Runfile"
)

var (
	inputFile string
	hidePanic = true // Hide full trace on panics
)

// showUsage exits with error code 2.
//
func showUsage() {
	runfileOpt := ""
	if config.EnableRunfileOverride {
		runfileOpt = "[-r runfile] "
	}
	pad := strings.Repeat(" ", len(config.Me)-1)
	fmt.Fprintf(config.ErrOut, "Usage:\n")
	fmt.Fprintf(config.ErrOut, "       %s -h | --help\n", config.Me)
	fmt.Fprintf(config.ErrOut, "       %s (show help)\n", pad)
	fmt.Fprintf(config.ErrOut, "  or   %s %slist\n", config.Me, runfileOpt)
	fmt.Fprintf(config.ErrOut, "       %s (list commands)\n", pad)
	fmt.Fprintf(config.ErrOut, "  or   %s %shelp <command>\n", config.Me, runfileOpt)
	fmt.Fprintf(config.ErrOut, "       %s (show help for <command>)\n", pad)
	fmt.Fprintf(config.ErrOut, "  or   %s %s<command> [option ...]\n", config.Me, runfileOpt)
	fmt.Fprintf(config.ErrOut, "       %s (run <command>)\n", pad)
	fmt.Fprintln(config.ErrOut, "Options:")
	fmt.Fprintln(config.ErrOut, "  -h, --help")
	fmt.Fprintln(config.ErrOut, "        Show help screen")
	if config.EnableRunfileOverride {
		fmt.Fprintln(config.ErrOut, "  -r, --runfile <file>")
		fmt.Fprintf(config.ErrOut, "        Specify runfile (default='%s')\n", runfileDefault)
	}
	fmt.Fprintln(config.ErrOut, "Note:")
	fmt.Fprintln(config.ErrOut, "  Options accept '-' | '--'")
	fmt.Fprintln(config.ErrOut, "  Values can be given as:")
	fmt.Fprintln(config.ErrOut, "        -o value | -o=value")
	fmt.Fprintln(config.ErrOut, "  Flags (booleans) can be given as:")
	fmt.Fprintln(config.ErrOut, "        -f | -f=true | -f=false")
	fmt.Fprintln(config.ErrOut, "  Short options cannot be combined")
	// flag.PrintDefaults()
	os.Exit(2)
}

// showVersion exits with error code 0
//
func showVersion() int {
	fmt.Println("run", versionString())
	return 0
}

// main
//
func main() {
	// Propagate the exit code correctly.
	// os.Exit aborts program immediately, so delay as long as possible
	// First defer in = last defer out
	cmdExitCode := 0
	defer func() {
		os.Exit(cmdExitCode)
	}()

	config.ErrOut = os.Stderr
	if exec, err := os.Executable(); err != nil { // Returns abs path on success
		config.RunBin = exec
	} else {
		config.RunBin = os.Args[0] // Punt to arg[0]
	}
	config.Me = path.Base(config.RunBin)
	// Configure logging
	//
	log.SetFlags(0)
	log.SetPrefix(config.Me + ": ")
	// Capture panics as log messages
	//
	//noinspection GoBoolExpressions
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
	if len(os.Args) > 1 {
		if strings.EqualFold(os.Args[1], "shebang") {
			os.Args = append(os.Args[:1], os.Args[2:]...)
			if len(os.Args) > 1 {
				shebangFile = os.Args[1]
				os.Args = append(os.Args[:1], os.Args[2:]...)
			}
			config.ShebangMode = len(shebangFile) > 0 && path.Base(shebangFile) != runfileDefault
		} else if strings.EqualFold(os.Args[1], "version") {
			cmdExitCode = showVersion()
			return
		}
	}
	// In shebang mode, we defer parsing args until we know if we are in "main" mode
	//
	if config.ShebangMode {
		config.Me = path.Base(shebangFile) // Script Name = executable Name for Help
		log.SetPrefix(config.Me + ": ")
		inputFile = shebangFile // shebang file = runfile
		config.EnableRunfileOverride = false
	} else {
		parseArgs()
	}
	// Verify file exists
	//
	if stat, err := os.Stat(inputFile); err == nil {
		if !stat.Mode().IsRegular() {
			log.Printf("Error reading file '%s': File not considered 'regular'\n", inputFile)
			showUsage() // exits
		}
		if config.RunFile, err = filepath.Abs(inputFile); err != nil {
			log.Printf("Error reading file '%s': Cannot determine absolute path\n", inputFile)
			showUsage() // exits
		}
	} else {
		log.Printf("Input file not found: '%s' : Please create the file or specify an alternative", inputFile)
		showUsage() // exits
	}
	// Read file into memory
	//
	fileBytes, err := readFile(config.RunFile)
	if err != nil {
		log.Printf("Error reading file '%s': %s\n", inputFile, err.Error())
		showUsage() // exits
	}
	// Parse the file
	//
	rfAst := parser.Parse(lexer.Lex(fileBytes))
	rf := ast.ProcessAST(rfAst)
	// Setup Commands
	//
	listCmd := &config.Command{
		Name:   "list",
		Title:  "(builtin) List available commands",
		Help:   func() { runfile.ListCommands() },
		Run:    func() int { runfile.ListCommands(); return 0 },
		Rename: func(_ string) {},
	}
	config.CommandMap["list"] = listCmd
	config.CommandList = append(config.CommandList, listCmd)
	helpCmd := &config.Command{
		Name:   "help",
		Title:  "(builtin) Show Help for a command",
		Help:   showUsage,
		Run:    func() int { runfile.RunHelp(rf); return 0 },
		Rename: func(_ string) {},
	}
	config.CommandMap["help"] = helpCmd
	config.CommandList = append(config.CommandList, helpCmd)
	// In shebang mode, Version registered as 'run-version'
	//
	versionName := "version"
	if config.ShebangMode {
		versionName = "run-version"
	}

	versionCmd := &config.Command{
		Name:   versionName,
		Title:  "(builtin) Show Run version",
		Help:   func() { showVersion() },
		Run:    showVersion,
		Rename: func(_ string) {},
	}
	config.CommandMap[versionName] = versionCmd
	config.CommandList = append(config.CommandList, versionCmd)
	builtinCnt := len(config.CommandList)
	for _, rfcmd := range rf.Cmds {
		name := strings.ToLower(rfcmd.Name) // normalize
		if _, ok := config.CommandMap[name]; ok {
			panic("Duplicate command: " + name)
		}
		cmd := &config.Command{
			Name:   rfcmd.Name,
			Title:  rfcmd.Title(),
			Help:   func(c *runfile.RunCmd) func() { return func() { runfile.ShowCmdHelp(c) } }(rfcmd),
			Run:    func(c *runfile.RunCmd) func() int { return func() int { return runfile.RunCommand(c) } }(rfcmd),
			Rename: func(c *runfile.RunCmd) func(string) { return func(s string) { c.Name = s } }(rfcmd),
		}
		config.CommandMap[name] = cmd
		config.CommandList = append(config.CommandList, cmd)
	}
	// In shebang mode, if only 1 runfile command defined, named "main", default to it directly
	//
	config.MainMode = config.ShebangMode &&
		len(config.CommandList) == (builtinCnt+1) &&
		strings.EqualFold(config.CommandList[builtinCnt].Name, "main")
	// Determine which command to run
	//
	var cmdName string
	if config.MainMode {
		// In main mode, we defer parsing args to the command
		//
		os.Args = os.Args[1:] // Discard 'Me'
		cmdName = "main"
		config.CommandList[builtinCnt].Rename(config.Me) // Print Help as script Name
	} else {
		// If we deferred parsing args, now is the time
		//
		if config.ShebangMode {
			parseArgs()
		}
		if len(os.Args) > 0 {
			cmdName, os.Args = os.Args[0], os.Args[1:]
		} else {
			// Default = first command in command list
			//
			cmdName = config.CommandList[0].Name
		}
	}
	// Run command, if present, else error
	//
	cmdName = strings.ToLower(cmdName) // normalize
	if cmd, ok := config.CommandMap[cmdName]; ok {
		cmdExitCode = cmd.Run()
	} else {
		log.Printf("command not found: %s", cmdName)
		runfile.ListCommands()
		os.Exit(2)
	}
}

func parseArgs() {
	var showHelp bool
	flag.CommandLine.SetOutput(config.ErrOut)
	flag.CommandLine.Usage = showUsage // Invoked if error parsing args
	flag.BoolVar(&showHelp, "help", false, "")
	flag.BoolVar(&showHelp, "h", false, "")
	// No -r/--runfile support in shebang mode
	//
	if config.EnableRunfileOverride {
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
		file  *os.File
		bytes []byte
	)

	// Open the file
	//
	if file, err = os.Open(path); err == nil {
		// Close file before we exit
		//
		//noinspection GoUnhandledErrorResult
		defer file.Close()
		// Read file into memory
		//
		if bytes, err = ioutil.ReadAll(file); err == nil {
			return bytes, nil
		}
	}
	// If we get here, we have an error
	//
	return nil, err
}
