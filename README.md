# Run: Easily manage and invoke small scripts and wrappers

![GitHub repo size](https://img.shields.io/github/repo-size/TekWizely/run)<!-- ALL-CONTRIBUTORS-BADGE:START - Do not remove or modify this section -->
[![All Contributors](https://img.shields.io/badge/all_contributors-5-orange.svg?style=flat-square)](#contributors-)<!-- ALL-CONTRIBUTORS-BADGE:END -->
![GitHub stars](https://img.shields.io/github/stars/TekWizely/run?style=social)
![GitHub forks](https://img.shields.io/github/forks/TekWizely/run?style=social)
![Twitter Follow](https://img.shields.io/twitter/follow/TekWizely?style=social)

Do you find yourself using tools like `make` to manage non build-related scripts?

Build tools are great, but they are not optimized for general script management.

Run aims to be better at managing small scripts and wrappers, while incorporating a familiar make-like syntax.

#### Runfile

Where make has the ubiquitous Makefile, run has the cleverly-named `"Runfile"`

By default, run will look for a file named `"Runfile"` in the current directory, exiting with error if not found.

Read below for details on specifying alternative runfiles, as well as other special modes you might find useful.

#### Commands

In place of make's targets, runfiles contain `'commands'`.

Similar to make, a command's label is used to invoke it from the command-line.

#### Scripts

Instead of recipes, each runfile command contains a `'script'` which is executed when the command is invoked.

You might be used to make's (default) behavior of executing each line of a recipe in a separate sub-shell.

In run, the entire script is executed within a single sub-shell.


#### TOC

- [Examples](#examples)
- [Special Modes](#special-modes)
- [Installing](#installing)
- [Contributing](#contributing)
- [Contact](#contact)
- [License](#license)
- [Just Looking for Bash Arg Parsing?](#just-looking-for-bash-arg-parsing)


-----------
## Examples

 - [Simple Command Definitions](#simple-command-definitions)
   - [Naming Commands](#naming-commands)
 - [Simple Title Definitions](#simple-title-definitions)
 - [Title & Description](#title--description)
 - [Arguments](#arguments)
 - [Command-Line Options](#command-line-options)
   - [Boolean (Flag) Options](#boolean-flag-options)
   - [Getting `-h` & `--help` For Free](#getting--h----help-for-free)
   - [Passing Options Directly Through to the Command Script](#passing-options-directly-through-to-the-command-script)
 - [Run Tool Help](#run-tool-help)
 - [Using an Alternative Runfile](#using-an-alternative-runfile)
   - [Via Command-Line](#via-command-line)
   - [Via Environment Variables](#via-environment-variables)
     - [`$RUNFILE`](#runfile-1)
       - [Using Direnv](#using-direnv-to-auto-configure-runfile)
     - [`$RUNFILE_ROOTS`](#runfile_roots)
 - [Runfile Variables](#runfile-variables)
   - [Local By Default](#local-by-default)
   - [Exporting Variables](#exporting-variables)
     - [Per-Command Variables](#per-command-variables)
     - [Exporting Previously-Defined Variables](#exporting-previously-defined-variables)
     - [Pre-Declaring Exports](#pre-declaring-exports)
       - [Forgetting To Define An Exported Variable](#forgetting-to-define-an-exported-variable)
   - [Referencing Other Variables](#referencing-other-variables)
   - [Shell Substitution](#shell-substitution)
   - [Conditional Assignment](#conditional-assignment)
 - [Runfile Attributes](#runfile-attributes)
   - [`.SHELL`](#runfile-attributes)
   - [`.RUN`](#runfile-attributes)
   - [`.RUNFILE`](#runfile-attributes)
   - [`.RUNFILE.DIR`](#runfile-attributes)
   - [`.SELF`](#runfile-attributes)
   - [`.SELF.DIR`](#runfile-attributes)
   - [Exporting Attributes](#exporting-attributes)
     - [Simple Export](#simple-export)
     - [Export With Name](#export-with-name)
 - [Assertions](#assertions)
 - [Includes](#includes)
   - [File Globbing](#file-globbing)
   - [Working Directory](#working-directory)
   - [File(s) Not Found](#files-not-found)
   - [Avoiding Include Loops](#avoiding-include-loops)
   - [Overriding Commands](#overriding-commands)
     - [Cannot Re-Register Command In Same Runfile](#cannot-re-register-command-in-same-runfile)
     - [Overrides Are Case-Insensitive](#overrides-are-case-insensitive)
     - [First Registered Command Defines Case For Help](#first-registered-command-defines-case-for-help)
     - [First Registered Command Defines Default Documentation](#first-registered-command-defines-default-documentation)
     - [Commands Are Listed In The Order They Are Registered](#commands-are-listed-in-the-order-they-are-registered)
 - [Invoking Other Commands & Runfiles](#invoking-other-commands--runfiles)
   - [.RUN & .RUNFILE Attributes](#run--runfile-attributes)
 - [Script Shells](#script-shells)
   - [Per-Command Shell Config](#per-command-shell-config)
   - [Global Default Shell Config](#global-default-shell-config)
   - [Other Executors](#other-executors)
     - [Python Example](#python-example)
   - [Custom `#!` Support](#custom--support)
     - [C Example](#c-example)

------------------------------
### Simple Command Definitions

_Runfile_

```
hello:
  echo "Hello, world"
```

We'll see that `hello` shows as an invokable command, but has no other help text.

_list commands_

```
$ run list

Commands:
  list       (builtin) List available commands
  help       (builtin) Show help for a command
  version    (builtin) Show run version
  hello
```

_show help for hello command_
```
$ run help hello

hello: no help available.
```

_invoke hello command_
```
$ run hello

Hello, world
```

#### Naming Commands

Run accepts the following pattern for command names:

```
alpha ::= 'a' .. 'z' | 'A' .. 'Z'
digit ::= '0' .. '9'

CMD_NAME ::= [ alpha | '_' ] ( [ alpha | digit | '_' | '-' ] )*
```

Some examples:
* `hello`
* `hello_world`
* `hello-world`
* `HelloWorld`

##### Case Sensitivity

###### Registering Commands

When registering commands, run treats the command name as case-insensitive and subject to [command override](#overriding-commands) rules.

*case-insensitive override example*

For example, run will generate an error if a command name is defined multiple times in the same runfile, even if the names use different cases:

_Runfile_
```
hello-world:
  echo "Hello, world"

HELLO-WORLD:
  echo "HELLO, WORLD"
```

_list commands_

```
$ run list

run: Runfile: command hello-world defined multiple times in the same file: lines 1 and 4
```

###### Invoking Commands

When invoking commands, run treats the command name as case-insensitive:

_Runfile_
```
Hello-World:
  echo "Hello, world"
```

_output_
```
$ run Hello-World
$ run Hello-world
$ run hello-world

Hello, world
```

###### Displaying Help

When displaying help text, run displays command names as they are originally defined:

_list commands_

```
$ run list

Commands:
  ...
  Hello-World
  ...
```

_show help for Hello-World command_
```
$ run help hello-world

Hello-World: no help available.
```

----------------------------
### Simple Title Definitions

We can add a simple title to our command, providing some help content.

_Runfile_

```
## Hello world example.
hello:
  echo "Hello, world"
```

_output_

```
$ run list

Commands:
  list       (builtin) List available commands
  help       (builtin) Show help for a command
  version    (builtin) Show run version
  hello      Hello world example.
  ...
```

```
$ run help hello

hello:
  Hello world example.
```

-----------------------
### Title & Description

We can further flesh out the help content by adding a description.

_Runfile_

```
##
# Hello world example.
# Prints "Hello, world".
hello:
  echo "Hello, world"
```

_output_

```
$ run list

Commands:
  list       (builtin) List available commands
  help       (builtin) Show help for a command
  version    (builtin) Show run version
  hello      Hello world example.
  ...
```

```
$ run help hello

hello:
  Hello world example.
  Prints "Hello, world".
```

-------------
### Arguments

Positional arguments are passed through to your command script.

_Runfile_

```
##
# Hello world example.
hello:
  echo "Hello, ${1}"
```

_output_

```
$ run hello Newman

Hello, Newman
```

------------------------
### Command-Line Options

You can configure command-line options and access their values with environment variables.

_Runfile_

```
##
# Hello world example.
# Prints "Hello, <name>".
# OPTION NAME -n,--name <name> Name to say hello to
hello:
  echo "Hello, ${NAME}"
```

_output_

```
$ run help hello

hello:
  Hello world example.
  Prints "Hello, <name>".
Options:
  -h, --help
        Show full help screen
  -n, --name <name>
        Name to say hello to
```

```
$ run hello --name=Newman
$ run hello -n Newman

Hello, Newman
```

#### Boolean (Flag) Options

Declare flag options by omitting the `'<...>'` segment.

_Runfile_

```
##
# Hello world example.
# OPTION NEWMAN --newman Say hello to Newman
hello:
  NAME="World"
  [[ -n "${NEWMAN}" ]] && NAME="Newman"
  echo "Hello, ${NAME}"
```

_output_

```
$ run help hello

hello:
  Hello world example.
  ...
  --newman
        Say hello to Newman
```

##### Setting a Flag Option to TRUE

```
$ run help --newman=true # true | True | TRUE
$ run help --newman=1    # 1 | t | T
$ run help --newman      # Empty value = true

Hello, Newman
```

##### Setting a Flag Option to FALSE

```
$ run help --newman=false # false | False | FALSE
$ run help --newman=0     # 0 | f | F
$ run help                # Default value = false

Hello, World

```

#### Getting `-h` & `--help` For Free

If your command defines one or more options, but does not explicitly configure options `-h` or `--help`, then they are automatically registered to display the command's help text.

_Runfile_
```
##
# Hello world example.
# Prints "Hello, world".
hello:
  echo "Hello, world"
```

_output_
```
$ run hello -h
$ run hello --help

hello:
  Hello world example.
  Prints "Hello, world".
```

#### Passing Options Directly Through to the Command Script

If your command does not define any options within the Runfile, then run will pass all command line arguments directly through to the command script.

_Runfile_
```
##
# Echo example
# Prints the arguments passed into the script
#
echo:
  echo script arguments = "${@}"
```

_output_
```
$ run echo -h --help Hello Newman

script arguments = -h --help Hello Newman
```

NOTE: As you likely surmised, help options (`-h` & `--help`) are not automatically registered when the command does not define any other options.

##### What if My Command Script DOES Define Options?

If your command script does define one or more options within the Runfile, you can still pass options directly through to the command script, but the syntax is a bit different:

_Runfile_
```
##
# Echo example
# Prints the arguments passed into the script
# Use -- to separate run options from script options
# OPTION ARG -a <arg> Contrived argument
#
echo:
  echo ARG = "${ARG}"
  echo script arguments = "${@}"
```

_output_
```
$ run echo -a my-arg -- -h --help Hello Newman

ARG = my-arg
script arguments = -h --help Hello Newman
```

Notice the `'--'` in the argument list - Run will stop parsing options when it encounters the `'--'` and pass the rest of the arguments through to the command script.

-----------------
### Run Tool Help

Invoking `-h` or `--help` with no command shows the help page for the run tool itself.

```
$ run --help

Usage:
       run <command> [option ...]
          (run <command>)
  or   run list
          (list commands)
  or   run help <command>
          (show help for <command>)
Options:
  -r, --runfile <file>
        Specify runfile (default='${RUNFILE:-Runfile}')
        ex: run -r /my/runfile list
Note:
  Options accept '-' | '--'
  Values can be given as:
        -o value | -o=value
  Flags (booleans) can be given as:
        -f | -f=true | -f=false
  Short options cannot be combined
```

--------------------------------
### Using an Alternative Runfile

#### Via Command Line

You can specify a runfile using the `-r | --runfile` option:

```
$ run --runfile /path/to/my/Runfile <command>
```

NOTE: When specifying a runfile, the file does **not** have to be named `"Runfile"`.

#### Via Environment Variables

##### $RUNFILE

You can specify a runfile using the `$RUNFILE` environment variable:

```
$ export RUNFILE="/path/to/my/Runfile"

$ run <command>
```

For some other interesting uses of `$RUNFILE`, see:
* [Invoking Other Commands & Runfiles](#invoking-other-commands--runfiles)
* [Using direnv to auto-configure $RUNFILE](#using-direnv-to-auto-configure-runfile)

NOTE: When specifying a runfile, the file does **not** have to be named `"Runfile"`.

##### $RUNFILE_ROOTS

You can instruct run to look _up_ the directory path in search of a runfile.

You do this using the `$RUNFILE_ROOTS` path variable.

* `$RUNFILE_ROOTS` is treated as a list of path entries (using standard os path separator)
* Behaves largely similar to [GIT_CEILING_DIRECTORIES](https://git-scm.com/docs/git#Documentation/git.txt-codeGITCEILINGDIRECTORIEScode)
* If `$PWD` is a child of a root entry, run walks-up looking for `Runfile`
* Roots themselves are generally treated as _exclusive_ (ie not checked)
* `$HOME`, if a configured root, is treated as _inclusive_ (ie it **is** checked)

_general usage_

    export RUNFILE_ROOTS="${HOME}"  # Will walk up to $HOME (inclusively)

_most permissive_

    export RUNFILE_ROOTS="/"  # Will walk up to / (exclusively)

NOTE: `$HOME` is given special treatment to support the case where a project is given its own _user_ account and lives in the _home_ folder of that user.

For the case of creating globally available tasks, see the [Special Modes](#special-modes) section.

---------------------
### Runfile Variables

You can define variables within your runfile:

_Runfile_
```
NAME := "Newman"

##
# Hello world example.
# Tries to print "Hello, ${NAME}"
hello:
  echo "Hello, ${NAME:-world}"
```

#### Local By Default

By default, variables are local to the runfile and are not part of your command's environment.

For example, you can access them within your command's description:

```
$ run help hello

hello:
  Hello world example.
  Tries to print "Hello, Newman"
```

But not within your commands script:

```
$ run hello

Hello, world
```

#### Exporting Variables

To make a variable available to your command script, you need to `export` it:

_Runfile_
```
EXPORT NAME := "Newman"

##
# Hello world example.
# Tries to print "Hello, ${NAME}"
hello:
  echo "Hello, ${NAME:-world}"
```

_output_
```
$ run hello

Hello, Newman
```

##### Per-Command Variables

You can create variables on a per-command basis:

_Runfile_
```
##
# Hello world example.
# Prints "Hello, ${NAME}"
# EXPORT NAME := "world"
hello:
  echo "Hello, ${NAME}"
```

_help output_
```
$ run help hello

hello:
  Hello world example.
  Prints "Hello, world"
```

_command output_

```
$ run hello

Hello, world
```

##### Exporting Previously-Defined Variables

You can export previously-defined variables by name:

_Runfile_
```
HELLO := "Hello"
NAME  := "world"

##
# Hello world example.
# EXPORT HELLO, NAME
hello:
  echo "${HELLO}, ${NAME}"
```

##### Pre-Declaring Exports
You can declare exported variables before they are defined:

_Runfile_
```
EXPORT HELLO, NAME

HELLO := "Hello"
NAME  := "world"

##
# Hello world example.
hello:
  echo "${HELLO}, ${NAME}"
```

###### Forgetting To Define An Exported Variable
If you export a variable, but don't define it, you will get a `WARNING`

_Runfile_
```
EXPORT HELLO, NAME

NAME := "world"

##
# Hello world example.
hello:
  echo "Hello, ${NAME}"
```

_output_
```
$ run hello

run: WARNING: exported variable not defined: 'HELLO'
Hello, world
```

#### Referencing Other Variables

You can reference other variables within your assignment:

_Runfile_
```
SALUTATION := "Hello"
NAME       := "Newman"

EXPORT MESSAGE := "${SALUTATION}, ${NAME}"

##
# Hello world example.
hello:
  echo "${MESSAGE}"
```

#### Shell Substitution

You can invoke sub-shells and capture their output within your assignment:

_Runfile_
```
SALUTATION := "Hello"
NAME       := "$( echo 'Newman )" # Trivial example

EXPORT MESSAGE := "${SALUTATION}, ${NAME}"

##
# Hello world example.
hello:
  echo "${MESSAGE}"
```

#### Conditional Assignment

You can conditionally assign a variable, which only assigns a value if one does not already exist.

_Runfile_
```
EXPORT NAME ?= "world"

##
# Hello world example.
hello:
  echo "Hello, ${NAME}"
```

_example with default_
```
$ run hello

Hello, world
```

_example with override_
```
NAME="Newman" run hello

Hello, Newman
```

----------------------
### Runfile Attributes

Attributes are special variables used by the Run engine.

Their names start with `.` to avoid colliding with [runfile variables](#runfile-variables) and environment variables.<br/>

Following is the list of Run's attributes:

| Attribute      | Description
|----------------|------------
| `.SHELL`       | Contains the shell command that will be used to execute command scripts. See [Script Shells](#script-shells) for more details.
| `.RUN`         | Contains the absolute path of the run binary currently in use. Useful for [Invoking Other Commands & Runfiles](#invoking-other-commands--runfiles).
| `.RUNFILE`     | Contains the absolute path of the **primary** Runfile.
| `.RUNFILE.DIR` | Contains the absolute path of the parent folder of the **primary** runfile.
| `.SELF`        | Contains the absolute path of the **current** (primary or included) runfile.
| `.SELF.DIR`    | Contains the absolute path of the parent folder of the **current** runfile.


#### Exporting Attributes

In order to access an attribute's value within your commands, you'll need to assign them to an [exported variable](#exporting-variables).

Older versions of Run required you to use a variable assignment:

_Runfile_
```
EXPORT RUNFILE     := ${.RUNFILE}
EXPORT RUNFILE_DIR := ${.RUNFILE.DIR}

## Prints the value of .RUNFILE
runfile:
    echo "${RUNFILE}"

## Prints the value of .RUNFILE.DIR
runfile-dir:
    echo "${RUNFILE_DIR}"
```

Newer versions of Run now support less verbose options:

##### Simple Export

You can quickly export an attribute with a default variable name:

_Runfile_
```
EXPORT .RUNFILE, .RUNFILE.DIR

## Prints the value of .RUNFILE
runfile:
    echo "${RUNFILE}"

## Prints the value of .RUNFILE.DIR
runfile-dir:
    echo "${RUNFILE_DIR}"
```

With this technique, Run uses the attribute's name to determine the exported variable's name by:
* Removing the leading `.` character
* Substituting any remaining `.` characters with `_`

##### Export With Name

If you want to export an attribute with a non-default variable name, you can use the `AS` syntax:

```
EXPORT .RUNFILE     AS RF
EXPORT .RUNFILE.DIR AS RFD

## Prints the value of .RUNFILE
runfile:
    echo "${RF}"

## Prints the value of .RUNFILE.DIR
runfile-dir:
    echo "${RFD}"
```

--------------
### Assertions

Assertions let you check against expected conditions, exiting with an error message when checks fail.

Assertions have the following syntax:

```
ASSERT <condition> [ "<error message>" | '<error message>' ]
```

*Note:* The error message is optional and will default to `"assertion failed"` if not provided

#### Condition

The following condition patterns are supported:

* `[  ...  ]`
* `[[ ... ]]`
* `(  ...  )`
* `(( ... ))`

*Note:* Run does not interpret the condition.  The condition text will be executed, unmodified (including surrounding braces/parens/etc), by the configured shell. Run will inspect the exit status of the check and pass/fail the assertion accordingly.

#### Assertion Example

Here's an example that uses both global and command-level assertions:

_Runfile_
```
##
# Not subject to any assertions
world:
	echo Hello, World

# Assertion applies to ALL following commands
ASSERT [ -n "${HELLO}" ] "Variable HELLO not defined"

##
# Subject to HELLO assertion, even though it doesn't use it
newman:
	echo Hello, Newman

##
# Subject to HELLO assertion, and adds another
# ASSERT [ -n "${NAME}" ] 'Variable NAME not defined'
name:
	echo ${HELLO}, ${NAME}
```

_example with no vars_
```
$ run world

Hello, World

$ run newman

run: ERROR: Runfile:7: Variable HELLO not defined

$ run name

run: ERROR: Runfile:7: Variable HELLO not defined
```

_example with HELLO_
```
$ HELLO=Hello run newman

Hello, Newman

$ HELLO=Hello run name

run: ERROR: Runfile:16: Variable NAME not defined
```

_example with HELLO and NAME_
```
$ HELLO=Hello NAME=Everybody run name

Hello, Everybody
```

*Note:* Assertions apply only to commands and are only checked when a command is invoked.  Any globally-defined assertions will apply to ALL commands defined after the assertion.

------------
### Includes

Includes let you organize commands across multiple Runfiles.

Includes have the following syntax:
```
INCLUDE <file pattern> | "<file pattern>" | '<file pattern>'
```

Simple example:

_file layout_
```
Runfile
Runfile-hello
```

_Runfile_
```
INCLUDE Runfile-hello
```

_Runfile-hello_
```
hello:
    echo "Hello from Runfile-hello"
```

_output_
```
$ run hello

Hello from Runfile-hello
```

#### File Globbing

Run utilizes [goreleaser/fileglob](https://github.com/goreleaser/fileglob) in order support file globbing for includes.

According to their README, `fileglob` supports:

* Asterisk wildcards (`*`)
* Super-asterisk wildcards (`**`)
* Single symbol wildcards (`?`)
* Character list matchers with negation and ranges (`[abc]`, `[!abc]`, `[a-c]`)
* Alternative matchers (`{a,b}`)
* Nested globbing (`{a,[bc]}`)
* Escapable wildcards (`\{a\}/\*`)

**Fileglob Example:**

_file layout_
```
Runfile
1/1/Runfile-1
2/2/Runfile-2
3/3/Runfile-3
```

_Runfile_
```
INCLUDE **/Runfile-*
```

#### Working Directory

Include names / glob-patterns are resolved relative to the Primary runfile's containing directory.

#### File(s) Not Found

##### OK For Glob

When using a globbing pattern, Run considers it OK if the pattern results in no files being found.

This makes it possible to support features like an optional Runfile include directory, or the ability to start a project folder with no includes but have them automatically picked up as you add them.

_Runfile_

```
INCLUDE maybe_some_runfiles/Runfile-*  # OK if not no files found
```

##### BAD For Single File

When using a single filename (no globbing), Run considers it an error if the include file is not found.

_Runfile_
```
INCLUDE Runfile-must-exist  # Errors if file not found
```

_output_
```
$ run list

run: include runfile not found: 'Runfile-must-exist'
```

#### Avoiding Include Loops

Run keeps track of already-included runfiles and will silently avoid including the same runfile multiple times.

_Runfile_
```
INCLUDE Runfile-hello
INCLUDE Runfile-hello  # Silently skipped
```

#### Overriding Commands

Run allows you override commands, as long as they were originally registered in a _different_ Runfile.

_Runfile_
```
## defined in Runfile
command1:
  echo command1 from Runfile

INCLUDE Runfile-include

## defined in Runfile
command2:
  echo command2 from Runfile
```

_Runfile-include_
```
## defined in Runfile-include
command1:
  echo command1 from Runfile-include

## defined in Runfile-include
command2:
  echo command2 from Runfile-include
```

_list commands_
```
$ run list

Commands:
  ...
  command1    defined in Runfile-include
  command2    defined in Runfile
```

Notice that the _included_ runfile overrides `command1`, but the _primary_ runfile overrides `command2`.

##### Cannot Re-Register Command In Same Runfile

Run will error when attempting to register a command multiple times within the _same_ Runfile:

_Runfile_
```
hello-world:
  echo "Hello, world"

hello-world:
  echo "Hello, world"
```

_list commands_
```
$ run list

run: Runfile: command hello-world defined multiple times in the same file: lines 1 and 4
```

##### Overrides Are Case-Insensitive

Run's override matching is case-insensitive:

_Runfile_
```
## defined in Runfile
command1:
  echo command1 from Runfile

include Runfile-include
```

_Runfile-include_
```
## defined in Runfile-include
COMMAND1:
  echo command1 from Runfile-include
```

_list commands_
```
$ run list

Commands:
  ...
  command1    defined in Runfile-include
```

Notice that `COMMAND1` from the _included_ runfile overrides `command1` from the _primary_ runfile.

###### First Registered Command Defines Case For Help

Run keeps track of the original case used when a command is first registered, and uses it when displaying help:

_Runfile_
```
## defined in Runfile
COMMAND1:
  echo command1 from Runfile

include Runfile-include
```

_Runfile-include_
```
## defined in Runfile-include
command1:
  echo command1 from Runfile-include
```

_list commands_
```
$ run list

Commands:
  ...
  COMMAND1    defined in Runfile-include
```

Notice that the displayed name comes from the original registration in the _primary_ runfile.

##### First Registered Command Defines Default Documentation

Run keeps track of the title & description when a command is first registered, and uses it if an overriding command does not define its own documentation:

_Runfile_
```
## title defined in Runfile
command1:
  echo command1 from Runfile

include Runfile-include
```

_Runfile-include_
```
command1:
  echo command1 from Runfile-include
```

_list commands_
```
$ run list

Commands:
  ...
  commmand1    title defined in Runfile
```

_command output_
```
$ run command1

command1 from Runfile-include
```

Notice that, even though `command1` from the _included_ runfile was invoked, the displayed title comes from the original registration in the _primary_ runfile.

##### Commands Are Listed In The Order They Are Registered

Run keeps track of the _order_ in which commands are registered, and maintains that order even if a command is later overridden:

_Runfile_
```
## defined in Runfile
command1:
  echo command1 from Runfile

## defined in Runfile
command2:
  echo command2 from Runfile

## defined in Runfile
command3:
  echo command3 from Runfile

include Runfile-include
```

_Runfile-include_
```
## defined in Runfile-include
command2:
  echo command2 from Runfile-include
```

_list commands_
```
$ run list

Commands:
  ...
  command1    defined in Runfile
  command2    defined in Runfile-include
  command3    defined in Runfile
```

Notice that `command2` is still shown _between_ `command1` and `command3`, matching the order in which it was originally registered.

--------------------------------------
### Invoking Other Commands & Runfiles

#### .RUN / .RUNFILE Attributes
Run exposes the following attributes:

* `.RUN` - Absolute path of the run binary currently in use
* `.RUNFILE` - Absolute path of the current **primary** Runfile

NOTE: Even from inside an [included](#includes) Runfile, `.RUNFILE` will always reference the primary Runfile

Your command script can use these to invoke other commands:

_Runfile_
```
##
# Invokes hello
# EXPORT RUN := ${.RUN}
# EXPORT RUNFILE := ${.RUNFILE}
test:
    "${RUN}" hello

hello:
    echo "Hello, World"
```

_output_
```
$ run test

Hello, World
```

-----------------
### Script Shells

Run's default shell is `'sh'`, but you can specify other shells.

All the standard shells should work.

#### Per-Command Shell Config

Each command can specify its own shell:
```
##
# Hello world example.
# NOTE: Requires ${.SHELL}
hello (bash):
  echo "Hello, world"
```

#### Global Default Shell Config

You can set the default shell for the entire runfile:

_Runfile_
```
# Set default shell for all actions
.SHELL = bash

##
# Hello world example.
# NOTE: Requires ${.SHELL}
hello:
  echo "Hello, world"
```

#### Other Executors

You can even specify executors that are not technically shells.

##### Python Example

_Runfile_
```
## Hello world python example.
hello (python):
	print("Hello, world from python!")
```

##### Script Execution : env

Run executes scripts using the following command:

```
/usr/bin/env $SHELL $TMP_SCRIPT_FILE [ARG ...]
```

Any executor that is on the `PATH`, can be invoked via `env`, and takes a filename as its first argument should work.

#### Custom `#!` Support

Run allows you to define custom `#!` lines in your command script:

##### C Example

Here's an example of running a `c` program from a shell script using a custom `#!` header:

_Runfile_
```
##
# Hello world c example using #! executor.
# NOTE: Requires gcc
hello:
  #!/usr/bin/env sh
  sed -n -e '7,$p' < "$0" | gcc -x c -o "$0.$$.out" -
  $0.$$.out "$0" "$@"
  STATUS=$?
  rm $0.$$.out
  exit $STATUS
  #include <stdio.h>

  int main(int argc, char **argv)
  {
    printf("Hello, world from c!\n");
    return 0;
  }
```

##### Script Execution: Direct

*NOTE:* The `#!` executor does not use `/user/bin/env` to invoke your script.  Instead, it attempts to make the temporary script file executable then invoke it directly.


-----------------
### Misc Features

#### Ignoring Script Lines

You can use a `#` on the first column of a command script to ignore a line:

_Runfile_
```
hello:
    # This comment WILL be present in the executed command script
    echo "Hello, Newman"
# This comment block WILL NOT be present in the executed command script
#   echo "Hello, World"
    echo "Goodbye, now"
```

*Note:* Run detects and skips these comment lines when parsing the runfile, so the `#` will work regardless of what language the script text is written in (i.e even if the target language doesn't support `#` for comments).


----------------
## Special Modes

### Shebang Mode

In `shebang mode`, you make your runfile executable and invoke commands directly through it:

_runfile.sh_
```
#!/usr/bin/env run shebang

## Hello example using shebang mode
hello:
  echo "Hello, world"

```

_output_
```
$ chmod +x runfile.sh
$ ./runfile.sh hello

Hello, world

```

#### Filename used in help text
In shebang mode, the runfile filename replaces references to the `run` command:

_shebang mode help example_
```
$ ./runfile.sh help

Usage:
       runfile.sh <command> [option ...]
                 (run <command>)
  or   runfile.sh list
                 (list commands)
  or   runfile.sh help <command>
                 (show help for <command>)
  ...
```

_shebang mode list example_

```
$ ./runfile.sh list

Commands:
  list           (builtin) List available commands
  help           (builtin) Show help for a command
  run-version    (builtin) Show run version
  hello          Hello example using shebang mode
```

#### Version command name

In shebang mode, the `version` command is renamed to `run-version`.  This enables you to create your own `version` command, while still providing access to run's version info, if needed.

_runfile.sh_

```
#!/usr/bin/env run shebang

## Show runfile.sh version
version:
    echo "runfile.sh v1.2.3"

## Hello example using shebang mode
hello:
  echo "Hello, world"
```

_shebang mode version example_
```
$ ./runfile.sh list
  ...
  run-version    (builtin) Show Run version
  version        Show runfile.sh version
  ...

$ ./runfile.sh version

runfile.sh v1.2.3

$ ./runfile.sh run-version

runfile.sh is powered by run v0.0.0. learn more at https://github.com/TekWizely/run
```

-------------
### Main Mode

In main mode you use an executable runfile that consists of a single command, aptly named `main`:

_runfile.sh_
```
#!/usr/bin/env run shebang

## Hello example using main mode
main:
  echo "Hello, world"
```

In this mode, run's built-in commands are disabled and the `main` command is invoked directly:

_output_
```
$ ./runfile.sh

Hello, world
```

#### Filename used in help text
In main mode, the runfile filename replaces references to `command` name:

_main mode help example_
```
$ ./runfile.sh --help

runfile.sh:
  Hello example using main mode

```

#### Help options

In main mode, help options (`-h` & `--help`) are automatically configured, even if no other options are defined.

This means you will need to use `--` in order to pass options through to the main script.

------------------------------------------
## Using direnv to auto-configure $RUNFILE

A nice hack to make executing run tasks within your project more convenient is to use [direnv](https://direnv.net/) to auto-configure the `$RUNFILE` environment variable:

_create + edit + activate rc file_
```
$ cd ~/my-project
$ direnv edit .
```

_edit .envrc_
```
export RUNFILE="${PWD}/Runfile"
```

Save & exit.  This will activate _immediately_ but will also activate whenever you `cd` into your project's root folder.

```
$ cd ~/my-project

direnv: export +RUNFILE
```

_verify_
```
$ echo $RUNFILE

/home/user/my-project/Runfile
```

With this, you can execute `run <cmd>` from anywhere in your project.

-------------
## Installing

### Via Bingo

[Bingo](https://github.com/TekWizely/bingo) makes it easy to install (and update) golang apps directly from source:

_install_
```
$ bingo install github.com/TekWizely/run
```

_update_
```
$ bingo update run
```

### Pre-Compiled Binaries

See the [Releases](https://github.com/TekWizely/run/releases) page as recent releases are accompanied by pre-compiled binaries for various platforms.

##### Not Seeing Binaries For Your Platform?

Run currently uses [goreleaser](https://goreleaser.com/) to generate release assets.

Feel free to [open an issue](https://github.com/TekWizely/run/issues/new) to discuss additional target platforms, or even create a PR against the [.goreleaser.yml](https://github.com/TekWizely/run/blob/master/.goreleaser.yml) configuration.

### Brew

#### Brew Core

Run is now available on homebrew core:

* https://formulae.brew.sh/formula/run

_install run via brew core_
```
$ brew install run
```

#### Brew Tap

In addition to being available in brew core, I have also created a tap to ensure the latest version is always available:

* https://github.com/TekWizely/homebrew-tap

_install run directly from tap_
```
$ brew install tekwizely/tap/run
```

_install tap to track updates_
```
$ brew tap tekwizely/tap

$ brew install run
```

#### NIX

For Nix users, a package is available on nixpkgs:

* [Nix package for Run](https://search.nixos.org/packages?show=run&from=0&size=1&sort=relevance&type=packages&query=run)

Supported Platforms:
* x86_64-darwin
* aarch64-darwin
* aarch64-linux
* i686-linux
* x86_64-linux

_install run on NixOS_
```bash
$ nix-env -iA nixos.run
```

_install run on non-NixOs_
```bash
$ nix-env -iA nixpkgs.run
```

#### AUR

For Archlinux users, a package is available on the AUR:

* https://aur.archlinux.org/packages/run-git

_install run from AUR using yay_

```bash
$ yay -S run-git
```
#### NPM / Yarn

NPM & Yarn users can install run via the `@tekwizely/run` package:

* [Run NPM Package Page](https://www.npmjs.com/package/@tekwizely/run)

```
$ npm i '@tekwizely/run'

$ yarn add '@tekwizely/run'
```

### Other Package Managers
I hope to have other packages available soon and will update the README as they become available.

---------------
## Contributing

To contribute to Run, follow these steps:

1. Fork this repository.
2. Create a branch: `git checkout -b <branch_name>`.
3. Make your changes and commit them: `git commit -m '<commit_message>'`
4. Push to the original branch: `git push origin <project_name>/<location>`
5. Create the pull request.

Alternatively see the GitHub documentation on [creating a pull request](https://help.github.com/en/github/collaborating-with-issues-and-pull-requests/creating-a-pull-request).

----------
## Contact

If you want to contact me you can reach me at TekWize.ly@gmail.com.

----------
## License

The `tekwizely/run` project is released under the [MIT](https://opensource.org/licenses/MIT) License.  See `LICENSE` file.

-------------------------------------
## Just Looking for Bash Arg Parsing?

If you happened to find this project on your quest for bash-specific arg parsing solutions, I found this fantastic S/O post with many great suggestions:

* [Parsing Command-Line Arguments in Bash (S/O)](https://stackoverflow.com/questions/192249/how-do-i-parse-command-line-arguments-in-bash)

---------------
## Contributors ✨

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/docs/en/emoji-key)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tr>
    <td align="center"><a href="http://chabad360.me"><img src="https://avatars2.githubusercontent.com/u/1668291?v=4?s=100" width="100px;" alt=""/><br /><sub><b>chabad360</b></sub></a><br /><a href="https://github.com/TekWizely/run/commits?author=chabad360" title="Documentation">📖</a> <a href="#infra-chabad360" title="Infrastructure (Hosting, Build-Tools, etc)">🚇</a> <a href="https://github.com/TekWizely/run/issues?q=author%3Achabad360" title="Bug reports">🐛</a></td>
    <td align="center"><a href="https://github.com/dawidd6"><img src="https://avatars0.githubusercontent.com/u/9713907?v=4?s=100" width="100px;" alt=""/><br /><sub><b>Dawid Dziurla</b></sub></a><br /><a href="#infra-dawidd6" title="Infrastructure (Hosting, Build-Tools, etc)">🚇</a></td>
    <td align="center"><a href="https://github.com/rwhogg"><img src="https://avatars3.githubusercontent.com/u/2373856?v=4?s=100" width="100px;" alt=""/><br /><sub><b>Bob "Wombat" Hogg</b></sub></a><br /><a href="https://github.com/TekWizely/run/commits?author=rwhogg" title="Documentation">📖</a></td>
    <td align="center"><a href="https://github.com/Gys"><img src="https://avatars0.githubusercontent.com/u/943251?v=4?s=100" width="100px;" alt=""/><br /><sub><b>Gys</b></sub></a><br /><a href="https://github.com/TekWizely/run/issues?q=author%3AGys" title="Bug reports">🐛</a></td>
    <td align="center"><a href="https://crimson.no"><img src="https://avatars.githubusercontent.com/u/125863?v=4?s=100" width="100px;" alt=""/><br /><sub><b>Robin Burchell</b></sub></a><br /><a href="https://github.com/TekWizely/run/commits?author=rburchell" title="Code">💻</a></td>
  </tr>
</table>

<!-- markdownlint-restore -->
<!-- prettier-ignore-end -->

<!-- ALL-CONTRIBUTORS-LIST:END -->

This project follows the [all-contributors](https://github.com/all-contributors/all-contributors) specification. Contributions of any kind welcome!
