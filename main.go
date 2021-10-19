package main

import (
	"errors"
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
	"github.com/tekwizely/run/internal/exec"
	"github.com/tekwizely/run/internal/lexer"
	"github.com/tekwizely/run/internal/parser"
	"github.com/tekwizely/run/internal/runfile"
	"github.com/tekwizely/run/internal/util"
)

const (
	runfileDefault = "Runfile"
	runfileEnv     = "RUNFILE"
	runfileRoots   = "RUNFILE_ROOTS"
)

var (
	inputFile string
	hidePanic = true // Hide full trace on panics
)

// showUsage exits with error code 2.
//
//goland:noinspection GoUnhandledErrorResult
func showUsage() {
	runfileOpt := ""
	if config.EnableRunfileOverride {
		runfileOpt = "[-r runfile] " // needs trailing space
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
		fmt.Fprintf(config.ErrOut, "        Specify runfile (default='${%s:-%s}')\n", runfileEnv, runfileDefault)
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

// showVersion
//
func showVersion() {
	fmt.Println("run", versionString())
}

// main
//
func main() {
	// NOTE: Only set this from exit status of run commands (builtins ok too)
	//       Actual errors in Run proper should invoke os.Exit directly
	cmdExitCode := 0
	// First defer in = last defer out
	//
	defer func() {
		// Cleanup temp folder/files
		//
		_ = exec.CleanupTemporaryDir() // TODO Message on error?
		// Propagate cmd exit code if non-0
		// os.Exit aborts program immediately, so delay as long as possible
		//
		if cmdExitCode != 0 {
			os.Exit(cmdExitCode)
		}
	}()

	config.ErrOut = os.Stderr
	if execPath, err := os.Executable(); err != nil { // Returns abs path on success
		config.RunBin = execPath
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
			showVersion()
			return // Exit early
		}
	}
	var stat os.FileInfo
	var err error
	// In shebang mode, we defer parsing args until we know if we are in "main" mode
	//
	if config.ShebangMode {
		config.Me = path.Base(shebangFile) // Script Name = executable Name for Help
		log.SetPrefix(config.Me + ": ")
		inputFile = shebangFile // shebang file = runfile
		config.EnableRunfileOverride = false
		stat, err = os.Stat(inputFile)
	} else {
		parseArgs()
		// No fallback logic when user specifies file, even if its "Runfile"
		//
		if len(inputFile) > 0 {
			stat, err = os.Stat(inputFile)
		} else {
			inputFile, stat, err = tryFindRunfile()
		}
	}
	// Verify file exists
	//
	if err == nil {
		if !stat.Mode().IsRegular() {
			log.Printf("Error reading file '%s': File not considered 'regular'\n", inputFile)
			showUsage() // exits
		}
		if config.RunFile, err = filepath.Abs(inputFile); err != nil {
			log.Printf("Error reading file '%s': Cannot determine absolute path\n", inputFile)
			showUsage() // exits
		}
	} else {
		// TODO log err?
		log.Printf("Input file not found: '%s' : Please create the file or specify an alternative", util.DefaultIfEmpty(inputFile, runfileDefault))
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
		Help:   showUsage, // exits
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
		Run:    func() int { showVersion(); return 0 },
		Rename: func(_ string) {},
	}
	config.CommandMap[versionName] = versionCmd
	config.CommandList = append(config.CommandList, versionCmd)
	builtinCnt := len(config.CommandList)
	commandLines := make(map[string]int)

	for _, rfcmd := range rf.Cmds {
		name := strings.ToLower(rfcmd.Name) // normalize
		if _, ok := config.CommandMap[name]; ok {
			if prevline, ok := commandLines[name]; ok {
				panic(fmt.Sprintf("Duplicate command: %s defined on line %d -- originally defined on line %d", name, rfcmd.Line, prevline))
			} else {
				panic(fmt.Sprintf("Duplicate command: %s defined on line %d -- this command is built-in", name, rfcmd.Line))
			}
		}
		commandLines[name] = rfcmd.Line
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
	flag.CommandLine.Usage = showUsage // exits - Invoked if error parsing args
	flag.BoolVar(&showHelp, "help", false, "")
	flag.BoolVar(&showHelp, "h", false, "")
	// No $RUNFILE/-r/--runfile support in shebang mode
	//
	if config.EnableRunfileOverride {
		defaultInputFile := util.GetEnvOrDefault(runfileEnv, "")
		flag.StringVar(&inputFile, "runfile", defaultInputFile, "")
		flag.StringVar(&inputFile, "r", defaultInputFile, "")
	}
	flag.Parse()
	os.Args = flag.Args()
	// Help?
	//
	if showHelp {
		showUsage() // exits
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

// tryFindRunfile attempts to locate a Runfile
//
// * Checks ${PWD}/Runfile
// * Checks $RUNFILE_ROOTS
//   - Behaves largely similar to GIT_CEILING_DIRECTORIES
//   - if $PWD is under any entry, then walks up looking for 'Runfile'
//   - Stops walking up at specified root value
//   - Does not check root folder itself
//   - Except for $HOME, which will be checked if present on $RUNFILE_ROOTS
//
func tryFindRunfile() (inputFile string, stat os.FileInfo, err error) {
	// Look for default Runfile
	//
	if stat, err = os.Stat(runfileDefault); err == nil {
		return runfileDefault, stat, err
	}
	// Look for root to possibly walk-up
	//
	if runfileRoots := filepath.SplitList(os.Getenv(runfileRoots)); len(runfileRoots) > 0 {
		// Need current working directory
		//
		var wd string
		if wd, err = os.Getwd(); err != nil {
			return "", nil, err
		}
		wd = path.Clean(wd)
		if !filepath.IsAbs(wd) {
			return "", nil, fmt.Errorf("working directory is not absolute: %v", wd)
		}
		// If we're already at the root, no need to look further
		//
		var wdDir string
		if wdDir = filepath.Dir(wd); wdDir == wd {
			return "", nil, errors.New("file not found")
		}
		// $HOME gets special treatment as an INCLUSIVE root
		//
		var home = os.Getenv("HOME") // not present => ''
		// Loop over $RUNFILE_ROOTS, stopping at FIRST match
		//
		var root string
		for _, _root := range runfileRoots {
			_root = path.Clean(_root)
			if filepath.IsAbs(_root) { // TODO Log false?
				// $HOME can match exactly
				//
				if _root == home && _root == wd {
					root = _root
					break
				}
				// !$HOME can only match sub-directory
				//
				if rel, err := filepath.Rel(_root, wd); err == nil && len(rel) > 0 && !strings.HasPrefix(rel, ".") {
					root = _root
					break
				}
			}
		}
		// Did we find one?
		//
		if len(root) > 0 {
			// We know current wd doesn't have it, so let's start at parent
			//
			wd = wdDir
			// In general, we don't check root proper, so stop if we get there
			//
			var rel string
			for rel, err = filepath.Rel(root, wd); err == nil && len(rel) > 0 && !strings.HasPrefix(rel, "."); rel, err = filepath.Rel(root, wd) {
				inputFile = filepath.Join(wd, runfileDefault)
				if stat, err = os.Stat(inputFile); err == nil {
					return
				}
				wd = path.Dir(wd)
			}
			// root exclusion exemption for $HOME
			//
			if root == home && err == nil && rel == "." {
				inputFile = filepath.Join(wd, runfileDefault)
				if stat, err = os.Stat(inputFile); err == nil {
					return
				}
			}
		}
	}
	// Nothing else to try, sorry dude
	//
	return "", nil, errors.New("file not found")
}
