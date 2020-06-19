/* Copyright 2020 Ciena Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package venom provides a utility to parse structure tags and convert them
// into Viper (https://github.com/spf13/viper) and Pflag
// (https://github.com/spf13/pflag) configurations that can be used to
// configure your application
package venom

import (
	"errors"
	"fmt"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ErrSpecificationType returned when the interface passed for processing
// not a pointer to a struct
var ErrSpecificationType = errors.New("configuration specification must be a pointer to a struct")

// ProcessingOptions provides mechanism to customizer how tags are converted info configuraiton
// settings
type ProcessingOptions uint32

// Defines process options for the structure tag parser
const (

	// NpProcessingOptions represents the zero value (i.e that no options are set)
	NoProcessingOptions ProcessingOptions = 0x0

	// WithEnv specifies that the parser should automatically generate a environment variable for options
	WithEnv = 0x1

	// WithFlag specifies that the parser should automatically generate a pflag for options
	WithFlag = 0x2

	// DefaultProcessingOptions  represents a useful set of default options for the parser
	DefaultProcessingOptions = WithEnv | WithFlag
)

var gatherRegexp = regexp.MustCompile("([^A-Z0-9]+|[A-Z0-9]+[^A-Z0-9]+|[A-Z0-9]+)")
var acronymRegexp = regexp.MustCompile("([A-Z0-9]+)([A-Z0-9][^A-Z0-9]+)")

// isTrue - attempts to parse the given value as a boolean and return the result. If
// the value does not parse as a boolean it is considered false.
func isTrue(value string) bool {
	val, err := strconv.ParseBool(value)
	if err == nil {
		return val
	}
	return false
}

// splitIntoWords separates the given value string into "words" separated
// bu the specified separation character. Separation is accomplished attempting
// to follow CamelCase format.
func splitIntoWords(value, sep string) string {
	words := gatherRegexp.FindAllStringSubmatch(value, -1)
	if len(words) == 0 {
		return value
	}

	var parts []string
	for _, words := range words {
		if m := acronymRegexp.FindStringSubmatch(words[0]); len(m) == 3 {
			parts = append(parts, m[1], m[2])
		} else {
			parts = append(parts, words[0])
		}
	}

	return strings.Join(parts, sep)
}

// AddConfiguration parses the struct tags associated withe configSpecification
// adding flags to the specified flagset as well as setting up environment
// variable configurations options based on the specified processing options.
func AddConfiguration(flagSet *pflag.FlagSet, configSpecification interface{}, prefix string, options ProcessingOptions, args []string) error {
	spec := reflect.ValueOf(configSpecification)

	if spec.Kind() != reflect.Ptr || spec.Elem().Kind() != reflect.Struct {
		return ErrSpecificationType
	}

	specElem := spec.Elem()
	specType := specElem.Type()

	for i := 0; i < specType.NumField(); i++ {
		field := specElem.Field(i)
		fieldType := specType.Field(i)
		fmt.Printf("Processing field '%s'\n", fieldType.Name)

		// If the field should not be processed, either implicitly or explicitly, then skip
		if !field.CanSet() || isTrue(fieldType.Tag.Get("ignored")) {
			continue
		}

		splitName := splitIntoWords(fieldType.Name, "_")

		// If an option for an environment variable configuration was set then process
		envVar := fieldType.Tag.Get("env")
		if envVar == "" {
			envVar = fieldType.Tag.Get("e")
		}
		if envVar != "" || options&WithEnv != 0 {
			if envVar == "" {
				envVar = strings.ToUpper(splitName)
			}
		}
		if envVar != "" && !strings.HasPrefix(envVar, prefix) {
			envVar = strings.ToUpper(fmt.Sprintf("%s_%s", prefix, envVar))
		}
		fmt.Printf("    ENV: '%s'\n", envVar)

		// Check for default value specification and if not specified then
		// use the types zero value
		defaultAsString := fieldType.Tag.Get("default")
		if defaultAsString == "" {
			defaultAsString = fieldType.Tag.Get("d")
		}
		var defaultValue interface{}
		var err error
		fmt.Printf("    DEFAULT (as string): '%v'\n", defaultAsString)

		if defaultAsString == "" {
			defaultValue = reflect.Zero(field.Type()).Interface()
		}
		fmt.Printf("    DEFAULT (as iface): '%v'\n", defaultValue)

		longFlag := fieldType.Tag.Get("long")
		if longFlag == "" {
			longFlag = fieldType.Tag.Get("l")
		}
		if longFlag != "" || options&WithFlag != 0 {
			if longFlag == "" {
				longFlag = strings.ToLower(splitName)
			}
		}
		fmt.Printf("    LONG: '%s'\n", longFlag)

		shortFlag := fieldType.Tag.Get("short")
		if shortFlag == "" {
			shortFlag = fieldType.Tag.Get("s")
		}

		if envVar != "" {
			_ = viper.BindEnv(fieldType.Name, envVar)
		}

		if longFlag != "" {
			help := fieldType.Tag.Get("help")
			if help == "" {
				help = fieldType.Tag.Get("h")
			}
			switch field.Type().Kind() {
			case reflect.String:
				if defaultAsString != "" {
					defaultValue = defaultAsString
				}
				viper.SetDefault(fieldType.Name, defaultValue.(string))
				flagSet.StringP(longFlag, shortFlag, defaultValue.(string), help)
			case reflect.Bool:
				if defaultAsString != "" {
					defaultValue, err = strconv.ParseBool(defaultAsString)
					if err != nil {
						return err
					}
				}
				viper.SetDefault(fieldType.Name, defaultValue.(bool))
				flagSet.BoolP(longFlag, shortFlag, defaultValue.(bool), help)
			case reflect.Int: //, reflect.Int8, reflect.Int16, reflect.Int32:
				if defaultAsString != "" {
					defaultValue, err = strconv.ParseInt(defaultAsString, 0, field.Type().Bits())
					if err != nil {
						return err
					}
					defaultValue = int(defaultValue.(int64))
				}
				viper.SetDefault(fieldType.Name, defaultValue.(int))
				flagSet.IntP(longFlag, shortFlag, defaultValue.(int), help)
			case reflect.Int8: //, reflect.Int8, reflect.Int16, reflect.Int32:
				if defaultAsString != "" {
					defaultValue, err = strconv.ParseInt(defaultAsString, 0, field.Type().Bits())
					if err != nil {
						return err
					}
					defaultValue = int8(defaultValue.(int64))
				}
				viper.SetDefault(fieldType.Name, defaultValue.(int8))
				flagSet.Int8P(longFlag, shortFlag, defaultValue.(int8), help)
			case reflect.Int16: //, reflect.Int8, reflect.Int16, reflect.Int32:
				if defaultAsString != "" {
					defaultValue, err = strconv.ParseInt(defaultAsString, 0, field.Type().Bits())
					if err != nil {
						return err
					}
					defaultValue = int16(defaultValue.(int64))
				}
				viper.SetDefault(fieldType.Name, defaultValue.(int16))
				flagSet.Int16P(longFlag, shortFlag, defaultValue.(int16), help)
			case reflect.Int32: //, reflect.Int8, reflect.Int16, reflect.Int32:
				if defaultAsString != "" {
					defaultValue, err = strconv.ParseInt(defaultAsString, 0, field.Type().Bits())
					if err != nil {
						return err
					}
					defaultValue = int32(defaultValue.(int64))
				}
				viper.SetDefault(fieldType.Name, defaultValue.(int32))
				flagSet.Int32P(longFlag, shortFlag, defaultValue.(int32), help)
			case reflect.Int64:
				if field.Type().PkgPath() == "time" && field.Type().Name() == "Duration" {
					if defaultAsString != "" {
						defaultValue, err = time.ParseDuration(defaultAsString)
						if err != nil {
							return err
						}
					}
					viper.SetDefault(fieldType.Name, defaultValue.(time.Duration))
					flagSet.DurationP(longFlag, shortFlag, defaultValue.(time.Duration), help)
				} else {
					if defaultAsString != "" {
						defaultValue, err = strconv.ParseInt(defaultAsString, 0, field.Type().Bits())
						if err != nil {
							return err
						}
					}
					viper.SetDefault(fieldType.Name, defaultValue.(int64))
					flagSet.Int64P(longFlag, shortFlag, defaultValue.(int64), help)
				}
			case reflect.Uint: //, reflect.Int8, reflect.Int16, reflect.Int32:
				if defaultAsString != "" {
					defaultValue, err = strconv.ParseUint(defaultAsString, 0, field.Type().Bits())
					if err != nil {
						return err
					}
					defaultValue = uint(defaultValue.(uint64))
				}
				viper.SetDefault(fieldType.Name, defaultValue.(uint))
				flagSet.UintP(longFlag, shortFlag, defaultValue.(uint), help)
			case reflect.Uint8: //, reflect.Int8, reflect.Int16, reflect.Int32:
				if defaultAsString != "" {
					defaultValue, err = strconv.ParseUint(defaultAsString, 0, field.Type().Bits())
					if err != nil {
						return err
					}
					defaultValue = uint8(defaultValue.(uint64))
				}
				viper.SetDefault(fieldType.Name, defaultValue.(uint8))
				flagSet.Uint8P(longFlag, shortFlag, defaultValue.(uint8), help)
			case reflect.Uint16: //, reflect.Int8, reflect.Int16, reflect.Int32:
				if defaultAsString != "" {
					defaultValue, err = strconv.ParseUint(defaultAsString, 0, field.Type().Bits())
					if err != nil {
						return err
					}
					defaultValue = uint16(defaultValue.(uint64))
				}
				viper.SetDefault(fieldType.Name, defaultValue.(uint16))
				flagSet.Uint16P(longFlag, shortFlag, defaultValue.(uint16), help)
			case reflect.Uint32: //, reflect.Int8, reflect.Int16, reflect.Int32:
				if defaultAsString != "" {
					defaultValue, err = strconv.ParseUint(defaultAsString, 0, field.Type().Bits())
					if err != nil {
						return err
					}
					defaultValue = uint32(defaultValue.(uint64))
				}
				viper.SetDefault(fieldType.Name, defaultValue.(uint32))
				flagSet.Uint32P(longFlag, shortFlag, defaultValue.(uint32), help)
			case reflect.Uint64:
				if defaultAsString != "" {
					defaultValue, err = strconv.ParseUint(defaultAsString, 0, field.Type().Bits())
					if err != nil {
						return err
					}
				}
				viper.SetDefault(fieldType.Name, defaultValue.(uint64))
				flagSet.Uint64P(longFlag, shortFlag, defaultValue.(uint64), help)
			case reflect.Float32:
				if defaultAsString != "" {
					defaultValue, err = strconv.ParseFloat(defaultAsString, field.Type().Bits())
					if err != nil {
						return err
					}
					defaultValue = float32(defaultValue.(float64))
				}
				viper.SetDefault(fieldType.Name, defaultValue.(float32))
				flagSet.Float32P(longFlag, shortFlag, defaultValue.(float32), help)
			case reflect.Float64:
				if defaultAsString != "" {
					defaultValue, err = strconv.ParseFloat(defaultAsString, field.Type().Bits())
					if err != nil {
						return err
					}
				}
				viper.SetDefault(fieldType.Name, defaultValue.(float64))
				flagSet.Float64P(longFlag, shortFlag, defaultValue.(float64), help)
			}
			fmt.Printf("    SETDEF: '%#+v'\n", defaultValue)
			_ = viper.BindPFlag(fieldType.Name, flagSet.Lookup(longFlag))
		}
	}

	return nil
}

// NewConfiguration constructs and returns a new PflagSet based on the structure tags
// associated with the specified configSpecification interface.
func NewConfiguration(configSpecification interface{}, prefix string, options ProcessingOptions, args []string) (*pflag.FlagSet, error) {
	flagSet := pflag.NewFlagSet(path.Base(args[0]), pflag.ExitOnError)
	if err := AddConfiguration(flagSet, configSpecification, prefix, options, args); err != nil {
		return nil, err
	}
	return flagSet, nil
}
