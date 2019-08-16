package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// normalizeCmdScript
// Removes leading and trailing lines that are empty or whitespace only.
// Removes all leading whitepace that matches leading whitespace on first non-empty line
//
func normalizeCmdScript(txt []string) []string {
	// Remove empty leading lines
	//
	for isLineWhitespaceOnly(txt[0]) {
		txt = txt[1:]
	}
	// Remove empty trailing lines
	//
	for isLineWhitespaceOnly(txt[len(txt)-1]) {
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

// flagOpt
//
type flagOpt interface {
	Name() string
	IsSet() bool
	String() string
	Set(string) error
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

// evaluateCmdOpts
//
func evaluateCmdOpts(cmd *runCmd, args []string) []string {
	flags := flag.NewFlagSet(cmd.name, flag.ExitOnError)
	flags.Usage = func() {
		if cmd.EnableUsage() {
			showCmdUsage(cmd)
		}
	}

	var (
		stringVals = make(map[string]*stringOpt)
		boolVals   = make(map[string]*boolOpt)
	)
	// Help : -h, --help
	//
	help := false
	if cmdAddHelp {
		flags.BoolVar(&help, "h", help, "")
		flags.BoolVar(&help, "help", help, "")
	}
	for _, opt := range cmd.config.opts {
		if cmdAddHelp {
			// 'h' != 'H'
			if opt.short == 'h' {
				panic(fmt.Sprintf("%s: Cannot redefine '-h'", cmd.name))
			}
			if strings.EqualFold(opt.long, "help") {
				panic(fmt.Sprintf("%s: Cannot redefine '--help'", cmd.name))
			}
		}
		optName := opt.name
		var flag flagOpt
		// Bool or String?
		//
		if len(opt.value) > 0 {
			var s = new(string)
			var sOpt = &stringOpt{name: optName, value: s}
			stringVals[optName] = sOpt
			flag = sOpt
		} else {
			var b = new(bool)
			var bOpt = &boolOpt{name: optName, value: b}
			boolVals[optName] = bOpt
			flag = bOpt
		}
		// Short?
		//
		if opt.short != 0 {
			flags.Var(flag, string([]rune{opt.short}), "")
		}
		// Long?
		//
		if len(opt.long) > 0 {
			flags.Var(flag, strings.ToLower(opt.long), "")
		}
	}
	_ = flags.Parse(args)
	if cmdAddHelp && help {
		showCmdHelp(cmd)
		os.Exit(2)
	}
	for name, value := range stringVals {
		cmd.env[name] = value.String()
	}
	for name, value := range boolVals {
		cmd.env[name] = value.String()
	}
	return flags.Args()
}

// showCmdHelp shows cmd, desc, usage and opts
//
func showCmdHelp(cmd *runCmd) {
	if cmd.EnableHelp() {
		fmt.Fprintf(errOut, "%s (%s):\n", cmd.name, cmd.Shell())
		// Desc
		//
		if len(cmd.config.desc) > 0 {
			for _, desc := range cmd.config.desc {
				fmt.Fprintf(errOut, "  %s\n", desc)
			}
			// } else {
			// 	fmt.Fprintf(errOut, "%s:\n", cmd.name)
		}
		showCmdUsage(cmd)
	} else {
		fmt.Fprintf(errOut, "%s (%s): No help available.\n", cmd.name, cmd.Shell())
	}
}

// showCmdUsage show only usage + opts
//
func showCmdUsage(cmd *runCmd) {
	// Usages
	//
	for i, usage := range cmd.config.usages {
		if i == 0 {
			fmt.Fprintf(errOut, "Usage:\n")
		}
		pad := strings.Repeat(" ", len(cmd.name)-1)
		if usage[0] == '(' {
			fmt.Fprintf(errOut, "       %s %s\n", pad, usage)
		} else {
			fmt.Fprintf(errOut, "  or   %s %s\n", cmd.name, usage)
		}
	}
	// Options
	//
	if len(cmd.config.opts) > 0 || cmdAddHelp {
		fmt.Fprintln(errOut, "Options:")
		if cmdAddHelp {
			fmt.Fprintln(errOut, "  -h, --help")
			fmt.Fprintln(errOut, "  \tShow full help screen")
		}
	}
	for _, opt := range cmd.config.opts {
		b := &strings.Builder{}
		b.WriteString("  ")
		if opt.short != 0 {
			b.WriteRune('-')
			b.WriteRune(opt.short)
		}
		if opt.long != "" {
			if opt.short != 0 {
				b.WriteString(", ")
			}
			b.WriteString("--")
			b.WriteString(opt.long)
		}
		if opt.value != "" {
			b.WriteRune(' ')
			b.WriteRune('<')
			b.WriteString(opt.value)
			b.WriteRune('>')
		}
		if opt.desc != "" {
			if opt.short != 0 && opt.long == "" && opt.value == "" {
				b.WriteString("    ")
			} else {
				b.WriteString("\n        ")
			}
			b.WriteString(opt.desc)
		}
		fmt.Fprintln(errOut, b.String())
	}
}

// listCommands prints the list of commands read from the runfile
//
func listCommands() {
	fmt.Fprintln(errOut, "Commands:")
	padLen := 0
	for _, cmd := range commandList {
		if len(cmd.name) > padLen {
			padLen = len(cmd.name)
		}
	}
	for _, cmd := range commandList {
		fmt.Fprintf(errOut, "  %s%s    %s\n", cmd.name, strings.Repeat(" ", padLen-len(cmd.name)), cmd.title)
	}
	pad := strings.Repeat(" ", len(me)-1)
	fmt.Fprintf(errOut, "Usage:\n")
	fmt.Fprintf(errOut, "       %s [-r runfile] help <command>\n", me)
	fmt.Fprintf(errOut, "       %s (show help for <command>)\n", pad)
	fmt.Fprintf(errOut, "  or   %s [-r runfile] <command> [option ...]\n", me)
	fmt.Fprintf(errOut, "       %s (run <command>)\n", pad)
}

// runHelp
//
func runHelp(rf *runfile) {
	cmdName := "help"
	// Command?
	//
	if len(os.Args) > 0 {
		cmdName = os.Args[0]
		os.Args = os.Args[1:]
	}
	if c, ok := commandMap[cmdName]; ok {
		c.help()
	} else {
		log.Printf("command not found: %s", cmdName)
		listCommands()
		os.Exit(2)
	}

}

// runCommand
//
func runCommand(cmd *runCmd) {
	os.Args = evaluateCmdOpts(cmd, os.Args)
	shell := defaultIfEmpty(cmd.config.shell, cmd.attrs[".SHELL"])
	executeCmdScript(shell, cmd.script, os.Args, cmd.env)
}
