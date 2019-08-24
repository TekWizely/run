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

func executeScript(shell string, script []string, args []string, env map[string]string, prefix string, out io.Writer) {
	if shell == "" {
		panic(errShell)
	}
	if len(script) == 0 {
		return
	}
	tmpFile, err := tempFile(fmt.Sprintf("%s-%s-*.sh", prefix, shell))
	if err != nil {
		log.Fatal(err)
	}
	defer tmpFile.Close()
	if showScriptFiles {
		fmt.Fprintln(errOut, tmpFile.Name())
	} else {
		defer os.Remove(tmpFile.Name()) // clean up
	}

	for _, line := range script {
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

		cmd = exec.Command(tmpFile.Name(), args...)
	} else {
		cmd = exec.Command("/usr/bin/env", append([]string{shell, tmpFile.Name()}, args...)...)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	// Merge passed-in env with os environment
	//
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	_ = cmd.Run()
}
func executeCmdScript(shell string, script []string, args []string, env map[string]string) {
	executeScript(shell, script, args, env, "cmd", os.Stdout)
}
func executeSubCommand(shell string, command string, env map[string]string, out io.Writer) {
	executeScript(shell, []string{command}, []string{}, env, "sub", out)
}

// // executeCmdScript
// //
// func executeCmdScriptDeprecated(shell string, script []string, env map[string]string) {
// 	if shell == "" {
// 		panic(errShell)
// 	}
// 	if len(script) == 0 {
// 		panic("Empty empty")
// 	}
// 	tmpFile, err := tempFile(fmt.Sprintf("cmd-%s-*.sh", shell))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer tmpFile.Close()
// 	if showScriptFiles {
// 		fmt.Fprintln(errOut, tmpFile.Name())
// 	} else {
// 		defer os.Remove(tmpFile.Name()) // clean up
// 	}
//
// 	for _, line := range script {
// 		if _, err = tmpFile.Write([]byte(line)); err != nil {
// 			log.Fatal(err)
// 		}
// 	}
// 	var cmd *exec.Cmd
//
// 	// Exec or shell ?
// 	//
// 	if shell == "exec" {
// 		// Try to make the cmd executable
// 		//
// 		var stat os.FileInfo
// 		if stat, err = tmpFile.Stat(); err != nil {
// 			log.Fatal(err)
// 		}
// 		// Add user-executable bit
// 		//
// 		if err = tmpFile.Chmod(stat.Mode() | 0100); err != nil {
// 			log.Fatal(err)
// 		}
// 		if err = tmpFile.Close(); err != nil {
// 			log.Fatal(err)
// 		}
//
// 		cmd = exec.Command(tmpFile.Name())
// 	} else {
// 		cmd = exec.Command("/usr/bin/env", shell, tmpFile.Name())
// 	}
//
// 	cmd.Stdin = os.Stdin
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr
// 	cmd.Env = os.Environ()
// 	for k, v := range env {
// 		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
// 	}
// 	_ = cmd.Run()
// }
//
// // executeSubCommand
// //
// func executeSubCommandDeprecated(shell string, command string, env map[string]string, out io.Writer) {
// 	if shell == "" {
// 		panic(errShell)
// 	}
// 	if len(command) == 0 {
// 		panic("Empty command")
// 	}
// 	tmpFile, err := tempFile(fmt.Sprintf("sub-%s-*.sh", shell))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer tmpFile.Close()
// 	if showScriptFiles {
// 		fmt.Fprintln(errOut, tmpFile.Name())
// 	} else {
// 		defer os.Remove(tmpFile.Name()) // clean up
// 	}
//
// 	if _, err = tmpFile.Write([]byte(command)); err != nil {
// 		log.Fatal(err)
// 	}
// 	cmd := exec.Command("/usr/bin/env", shell, tmpFile.Name())
//
// 	cmd.Stdin = os.Stdin
// 	cmd.Stdout = out
// 	cmd.Stderr = os.Stderr
// 	cmd.Env = os.Environ()
// 	for k, v := range env {
// 		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
// 	}
// 	_ = cmd.Run()
// }

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
