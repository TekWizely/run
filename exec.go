package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

var tempDir string

// executeCmdScript
//
func executeCmdScript(shell string, body []string, env map[string]string) {
	if shell == "" {
		panic("Shell not specified")
	}
	if len(body) == 0 {
		panic("Empty astCmd")
	}
	tmpFile, err := tempFile(fmt.Sprintf("cmd-%s-*.sh", shell))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(tmpFile.Name())
	defer tmpFile.Close()
	//defer os.Remove(tmpFile.Name()) // clean up

	for _, line := range body {
		if _, err = tmpFile.Write([]byte(line)); err != nil {
			log.Fatal(err)
		}
	}
	var cmd *exec.Cmd

	// Exec or shell ?
	//
	if shell == "exec" {
		// Try to make the cmd executable
		//
		var stat os.FileInfo
		if stat, err = tmpFile.Stat(); err != nil {
			log.Fatal(err)
		}
		// Add user-executable bit
		//
		if err = tmpFile.Chmod(stat.Mode() | 0100); err != nil {
			log.Fatal(err)
		}
		if err = tmpFile.Close(); err != nil {
			log.Fatal(err)
		}

		cmd = exec.Command(tmpFile.Name())
	} else {
		cmd = exec.Command("/usr/bin/env", shell, tmpFile.Name())
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	_ = cmd.Run()
}

// executeSubCommand
//
func executeSubCommand(shell string, command string, env map[string]string, out io.Writer) {
	if shell == "" {
		panic("Shell not specified")
	}
	if len(command) == 0 {
		panic("Empty sub command")
	}
	tmpFile, err := tempFile(fmt.Sprintf("sub-%s-*.sh", shell))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(tmpFile.Name())
	defer tmpFile.Close()
	//defer os.Remove(tmpFile.Name()) // clean up

	if _, err = tmpFile.Write([]byte(command)); err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command("/usr/bin/env", shell, tmpFile.Name())

	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	_ = cmd.Run()
}

// tempFile
//
func tempFile(pattern string) (*os.File, error) {
	if tempDir == "" {
		var err error
		tempDir, err = ioutil.TempDir("", "runfile-")
		if err != nil {
			return nil, err
		}
	}
	return ioutil.TempFile(tempDir, pattern)
}
