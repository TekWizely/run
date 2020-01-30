package exec

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/tekwizely/run/internal/config"
)

var tempDir string

func executeScript(shell string, script []string, args []string, env map[string]string, prefix string, out io.Writer) int {
	if shell == "" {
		panic(config.ErrShell)
	}
	if len(script) == 0 {
		return 0
	}
	tmpFile, err := tempFile(fmt.Sprintf("%s-%s-*.sh", prefix, shell))
	if err != nil {
		log.Fatal(err)
	}
	defer tmpFile.Close()
	if config.ShowScriptFiles {
		fmt.Fprintln(config.ErrOut, tmpFile.Name())
	} else {
		defer os.Remove(tmpFile.Name()) // clean up
	}

	for _, line := range script {
		if _, err = tmpFile.Write([]byte(line)); err != nil {
			log.Fatal(err)
		}
	}
	var cmd *exec.Cmd

	// Shebang or env ?
	//
	if shell == "#!" {
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
	err = cmd.Run()
	if err == nil {
		return 0
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode()
	}
	panic(err)
}

// ExecuteCmdScript executes a command script.
//
func ExecuteCmdScript(shell string, script []string, args []string, env map[string]string) int {
	return executeScript(shell, script, args, env, "cmd", os.Stdout)
}

// ExecuteSubCommand executes a command substitution.
//
func ExecuteSubCommand(shell string, command string, env map[string]string, out io.Writer) int {
	return executeScript(shell, []string{command}, []string{}, env, "sub", out)
}

// ExecuteTest will execute the test command against the supplied test string
//
func ExecuteTest(shell string, test string, env map[string]string) int {
	return executeScript(shell, []string{test}, []string{}, env, "test", os.Stdout)
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
