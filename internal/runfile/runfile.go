package runfile

import (
	"strings"

	"github.com/tekwizely/run/internal/config"
)

// CmdProvider allows us to construct commands
// multiple times in different contexts.
type CmdProvider interface {
	GetCmd(r *Runfile) *RunCmd
	GetCmdEnv(r *Runfile, env map[string]string) *RunCmd
}

// Runfile stores the processed file, ready to run.
//
type Runfile struct {
	Scope *Scope
	Cmds  []CmdProvider
}

// NewRunfile is a convenience method.
//
func NewRunfile() *Runfile {
	return &Runfile{
		Scope: NewScope(),
		Cmds:  []CmdProvider{},
	}
}

// RunCmdOpt captures an OPTION.
//
type RunCmdOpt struct {
	Name       string
	Required   bool
	HasDefault bool
	Default    string
	Short      rune
	Long       string
	Example    string
	Desc       string
}

// RunCmdRun captures a command config RUN invocation.
// TODO Better name?
//
type RunCmdRun struct {
	Command string
	Args    []string
}

// RunCmdConfig captures the configuration for a command.
//
type RunCmdConfig struct {
	Shell      string
	Desc       []string
	Usages     []string
	Opts       []*RunCmdOpt
	BeforeRuns []*RunCmdRun
	AfterRuns  []*RunCmdRun
}

// RunCmd captures a command.
//
type RunCmd struct {
	Name    string
	Config  *RunCmdConfig
	Scope   *Scope
	Script  []string
	Runfile string
	Line    int
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
