package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tekwizely/run/internal/ast"
	"github.com/tekwizely/run/internal/config"
	"github.com/tekwizely/run/internal/exec"
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
	hidePanic = true // Hide full trace on panics
)

// showUsageHint prints a terse usage string.
//
func showUsageHint() {
	_, _ = fmt.Fprintf(config.ErrOut, "see '%s --help' for more information\n", config.Me)
}

// showRunHelp
//
//goland:noinspection GoUnhandledErrorResult // fmt.*
func showRunHelp() {
	pad := strings.Repeat(" ", len(config.Me)-1)
	fmt.Fprintf(config.ErrOut, "Usage:\n")
	fmt.Fprintf(config.ErrOut, "       %s <command> [option ...]\n", config.Me)
	fmt.Fprintf(config.ErrOut, "       %s (run <command>)\n", pad)

	fmt.Fprintf(config.ErrOut, "  or   %s list\n", config.Me)
	fmt.Fprintf(config.ErrOut, "       %s (list commands)\n", pad)

	fmt.Fprintf(config.ErrOut, "  or   %s help <command>\n", config.Me)
	fmt.Fprintf(config.ErrOut, "       %s (show help for <command>)\n", pad)

	fmt.Fprintln(config.ErrOut, "Options:")
	if config.EnableRunfileOverride {
		fmt.Fprintln(config.ErrOut, "  -r, --runfile <file>")
		fmt.Fprintf(config.ErrOut, "        Specify runfile (default='${%s:-%s}')\n", runfileEnv, runfileDefault)
		fmt.Fprint(config.ErrOut, "        ex: run -r /my/runfile list\n")
	}
	fmt.Fprintln(config.ErrOut, "Note:")
	fmt.Fprintln(config.ErrOut, "  Options accept '-' | '--'")
	fmt.Fprintln(config.ErrOut, "  Values can be given as:")
	fmt.Fprintln(config.ErrOut, "        -o value | -o=value")
	fmt.Fprintln(config.ErrOut, "  Flags (booleans) can be given as:")
	fmt.Fprintln(config.ErrOut, "        -f | -f=true | -f=false")
	fmt.Fprintln(config.ErrOut, "  Short options cannot be combined")
	if !config.ShebangMode {
		fmt.Fprintln(config.ErrOut, "\nLearn more about run at https://github.com/TekWizely/run") // Leading \n
	}
}

// showVersion
//
func showVersion() {
	if config.ShebangMode {
		fmt.Printf("%s is powered by run %s. learn more at https://github.com/TekWizely/run\n", config.Me, versionString())
	} else {
		fmt.Println("run", versionString())
	}
}

// main
//
//goland:noinspection GoUnhandledErrorResult // fmt.*
func main() {
	// NOTE: Instead of os.Exit, set exitCode then return
	//
	exitCode := 0
	// First defer in = last defer out
	//
	defer func() {
		// Cleanup temp folder/files
		//
		_ = exec.CleanupTemporaryDir() // TODO Message on error?
		// Propagate exit code if non-0
		// os.Exit aborts program immediately, so delay as long as possible
		//
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	}()

	// Hack to allow ast to invoke parse without circular dependency
	//
	ast.ParseBytes = parser.ParseBytes

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
	log.SetPrefix(config.Me + ": ") // May change for Shebang Mode
	// Capture panics as log messages
	//
	//goland:noinspection GoBoolExpressions
	if hidePanic {
		defer func() {
			if r := recover(); r != nil {
				// ~= log.Fatal
				log.Print(r)
				exitCode = 1
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
	config.RunfileIsDefault = !config.ShebangMode // May change once we know more about runfile
	var exists bool
	var err error
	// In shebang mode, we defer parsing args until we know if we are in "main" mode
	//
	if config.ShebangMode {
		config.Me = path.Base(shebangFile) // Script Name = executable Name for Help
		log.SetPrefix(config.Me + ": ")
		config.Runfile = shebangFile // shebang file = runfile
		config.EnableRunfileOverride = false
		_, exists, err = util.StatIfExists(config.Runfile)
	} else {
		exitCode = parseArgs()
		if exitCode != 0 {
			return
		}
		// No fallback logic when user specifies file, even if its "Runfile"
		//
		if len(config.Runfile) > 0 {
			_, exists, err = util.StatIfExists(config.Runfile)
		} else {
			config.Runfile, _, exists, err = tryFindRunfile()
		}
	}
	var rf *runfile.Runfile
	var bytes []byte
	// Verify file exists
	//
	if exists {
		// Read from absolute path; Also used for .RUNFILE attribute
		//
		if config.RunfileAbs, err = filepath.Abs(config.Runfile); err != nil {
			log.Printf("ERROR: runfile '%s': cannot determine absolute path", config.Runfile)
			exitCode = 2
			return
		}
		// Runfile parent directory used for .RUNFILE.DIR attribute
		//
		config.RunfileAbsDir = filepath.Dir(config.RunfileAbs)
		// Current runfile/dir values, also used for .SELF / .SELF.DIR attributes
		//
		config.CurrentRunfile = util.TryMakeRelative(config.RunfileAbsDir, config.RunfileAbs)
		config.CurrentRunfileAbs = config.RunfileAbs
		config.CurrentRunfileAbsDir = config.RunfileAbsDir
		// Read the file (will re-check exists/stat, but that's the cost of the abstraction)
		//
		bytes, exists, err = util.ReadFileIfExists(config.RunfileAbs)
	}
	if exists {
		// Mark primary Runfile as already included (to avoid include loops)
		//
		config.IncludedFiles[config.RunfileAbs] = struct{}{}
		// Update IsDefault now that we know about the Runfile
		//
		config.RunfileIsDefault = !config.ShebangMode && config.Runfile == runfileDefault
		rfAst := parser.ParseBytes(bytes)
		// Process the AST
		//
		rf = ast.ProcessAST(rfAst)
		config.RunfileIsLoaded = true
	} else {
		if err == nil {
			log.Printf("ERROR: runfile '%s' not found: please create the file or specify an alternative\n\n", util.DefaultIfEmpty(config.Runfile, runfileDefault)) // 2 x \n
		} else {
			// If path error, hide the operation (stat, open, etc)
			//
			if pathErr, ok := err.(*os.PathError); ok {
				log.Printf("ERROR: %s: %s\n\n", pathErr.Path, pathErr.Err) // 2 x \n
			} else {
				log.Printf("ERROR: %s\n\n", err) // 2 x \n
			}
		}
	}
	// Setup Commands
	//
	listCmd := &config.Command{
		Name:    "list",
		Title:   "(builtin) List available commands",
		Help:    func() { runfile.ListCommands() },
		Run:     func(_ []string, _ map[string]string) int { runfile.ListCommands(); return 0 },
		Rename:  func(_ string) {},
		Builtin: true,
	}
	config.CommandMap[listCmd.Name] = listCmd
	config.CommandList = append(config.CommandList, listCmd)
	helpCmd := &config.Command{
		Name:    "help",
		Title:   "(builtin) Show help for a command",
		Help:    showRunHelp,
		Run:     func(_ []string, _ map[string]string) int { return runfile.RunHelp() },
		Rename:  func(_ string) {},
		Builtin: true,
	}
	config.CommandMap[helpCmd.Name] = helpCmd
	config.CommandList = append(config.CommandList, helpCmd)
	// In shebang mode, Version registered as 'run-version'
	//
	versionName := "version"
	if config.ShebangMode {
		versionName = "run-version"
	}
	versionCmd := &config.Command{
		Name:    versionName,
		Title:   "(builtin) Show run version",
		Help:    func() { showVersion() },
		Run:     func(_ []string, _ map[string]string) int { showVersion(); return 0 },
		Rename:  func(_ string) {},
		Builtin: true,
	}
	config.CommandMap[versionName] = versionCmd
	config.CommandList = append(config.CommandList, versionCmd)
	builtinCnt := len(config.CommandList)

	// Register runfile commands, if loaded
	//
	if rf != nil {
		firstRunCommandsByName := make(map[string]*runfile.RunCmd) // Keep track of first occurrences for name and default title/description
		runCommandsByFileAndName := make(map[string]map[string]*runfile.RunCmd)
		runCommandsByName := make(map[string]*runfile.RunCmd)
		for _, cmdProvider := range rf.Cmds {
			newRunCommand := cmdProvider.GetCmd(rf)
			newRunCommandName := newRunCommand.Name     // Un-normalized name used for help
			name := strings.ToLower(newRunCommand.Name) // normalize
			// Keep track of which Runfile a command is registered in, to avoid dupes from same file
			//
			runCommandsByNameForFile, ok := runCommandsByFileAndName[newRunCommand.Runfile]
			if !ok {
				runCommandsByNameForFile = make(map[string]*runfile.RunCmd)
				runCommandsByFileAndName[newRunCommand.Runfile] = runCommandsByNameForFile
			}
			// Look for dupes
			//
			configIndex := -1 // Keep original CommandList index when overriding commands; Makes help lists consistent
			if firstRunCommand, ok := firstRunCommandsByName[name]; ok {
				existingConfigCommand := config.CommandMap[name] // Should ALWAYS succeed
				// Can't override builtin commands
				//
				if existingConfigCommand.Builtin {
					panic(fmt.Sprintf("%s:%d cannot override built-in command %s", newRunCommand.Runfile, newRunCommand.Line, name))
				}
				// Can't override commands defined in same runfile
				//
				if oldRunCommandForFile, ok := runCommandsByNameForFile[name]; ok {
					panic(fmt.Sprintf("%s: command %s defined multiple times in the same file: lines %d and %d", newRunCommand.Runfile, name, oldRunCommandForFile.Line, newRunCommand.Line))
				}
				// OK to override command, but notify user
				//
				if config.ShowNotices {
					if oldRunCommand, ok := runCommandsByName[name]; ok { // Should ALWAYS succeed
						log.Printf("NOTICE: %s:%d command %s overrides command %s defined in %s:%d", newRunCommand.Runfile, newRunCommand.Line, newRunCommand.Name, oldRunCommand.Name, oldRunCommand.Runfile, oldRunCommand.Line)
					}
				}
				// Remove old command, taking note of command name and CommandList index, which will be re-used
				//
				delete(config.CommandMap, name) // Technically not needed but feels cleaner
				for i := range config.CommandList {
					if config.CommandList[i] == existingConfigCommand {
						configIndex = i
						break
					}
				}
				// For help display, first registered command wins
				//
				newRunCommandName = firstRunCommand.Name
				// If no Title/Description defined, use first command's as its considered canonical
				//
				if len(newRunCommand.Config.Desc) == 0 {
					newRunCommand.Config.Desc = firstRunCommand.Config.Desc // TODO Make defensive copy?
				}
			}
			// Register cmd
			//
			runCommandsByNameForFile[name] = newRunCommand
			runCommandsByName[name] = newRunCommand
			cmd := &config.Command{
				Name:  newRunCommandName,
				Title: newRunCommand.Title(),
				Help:  func(c *runfile.RunCmd) func() { return func() { runfile.ShowCmdHelp(c) } }(newRunCommand),
				Run: func(a runfile.CmdProvider) func([]string, map[string]string) int {
					return func(args []string, env map[string]string) int { return runfile.RunCommand(a, rf, args, env) }
				}(cmdProvider),
				Rename:  func(c *runfile.RunCmd) func(string) { return func(s string) { c.Name = s } }(newRunCommand),
				Builtin: false,
			}
			config.CommandMap[name] = cmd
			// Append or replace?
			//
			if configIndex >= 0 {
				config.CommandList[configIndex] = cmd
			} else {
				config.CommandList = append(config.CommandList, cmd)
				firstRunCommandsByName[name] = newRunCommand
			}
		}
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
			exitCode = parseArgs()
			if exitCode != 0 {
				return
			}
		}
		if len(os.Args) > 0 {
			cmdName, os.Args = os.Args[0], os.Args[1:]
		} else {
			//
			// Default (no command) action
			//

			if config.RunfileIsLoaded && !config.ShebangMode /*&& !config.RunfileIsDefault*/ {
				fmt.Fprintf(config.ErrOut, "using runfile: %s\n\n", config.RunfileAbs) // 2 x \n
			}
			runfile.ListCommands()

			pad := strings.Repeat(" ", len(config.Me)-1)
			fmt.Fprintf(config.ErrOut, "\nUsage:\n") // Leading \n
			fmt.Fprintf(config.ErrOut, "       %s <command> [option ...]\n", config.Me)
			fmt.Fprintf(config.ErrOut, "       %s (run <command>)\n", pad)
			fmt.Fprintf(config.ErrOut, "  or   %s help <command>\n", config.Me)
			fmt.Fprintf(config.ErrOut, "       %s (show help for <command>)\n\n", pad) // 2 x \n
			showUsageHint()
			exitCode = 2
			return
		}
	}
	// Run command, if present, else error
	//
	cmdName = strings.ToLower(cmdName) // normalize
	var cmd *config.Command
	var ok bool
	if cmd, ok = config.CommandMap[cmdName]; !ok {
		log.Printf("command not found: %s", cmdName)
		runfile.ListCommands()
		showUsageHint()
		exitCode = 2
		return
	}
	exitCode = cmd.Run(os.Args, map[string]string{})
}

func parseArgs() int {
	flag.CommandLine.Init(config.Me, flag.ContinueOnError)
	flag.CommandLine.SetOutput(config.ErrOut)

	var showHelp bool
	flag.BoolVar(&showHelp, "help", false, "")
	flag.BoolVar(&showHelp, "h", false, "")
	// No $RUNFILE/-r/--runfile support in shebang mode
	//
	if config.EnableRunfileOverride {
		defaultInputFile := util.GetEnvOrDefault(runfileEnv, "")
		flag.StringVar(&config.Runfile, "runfile", defaultInputFile, "")
		flag.StringVar(&config.Runfile, "r", defaultInputFile, "")
	}
	exitCode := 0
	// Invoked if error parsing args - sets exit code 2
	//
	flag.CommandLine.Usage = func() {
		showUsageHint()
		exitCode = 2
	}
	flag.Parse()
	if exitCode != 0 {
		return exitCode
	}
	// Help?
	//
	if showHelp {
		showRunHelp()
		return 2
	}
	os.Args = flag.Args()
	return 0
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
func tryFindRunfile() (inputFile string, stat os.FileInfo, exists bool, err error) {
	// Look for default Runfile
	//
	if stat, exists, err = util.StatIfExists(runfileDefault); exists {
		return runfileDefault, stat, exists, err
	}
	// Look for root to possibly walk-up
	//
	if runfileRoots := filepath.SplitList(os.Getenv(runfileRoots)); len(runfileRoots) > 0 {
		// Need current working directory
		//
		var wd string
		if wd, err = os.Getwd(); err != nil {
			return "", nil, false, err
		}
		wd = path.Clean(wd)
		if !filepath.IsAbs(wd) {
			return "", nil, false, fmt.Errorf("working directory is not absolute: %v", wd)
		}
		// If we're already at the root, no need to look further
		//
		var wdDir string
		if wdDir = filepath.Dir(wd); wdDir == wd {
			return "", nil, false, errors.New("'Runfile' not found. please create file or specify an alternative")
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
				if _root == home && home == wd {
					root = _root
					break
				}
				// !$HOME can only match subdirectory
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
				if stat, exists, err = util.StatIfExists(inputFile); exists {
					return
				}
				wd = path.Dir(wd)
			}
			// root exclusion exemption for $HOME
			//
			if root == home && err == nil && rel == "." {
				inputFile = filepath.Join(wd, runfileDefault)
				if stat, exists, err = util.StatIfExists(inputFile); exists {
					return
				}
			}
		}
	}
	// Nothing else to try, sorry dude
	//
	return "", nil, false, errors.New("'Runfile' not found. please create file or specify an alternative")
}
