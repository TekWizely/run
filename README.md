# Run

<!--- These are examples. See https://shields.io for others or to customize this set of shields. You might want to include dependencies, project status and licence info here --->
![GitHub repo size](https://img.shields.io/github/repo-size/TekWizely/run)
![GitHub contributors](https://img.shields.io/github/contributors/TekWizely/run)
![GitHub stars](https://img.shields.io/github/stars/TekWizely/run?style=social)
![GitHub forks](https://img.shields.io/github/forks/TekWizely/run?style=social)
![Twitter Follow](https://img.shields.io/twitter/follow/TekWizely?style=social)

Like to use `make` to manage your scripts? Here's a dedicated tool for that!

Run aims to be a better tool for managing multiple scripts within a single file.

## Runfile
Where make has the ubiquitous Makefile, run has the cleverly-named `"Runfile"`

By default, run will look for a file named `"Runfile"` in the current directory, exiting with error if not found.

Read below for details on specifying alternative runfiles, as well as other special modes you might find useful.

-----------
## Examples

### Simple Command Definitions

Here's a simple `hello world` example.

_Runfile_

```
hello (bash):
  echo "Hello, world"
```

We'll see that `hello` shows as an invokable command, but has no other help text.

_list commands_

```bash
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
```bash
$ run help hello

hello (bash): No help available.
```

_invoke hello command_
```bash
$ run hello

Hello, world
```

----------------------------
### Simple Title Definitions

We can add a simple title to our command, providing some help content.

_Runfile_

```
## Hello world example.
hello (bash):
  echo "Hello, world"
```

_output_

```bash
$ run list

Commands:
  list     (builtin) List available commands
  help     (builtin) Show Help for a command
  hello    Hello world example.
  ...
```

```bash
$ run help hello

hello (bash):
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
# NOTE: Requires bash.
hello (bash):
  echo "Hello, world"
```

_output_

```bash
$ run list

Commands:
  list     (builtin) List available commands
  help     (builtin) Show Help for a command
  hello    Hello world example.
  ...
```

```bash
$ run help hello

hello (bash):
  Hello world example.
  Prints "Hello, world".
  NOTE: Requires bash.
```

-------------
### Arguments

Positional arguments are passed through to your command script.

_Runfile_

```
##
# Hello world example.
hello (bash):
  echo "Hello, ${1}"
```

_output_

```bash
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
hello (bash):
  echo "Hello, ${NAME}"
```

_output_

```bash
$ run help hello

hello (bash):
  Hello world example.
  Prints "Hello, <name>".
Options:
  -h, --help
        Show full help screen
  -n, --name <name>
        Name to say hello to
```

```bash
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
hello (bash):
  NAME="World"
  [[ -n "${NEWMAN}" ]] && NAME="Newman"
  echo "Hello, ${NAME}"
```

_output_

```bash
$ run help hello

hello (bash):
  Hello world example.
  ...
  --newman
        Say hello to Newman
```

##### Setting a Flag Option to TRUE

```bash
$ run help --newman=true # true | True | TRUE
$ run help --newman=1    # 1 | t | T
$ run help --newman      # Empty value = true

Hello, Newman
```

##### Setting a Flag Option to FALSE

```bash
$ run help --newman=false # false | False | FALSE
$ run help --newman=0     # 0 | f | F
$ run help                # Default value = false

Hello, World

```

#### Getting `-h` & `--help` For Free
If your command does not explicitly configure options `-h` or `--help`, then they are automatically registered to display the command's help text.

```bash
$ run hello --help

hello (bash):
  ...
```

-----------------
### Run Tool Help

Invoking the `help` command with no other arguments shows the help page for the run tool itself.

```bash
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

```bash
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
hello (bash):
  echo "Hello, ${NAME:-world}"
```

#### Local By Default
By default, variables are local to the runfile and are not part of your command's environment.

For example, you can access them within your command's description:

```bash
$ run hello -h

hello (bash):
  Hello world example.
  Tries to print "Hello, Newman"
```

But not within your commands script:

```bash
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
hello (bash):
  echo "Hello, ${NAME:-world}"
```

_output_
```bash
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
hello (bash):
  echo "Hello, ${NAME}"
```

_help output_
```bash
$ run hello -h

hello (bash):
  Hello world example.
  Prints "Hello, world"
```

_command output_

```bash
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
hello (bash):
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
hello (bash):
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
hello (bash):
  echo "Hello, ${NAME}"
```

_output_
```bash
$ run hello

run: Warning: exported variable not defined:  HELLO
Hello, world
```

#### Conditional Assignment

You can conditionally assign a variable, which only assigns a value if one does not already exist.


_Runfile_
```
EXPORT NAME ?= "world"

##
# Hello world example.
hello (bash):
  echo "Hello, ${NAME}"
```

_example with default_
```bash
$ run hello

Hello, world
```

_example with override_
```bash
NAME="Newman" run hello

Hello, Newman
```

## Script Targets
All of the examples use `bash` as the script target, but you can use other targets

### Python Example

_Runfile_
```
## Hello world python example
hello (python):
	print("Hello, world")
```

_output_
```bash
$ run list

Commands:
  ...
  hello    Hello world python example
  ...
```
```bash
$ run hello -h

hello (python):
  Hello world python example
```
```bash
$ run hello

Hello, world
```

### Other Targets
Run executes scripts using the following pattern:

```
/usr/bin/env $TARGET $TMP_SCRIPT_FILE [ARG ...]
```

Any target that that is on the `PATH`, can be invoked via `env`, and takes a filename as its first argument should work.

### Custom `#!` Support
If you want a custom `#!` in your script, you can use the `shebang` target.
Here's an example of running a `c` program as a shell script:

_Runfile_
```
##
# Hello example using c with shebang target
# NOTE: Requires bash + gcc
hello(shebang):
  #!/usr/bin/env bash
  sed -n -e '7,$p' < "$0" | gcc -x c -o "$0.$$.out" -
  $0.$$.out "$0" "$@"
  STATUS=$?
  rm $0.$$.out
  exit $STATUS
  #include <stdio.h>

  int main(int argc, char **argv)
  {
    printf("Hello, world\n");
    return 0;
  }
```

_output_
```bash
$ run hello -h

hello (shebang):
  Hello example using c with shebang target
  NOTE: Requires bash + gcc
```

```bash
$ run hello

Hello, world
```

## Special Modes

### Shebang Mode

In `shebang mode`, you make your runfile executable and invoke commands directly through it:

_runfile.sh_
```
#!/usr/bin/env run shebang

## Hello example using shebang mode
hello(bash):
  echo "Hello, world"

```

_output_
```bash
$ chmod +x runfile.sh
$ ./runfile.sh hello

Hello, world

```

#### Filename used in help text
In shebang mode, the runfile filename replaces references to the `run` command:

_shebang mode help example_
```bash
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

```bash
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

### Main Mode

In main mode you use an executable runfile that consists of a single command, aptly named `main`:

_runfile.sh_
```
#!/usr/bin/env run shebang

## Hello example using main mode
main(bash):
  echo "Hello, world"
```

In this mode, run's built-in commands are disabled and the `main` command is invoked directly:

_output_
```bash
$ ./runfile.sh

Hello, world
```

#### Filename used in help text
In main mode, the runfile filename replaces references to `command` name:

_main mode help example_
```bash
$ ./runfile.sh --help

runfile.sh (bash):
  Hello example using main mode

```

## Installing

### Go Get

```bash
$ GOPATH=/go/path/ go get github.com/tekwizely/run

$ /go/path/bin/run help
```

### Pre-Compiled Binaries

See the [Releases](https://github.com/TekWizely/run/releases) page as some releases may be accompanied by pre-compiled binaries for various platforms.

### Package Managers
I hope to have `brew`, `deb` and other packages available soon.

## Contributing
To contribute to Run, follow these steps:

1. Fork this repository.
2. Create a branch: `git checkout -b <branch_name>`.
3. Make your changes and commit them: `git commit -m '<commit_message>'`
4. Push to the original branch: `git push origin <project_name>/<location>`
5. Create the pull request.

Alternatively see the GitHub documentation on [creating a pull request](https://help.github.com/en/github/collaborating-with-issues-and-pull-requests/creating-a-pull-request).

## Contact

If you want to contact me you can reach me at TekWize.ly@gmail.com.


## License

The `tekwizely/run` project is released under the [MIT](https://opensource.org/licenses/MIT) License.  See `LICENSE` file.
