package runfile

import (
	"strings"

	"github.com/tekwizely/run/internal/config"
)

// Runfile stores the processed file, ready to run.
//
type Runfile struct {
	Scope *Scope
	Cmds  []*RunCmd
}

// NewRunfile is a convenience method.
//
func NewRunfile() *Runfile {
	return &Runfile{
		Scope: NewScope(),
		Cmds:  []*RunCmd{},
	}
}

// RunCmdOpt captures an OPTION
//
type RunCmdOpt struct {
	Name  string
	Short rune
	Long  string
	Value string
	Desc  string
}

// RunCmdConfig captures the configuration for a command.
//
type RunCmdConfig struct {
	Shell  string
	Desc   []string
	Usages []string
	Opts   []*RunCmdOpt
}

// RunCmd captures a command.
//
type RunCmd struct {
	Name   string
	Config *RunCmdConfig
	Scope  *Scope
	Script []string
	Line   int
}

// Title fetches the first line of the description as the command title.
//
func (c *RunCmd) Title() string {
	if len(c.Config.Desc) > 0 {
		return c.Config.Desc[0]
	}
	return ""
}

// Shell fetches the shell for the command, defaulting to the global '.SHELL'.
//
func (c *RunCmd) Shell() string {
	var shell string
	if shell = c.Config.Shell; len(shell) == 0 {
		// Shebang?
		//
		if len(c.Script) > 0 && strings.HasPrefix(c.Script[0], "#!") {
			shell = "#!"
		} else {
			// Global shell configured?
			//
			var ok bool
			if shell, ok = c.Scope.Attrs[".SHELL"]; !ok || len(shell) == 0 {
				// Use default
				//
				shell = config.DefaultShell
			}
		}
	}
	return shell
}

// EnableHelp returns whether a help screen should be shown for a command.
// Returns false if there isn't any custom information to display.
//
func (c *RunCmd) EnableHelp() bool {
	return len(c.Config.Desc) > 0 || len(c.Config.Usages) > 0 || len(c.Config.Opts) > 0
}
