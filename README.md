# Venom - Declarative golang configuration using Viper and Pflag

## Background
The _Viper_ and _Pflag_ projects provide a diverse and fairly complete set of
capabilities to provide application configuration. Unfortunately, providing the
boilerplate code to set up the application configuration, especially in the
simply cases, can be  daunting.

The _Venom_ project was conceived to provide a straight forward, declarative
mechanism to specify application configuration and a function to convert that
declarative specification to _Viper_ and _Pflag_ configuration options at
runtime.

## Declarative Specification
The _Venom_ project leverages _GoLang_'s structure tag capability to allow
application configuration to be declared as tags against a structure that is
used to capature the application configuration options. The design of the
_Venom_ is such that a minimal set of structure tags should be required and that
in most cases only a `default` value may need to be specified as in the
following example.`

```golang
type MyConfiguration struct {
        Verbose        bool
        LogLevel       string        `default:"warn"`
        RequestTimeout time.Duration `default:"5s"`
}
```

### Available Structure Tags
When processing structure with tags the following tags are available. Any
structure field that is not public, begins with a lower case letter, will be skipped and not processed.

| TAG | EXAMPLE | DEFAULT | DESCRIPTION |
| --- | --- | --- | --- |
| `long` or `l` | `long:"field-name"` | struct member name, broken based on CamelCase, separated, and lower cased | the long flag name used to set the configuration option |
| `short` or `s` | `short:"c"` | none | the character used for the short flag to set the configuraiton option |
| `default` or `d` | `default:"5s"` | zero value | the default value for the argument represented as a string |
| `env` or `e` | `env:"FIELD_NAME"` | struct member name, broken based on CamelCase, separated, and upper cased | the environment variable used to set the configuration option |
| `help` or `h` | `help:"help message"` | none | the help message to display for the command argument |
| `ignored` | `ignored:"true"` | false | if true will not establish configuration for the struct member |

### Processing Options
The following structure is used to customize the processing of structure tags
```
type ProcessingOptions struct {
    Flags         Flags
    LongSeparator string
    EnvSeparator  string
}
```

The `Flags` field is used to determine if the tag parser should generate
bindings to environment variables, `WithEnv`, and/or flags, `WithFlag`.

The separator used when generating environment variables and long flags
names can be customized using the `EnvSeparator` and `LongSeparator`
fields.

A "sane" default for processing options is defined for use and is set to
```
var DefaultOptions = ProcessingOptions{
    Flags:         WithEnv | WithFlag,
    LongSeparator: "-",
    EnvSeparator:  "_",
}
```

### Example
It is important to note that this utility does not try to obfiscate the
underlying packages and is meant as a utility to build the underlying
structures needed by those packages. As such the code that uses this 
module will also need to use viper/pflag calls to retrieve the command
line options.

```
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dbainbri-ciena/venom"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config defines a configuration structure with tags to define defaults
// and help strings
type Config struct {
	Time          time.Duration `default:"5s" help:"pick a time"`
	SomeString    string        `default:"hello" help:"some message"`
	MySpecialBool bool          `default:"true" help:"a bool defaulted to true"`
	MyNormalBool  bool          `default:"false" help:"a bool defaulted to false"`
}

func main() {
	var err error

	// Build a pflags struct based on the config structure and its tags.
	// The environment variables will be prefixed with "VOLTHA" and the
	// default options are being used.
	flags, err := venom.NewConfiguration(&Config{}, "VOLTHA",
		venom.DefaultOptions, os.Args)
	if err != nil {
		panic(err)
	}

	// Depending if there are command line argument pass those arguments
	// to the pflags module to parse
	if len(os.Args) > 1 {
		err = flags.Parse(os.Args[1:])
	} else {
		err = flags.Parse([]string{})
	}
	if err != nil {
		// If err, (not h
		if err == pflag.ErrHelp {
			// Help request, exit
			return
		}
		panic(err)
	}

	// Unmarshal the config options into a structure for
	// ease of access
	config := Config{}
	if err := viper.Unmarshal(&config); err != nil {
		panic(err)
	}

	// Display the configuration as JSON
	if bytes, err := json.MarshalIndent(&config, "", "  "); err != nil {
		panic(err)
	} else {
		fmt.Println(string(bytes))
	}
}
```
