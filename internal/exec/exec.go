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

var tmpDir string

func executeScript(shell string, script []string, args []string, env map[string]string, prefix string, out io.Writer) int {
	if shell == "" {
		panic(config.ErrShell)
	}
	if len(script) == 0 {
		return 0
	}
	// Tmp file will be cleaned up via CleanupTemporaryDir
	//
	tmpFile, err := tmpFile(fmt.Sprintf("%s-%s-*.sh", prefix, shell))
	if err != nil {
		// ~= log.Fatal
		log.Print(err)
		return 1
	}
	defer func() { _ = tmpFile.Close() }()

	for _, line := range script {
		if _, err = tmpFile.Write([]byte(line)); err != nil {
			// ~= log.Fatal
			log.Print(err)
			return 1
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
			// ~= log.Fatal
			log.Print(err)
			return 1
		}
		// Add user-executable bit
		//
		if err = tmpFile.Chmod(stat.Mode() | 0100); err != nil {
			// ~= log.Fatal
			log.Print(err)
			return 1
		}
		if err = tmpFile.Close(); err != nil {
			// ~= log.Fatal
			log.Print(err)
			return 1
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

// tmpFile creates a temporary file relative to tmpDir
// Created files will be cleaned up in CleanupTemporaryDir
//
func tmpFile(pattern string) (*os.File, error) {
	if tmpDir == "" {
		var err error
		tmpDir, err = ioutil.TempDir("", "runfile-")
		//goland:noinspection GoBoolExpressions
		if config.ShowScriptTmpDir {
			_, _ = fmt.Fprintln(config.ErrOut, "temp dir: ", tmpDir)
		}
		if err != nil {
			return nil, err
		}
	}
	return ioutil.TempFile(tmpDir, pattern)
}

// CleanupTemporaryDir attempts to remove the previously created tmpDir
// and any files within it.
//
func CleanupTemporaryDir() error {
	//goland:noinspection GoBoolExpressions
	if tmpDir != "" && !config.ShowScriptTmpDir {
		return os.RemoveAll(tmpDir)
	}
	return nil
}
