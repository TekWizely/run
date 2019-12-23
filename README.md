# Run: Easily manage and invoke small scripts and wrappers

<!--- These are examples. See https://shields.io for others or to customize this set of shields. You might want to include dependencies, project status and licence info here --->
![GitHub repo size](https://img.shields.io/github/repo-size/TekWizely/run)
![GitHub contributors](https://img.shields.io/github/contributors/TekWizely/run)
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
  list     (builtin) List available commands
  help     (builtin) Show Help for a command
  hello
  Usage:
         run [-r runfile] help <command>
            (show help for <command>)
    or   run [-r runfile] <command> [option ...]
            (run <command>)
```

_show help for hello command_
```
$ run help hello

hello: No help available.
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

When displaying help text, run treats the command name as case-sensitive, displaying the command name as it is defined:

_list commands_

```
$ run list

Commands:
  ..
  Hello-World
  ...
```

_show help for Hello-World command_
```
$ run help Hello-World

Hello-World: No help available.
```

###### Duplicate Command Names

When registering commands, run treats the command name as case-insensitive, generating an error if a command name is defined multiple times:

_Runfile_
```
hello-world:
  echo "Hello, world"

Hello-World:
  echo "Hello, world"
```

_list commands_

```
$ run list

panic: Duplicate command: hello-world
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
  list     (builtin) List available commands
  help     (builtin) Show Help for a command
  hello    Hello world example.
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
  list     (builtin) List available commands
  help     (builtin) Show Help for a command
  hello    Hello world example.
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
$ run echo -a myarg -- -h --help Hello Newman

ARG = myarg
script arguments = -h --help Hello Newman
```

Notice the `'--'` in the argument list - Run will stop parsing options when it encounters the `'--'` and pass the rest of the arguments through to the command script.

-----------------
### Run Tool Help

Invoking the `help` command with no other arguments shows the help page for the run tool itself.

```
$ run help

Usage:
       run -h | --help
          (show help)
  or   run [-r runfile] list
          (list commands)
  or   run [-r runfile] help <command>
          (show help for <command>)
  or   run [-r runfile] <command> [option ...]
          (run <command>)
Options:
  -h, --help
        Show help screen
  -r, --runfile <file>
        Specify runfile (default='Runfile')
Note:
  Options accept '-' | '--'
  Values can be given as:
        -o value | -o=value
  Flags (booleans) can be given as:
        -f | -f=true | -f=false
  Short options cannot be combined
```

------------------------------------
### Using an Alternative Runfile

You can specify a runfile using the `-r | --runfile` option:

```
$ run --runfile /path/to/my/file <command>
```

When specifying a runfile, the file does **not** have to be named `"Runfile"`.

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

run: Warning: exported variable not defined:  HELLO
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

-----------------
### Script Shells

Run's default shell is `'sh'`, but you can specify other shells.

All of the standard shells should work.

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

*NOTE:* The `#!` executor does not use `/user/bin/env` to invoke your script.  Instead it attempts to make the temporary script file executable then invoke it directly.

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
       runfile.sh -h | --help
                 (show help)
  or   runfile.sh list
                 (list commands)
  or   runfile.sh help <command>
                 (show help for <command>)
  or   runfile.sh <command> [option ...]
                 (run <command>)
  ...
```

_shebang mode list example_

```
$ ./runfile.sh list

Commands:
  list     (builtin) List available commands
  help     (builtin) Show Help for a command
  hello    Hello example using shebang mode
Usage:
       runfile.sh help <command>
                 (show help for <command>)
  or   runfile.sh <command> [option ...]
                 (run <command>)
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

-------------
## Installing

### Go Get

```
$ GOPATH=/go/path/ go get github.com/tekwizely/run

$ /go/path/bin/run help
```

### Pre-Compiled Binaries

See the [Releases](https://github.com/TekWizely/run/releases) page as some releases may be accompanied by pre-compiled binaries for various platforms.

### Brew

A brew core formula is in the works, but in the meantime, I have created a tap tha can be used to install run:

* [TekWizely Homebrew Tap](https://github.com/TekWizely/homebrew-tap)


_install run directly_
```
$ brew install tekwizely/tap/run
```

_install tap to track updates_
```
$ brew tap tekwizely/tap

$ brew install run
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
