package runfile

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/tekwizely/run/internal/config"
	"github.com/tekwizely/run/internal/exec"
)

// NormalizeCmdScript normalizes the command script text.
// Removes leading and trailing lines that are empty or whitespace only.
// Removes all leading whitespace that matches leading whitespace on first non-empty line
//
func NormalizeCmdScript(txt []string) []string {
	if len(txt) == 0 {
		return txt
	}
	// Remove empty leading lines
	//
	for len(txt) > 0 && isLineWhitespaceOnly(txt[0]) {
		txt = txt[1:]
	}
	// Remove empty trailing lines
	//
	for len(txt) > 0 && isLineWhitespaceOnly(txt[len(txt)-1]) {
		txt = txt[:len(txt)-1]
	}
	// Still have anything?
	//
	if len(txt) > 0 {
		// Leading whitespace on first line is considered as indention-only
		//
		runes := []rune(txt[0])
		i := 0
		for isWhitespace(runes[i]) {
			i++
		}
		// Any leading ws?
		//
		if i > 0 {
			leadingWS := string(runes[:i])
			for j, line := range txt {
				if strings.HasPrefix(line, leadingWS) {
					txt[j] = line[len(leadingWS):]
				}
			}
		}

	}
	return txt
}

// NormalizeCmdDesc normalizes the command description text.
// Removes leading and trailing lines that are empty or whitespace only.
//
func NormalizeCmdDesc(txt []string) []string {
	if len(txt) == 0 {
		return txt
	}
	// Remove empty leading lines
	//
	for len(txt) > 0 && isLineWhitespaceOnly(txt[0]) {
		txt = txt[1:]
	}
	// Remove empty trailing lines
	//
	for len(txt) > 0 && isLineWhitespaceOnly(txt[len(txt)-1]) {
		txt = txt[:len(txt)-1]
	}
	return txt
}

// isLineWhitespaceOnly return true if the input contains ONLY (' ' | '\t' | '\n' | '\r')
//
func isLineWhitespaceOnly(line string) bool {

	for _, r := range line {
		// TODO Consider using a more liberal whitespace check ( i.e unicode.IsSpace() )
		if !isWhitespace(r) {
			return false
		}
	}
	return true
}

// isWhitespace return true if the input is one of (' ' | '\t' | '\n' | '\r')
//
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// cmdFlagOpt describes a command option flag.
//
type cmdFlagOpt interface {
	// Name fetches the name of the option variable name (not the arg short/long).
	//
	Name() string
	// IsSet returns true if the flag was set from the arguments.
	//
	IsSet() bool
	// Get retrieves the value.
	// Implements flag.Getter.
	//
	Get() interface{}
	// Set sets value.
	// Implements flag.Value.
	//
	Set(string) error
	// String gets the value as a string.
	// Implements flag.Value.
	//
	String() string
}

// stringOpt
//
type stringOpt struct {
	name  string // opt name, not short/long code
	value *string
	set   bool
}

func (a *stringOpt) Name() string {
	return a.name
}
func (a *stringOpt) IsSet() bool {
	return a.set
}
func (a *stringOpt) Set(value string) error {
	*a.value = value
	a.set = true
	return nil
}
func (a *stringOpt) Get() interface{} {
	return *a.value
}
func (a *stringOpt) String() string {
	if a.value == nil {
		return ""
	}
	return *a.value
}

// boolOpt
//
type boolOpt struct {
	name  string // opt name, not short/long code
	value *bool
	set   bool
}

func (a *boolOpt) Name() string {
	return a.name
}
func (a *boolOpt) IsSet() bool {
	return a.set
}
func (a *boolOpt) Set(value string) error {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	*a.value = b
	a.set = true
	return nil
}
func (a *boolOpt) Get() interface{} {
	return *a.value
}
func (a *boolOpt) String() string {
	if a.value == nil || !*a.value {
		return ""
	}
	return "1"
}
func (a *boolOpt) IsBoolFlag() bool {
	return true
}

// evaluateCmdOpts returns (args,0) or (nil,!0)
//
func evaluateCmdOpts(cmd *RunCmd, args []string) ([]string, int) {
	// If no options defined, pass all args through to command script
	// NOTE: For MainMode we still define options, mainly for --help
	//
	if len(cmd.Config.Opts) == 0 && !config.MainMode {
		return args, 0
	}
	flags := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	var (
		stringValues = make(map[string]*stringOpt)
		boolValues   = make(map[string]*boolOpt)
	)
	// Help : -h, --help
	//
	help := false
	hasHelpShort := false
	hasHelpLong := false
	for _, opt := range cmd.Config.Opts {
		// If explicitly added, then cannot be overridden
		//
		// 'h' != 'H'
		if opt.Short == 'h' {
			hasHelpShort = true
		}
		if strings.EqualFold(opt.Long, "help") {
			hasHelpLong = true
		}
		optName := opt.Name
		var flagOpt cmdFlagOpt
		// Bool or String?
		//
		if len(opt.Value) > 0 {
			var s = new(string)
			var sOpt = &stringOpt{name: optName, value: s}
			stringValues[optName] = sOpt
			flagOpt = sOpt
		} else {
			var b = new(bool)
			var bOpt = &boolOpt{name: optName, value: b}
			boolValues[optName] = bOpt
			flagOpt = bOpt
		}
		// Short?
		//
		if opt.Short != 0 {
			flags.Var(flagOpt, string([]rune{opt.Short}), "")
		}
		// Long?
		//
		if len(opt.Long) > 0 {
			flags.Var(flagOpt, strings.ToLower(opt.Long), "")
		}
	}
	if !hasHelpShort {
		flags.BoolVar(&help, "h", help, "")
	}
	if !hasHelpLong {
		flags.BoolVar(&help, "help", help, "")
	}
	exitCode := 0
	// Invoked if error parsing args - sets exit code 2
	//
	flags.Usage = func() {
		// Show less verbose usage.
		// User can use -h/--help for full desc+usage
		//
		showCmdUsage(cmd)
		exitCode = 2
	}
	_ = flags.Parse(args)
	if exitCode != 0 {
		return nil, exitCode
	}
	// User explicitly asked for help
	//
	if help {
		// Show full help details
		//
		ShowCmdHelp(cmd)
		return nil, 2
	}
	// TODO Maybe make args property instead of stashing in vars?
	for name, value := range stringValues {
		cmd.Scope.Vars[name] = value.String()
		cmd.Scope.AddExport(name)
	}
	for name, value := range boolValues {
		cmd.Scope.Vars[name] = value.String()
		cmd.Scope.AddExport(name)
	}
	return flags.Args(), 0
}

// ShowCmdHelp shows cmd, desc, usage and opts
//
//goland:noinspection GoUnhandledErrorResult // fmt.*
func ShowCmdHelp(cmd *RunCmd) {
	var shell = ""
	//goland:noinspection GoBoolExpressions
	if config.ShowCmdShells {
		shell = fmt.Sprintf(" (%s)", cmd.Shell())
	}

	if !cmd.EnableHelp() {
		fmt.Fprintf(config.ErrOut, "%s%s: no help available.\n", cmd.Name, shell)
		return
	}
	fmt.Fprintf(config.ErrOut, "%s%s:\n", cmd.Name, shell)
	// Desc
	//
	if len(cmd.Config.Desc) > 0 {
		for _, desc := range cmd.Config.Desc {
			fmt.Fprintf(config.ErrOut, "  %s\n", desc)
		}
		// } else {
		// 	fmt.Fprintf(errOut, "%s:\n", cmd.name)
	}
	showCmdUsage(cmd)
}

// ShowCmdUsage show only usage + opts
//
//goland:noinspection GoUnhandledErrorResult // fmt.*
func showCmdUsage(cmd *RunCmd) {
	var shell = ""
	//goland:noinspection GoBoolExpressions
	if config.ShowCmdShells {
		shell = fmt.Sprintf(" (%s)", cmd.Shell())
	}
	if !cmd.EnableHelp() {
		fmt.Fprintf(config.ErrOut, "%s%s: no help available.\n", cmd.Name, shell)
		return
	}
	// Usages
	//
	for i, usage := range cmd.Config.Usages {
		or := "or"
		if i == 0 {
			fmt.Fprintf(config.ErrOut, "Usage:\n")
			or = "  " // 2 spaces
		}
		pad := strings.Repeat(" ", len(cmd.Name)-1)
		if usage[0] == '(' {
			fmt.Fprintf(config.ErrOut, "       %s %s\n", pad, usage)
		} else {
			fmt.Fprintf(config.ErrOut, "  %s   %s %s\n", or, cmd.Name, usage)
		}
	}
	hasHelpShort := false
	hasHelpLong := false
	for _, opt := range cmd.Config.Opts {
		if opt.Short == 'h' {
			hasHelpShort = true
		}
		if opt.Long == "help" {
			hasHelpLong = true
		}
	}
	// Options
	//
	if len(cmd.Config.Opts) > 0 {
		fmt.Fprintln(config.ErrOut, "Options:")
		if !hasHelpShort || !hasHelpLong {
			switch {
			case !hasHelpShort && hasHelpLong:
				fmt.Fprintln(config.ErrOut, "  -h")
			case hasHelpShort && !hasHelpLong:
				fmt.Fprintln(config.ErrOut, "  --help")
			default:
				fmt.Fprintln(config.ErrOut, "  -h, --help")
			}
			fmt.Fprintln(config.ErrOut, "        Show full help screen")
		}
	}
	for _, opt := range cmd.Config.Opts {
		b := &strings.Builder{}
		b.WriteString("  ")
		if opt.Short != 0 {
			b.WriteRune('-')
			b.WriteRune(opt.Short)
		}
		if opt.Long != "" {
			if opt.Short != 0 {
				b.WriteString(", ")
			}
			b.WriteString("--")
			b.WriteString(opt.Long)
		}
		if opt.Value != "" {
			b.WriteRune(' ')
			b.WriteRune('<')
			b.WriteString(opt.Value)
			b.WriteRune('>')
		}
		if opt.Desc != "" {
			if opt.Short != 0 && opt.Long == "" && opt.Value == "" {
				b.WriteString("    ")
			} else {
				b.WriteString("\n        ") // Leading \n
			}
			b.WriteString(opt.Desc)
		}
		fmt.Fprintln(config.ErrOut, b.String())
	}
}

// ListCommands prints the list of commands read from the runfile
//
func ListCommands() {
	_, _ = fmt.Fprintln(config.ErrOut, "Commands:")
	padLen := 0
	for _, cmd := range config.CommandList {
		if len(cmd.Name) > padLen {
			padLen = len(cmd.Name)
		}
	}
	for _, cmd := range config.CommandList {
		_, _ = fmt.Fprintf(config.ErrOut, "  %s%s    %s\n", cmd.Name, strings.Repeat(" ", padLen-len(cmd.Name)), cmd.Title)
	}
}

// RunHelp shows help for the specified command.
// On success, returns exit code 0
// If command not found, prints error message and returns exit code 2
// If no command given, prints usage message and returns exit code 2
//
func RunHelp() int {
	var cmdName string
	if len(os.Args) > 0 {
		cmdName = os.Args[0]
		os.Args = os.Args[1:]
	}
	if len(cmdName) > 0 {
		cmdName = strings.ToLower(cmdName)
		if c, ok := config.CommandMap[cmdName]; ok {
			c.Help()
			return 0
		}
		// NOTE: No further 'see' messages when help invoked *with* a command
		//
		log.Printf("command not found: %s\n\n", cmdName) // 2 x \n
		ListCommands()
	} else {
		_, _ = fmt.Fprintf(config.ErrOut, "usage: '%s help <command>'\n\n", config.Me) // 2 x \n
		ListCommands()
		_, _ = fmt.Fprintf(config.ErrOut, "\nsee '%s --help' for more information\n", config.Me) // Leading \n
	}
	return 2
}

// RunCommand executes a command returning an exit code
//
func RunCommand(cmd *RunCmd) int {
	exitCode := 0
	os.Args, exitCode = evaluateCmdOpts(cmd, os.Args)
	if exitCode != 0 {
		return exitCode
	}
	env := make(map[string]string)
	for _, name := range cmd.Scope.GetExports() {
		if value, ok := cmd.Scope.GetVar(name); ok {
			env[name] = value
		} else {
			log.Printf("WARNING: exported variable not defined: '%s'", name)
		}
	}
	// Check Asserts - Uses global .SHELL
	//
	shell, ok := cmd.Scope.GetAttr(".SHELL")
	if !ok || len(shell) == 0 {
		shell = config.DefaultShell
	}
	for _, assert := range cmd.Scope.Asserts {
		if exec.ExecuteTest(shell, assert.Test, env) != 0 {
			// Print message if one configured
			//
			if len(assert.Message) > 0 {
				log.Printf("ERROR: %s:%d: %s", assert.Runfile, assert.Line, assert.Message)
			} else {
				log.Printf("ERROR: %s:%d: assertion failed", assert.Runfile, assert.Line)
			}
			// ~= log.Fatal
			return 1
		}
	}
	// Execute script - Uses cmd shell
	//
	shell = cmd.Shell()
	return exec.ExecuteCmdScript(shell, cmd.Script, os.Args, env)
}
