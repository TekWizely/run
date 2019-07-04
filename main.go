package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
)

// usage exits with error code 2.
//
func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s runfile [option ...]\n", path.Base(os.Args[0]))
	os.Exit(2)
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

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var _, script string
	_, script, os.Args = os.Args[0], os.Args[1], os.Args[2:] // skip script (args[1])
	// fmt.Printf("\nMe: '%s'\nScript:'%s'\nArgs: %v\n", me, script, os.Args)
	//	os.Executable()

	// Read file into memory
	//
	bytes, err := readFile(script)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file '%s': %s\n", script, err.Error())
		usage() // exits
	}
	// Parse the file
	//
	ast := parse(lex(string(bytes)))

	// Do have a command arg?
	//
	if len(os.Args) > 0 {
		cmd := os.Args[0]
		if cmdText := ast.Command(cmd); cmdText != nil {
			execute(cmdText.text)
		} else {
			panic("No command found named '" + cmd + "'")
		}
	} else {
		// Print list of commands
		//
		fmt.Println("Commands:")
		for _, cmd := range ast.Commands() {
			fmt.Println(cmd)
		}
	}
}

func execute(script []string) {
	tmpFile, err := ioutil.TempFile("", "runcmd-*.sh")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpFile.Name()) // clean up

	for _, line := range script {
		if _, err = tmpFile.Write([]byte(line)); err != nil {
			log.Fatal(err)
		}
	}
	if err = tmpFile.Close(); err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command("/usr/bin/env", "bash", tmpFile.Name(), "/tmp")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}
