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
)

var (
	inputFile string
	showHelp  bool
)

func init() {
	const (
		runfileDefault  = "Runfile"
		runfileUsage    = "specify runfile to use"
		showHelpDefault = false
		showHelpUsage   = "show usage screen"
	)
	flag.StringVar(&inputFile, "runfile", runfileDefault, runfileUsage)
	flag.BoolVar(&showHelp, "help", showHelpDefault, showHelpUsage)
	flag.BoolVar(&showHelp, "h", showHelpDefault, showHelpUsage+" (shorthand)")
}

// usage exits with error code 2.
//
func usage() {
	me := path.Base(os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "usage: %s -h | -help\n", me)
	fmt.Fprintf(flag.CommandLine.Output(), "     : %s [-runfile runfile] list\n", me)
	fmt.Fprintf(flag.CommandLine.Output(), "     : %s [-runfile runfile] <command> [option ...]\n", me)
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	// Configure logging
	//
	log.SetFlags(0)
	log.SetPrefix(path.Base(os.Args[0]) + ": ")
	// Capture panics as log messages
	//
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		log.Fatal(r)
	// 	}
	// }()
	flag.Parse()
	os.Args = flag.Args()
	fmt.Printf("\nArgs: %v\n", os.Args)
	if showHelp {
		usage()
	}
	cmdName := "list"
	if len(os.Args) > 0 {
		cmdName, os.Args = os.Args[0], os.Args[1:]
	}

	// Read file into memory
	//
	fileBytes, err := readFile(inputFile)
	if err != nil {
		log.Printf("Error reading file '%s': %s\n", inputFile, err.Error())
		usage() // exits
	}
	// Parse the file
	//
	ast := parse(lex(fileBytes))

	switch {
	case cmdName == "list":
		listCommands(ast, os.Stdout)
	case !ast.hasCommand(cmdName):
		log.Printf("command not found: %s", cmdName)
		listCommands(ast, os.Stderr)
		os.Exit(2)
	default:
		rf := processAST(ast)
		cmd := rf.cmds[cmdName]
		shell := defaultIfEmpty(cmd.shell, cmd.attrs[".SHELL"])
		executeCmdScript(shell, cmd.script, cmd.env)
	}
}

// listCommands prints the list of commands read from the runfile
//
func listCommands(ast *ast, out io.Writer) {
	fmt.Fprintln(out, "Commands:")
	for cmd := range ast.commands {
		fmt.Fprintln(out, "\t", cmd)
	}
}

// Returns contents of file at specified path as a byte array
//
func readFile(path string) ([]byte, error) {
	var err error
	// Stat the file
	//
	if stat, err := os.Stat(path); err == nil {
		// Confirm file is regular
		//
		if !stat.Mode().IsRegular() {
			return nil, errors.New("File not found")
		}
		// Open the file
		//
		if file, err := os.Open(path); err == nil {
			// Close file before we exit
			//
			defer file.Close()
			// Read file into memory
			//
			if bytes, err := ioutil.ReadAll(file); err == nil {
				return bytes, nil
			}
		}
	}
	// If we get here, we have an error
	//
	return nil, err
}
