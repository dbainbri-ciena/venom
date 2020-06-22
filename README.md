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
| `long` or `l` | `long:"field_name"` | struct member name, broken based on CamelCase, separated with `_`, and lower cased | the long flag name used to set the configuration option |
| `short` or `s` | `short:"c"` | none | the character used for the short flag to set the configuraiton option |
| `default` or `d` | `default:"5s"` | zero value | the default value for the argument represented as a string |
| `env` or `e` | `env:"FIELD_NAME"` | struct member name, broken based on CamelCase, separated with `_`, and upper cased | the environment variable used to set the configuration option |
| `help` or `h` | `help:"help message"` | none | the help message to display for the command argument |
| `ignored` | `ignored:"true"` | false | if true will not establish configuration for the struct member |
