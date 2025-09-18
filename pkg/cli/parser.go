package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
)

// ErrHelpRequested is returned when help flag is detected
type helpRequestedError struct{}

func (e helpRequestedError) Error() string {
	return "help requested"
}

var ErrHelpRequested = helpRequestedError{}

// OptionValue represents a parsed option value
type OptionValue interface {
	String() string
	StringSlice() []string
	IsSet() bool
}

// StringValue holds a string option value
type StringValue struct {
	value string
	set   bool
}

func (s *StringValue) String() string {
	return s.value
}

func (s *StringValue) StringSlice() []string {
	if !s.set {
		return nil
	}
	return []string{s.value}
}

func (s *StringValue) IsSet() bool {
	return s.set
}

// StringSliceValue holds a string slice option value
type StringSliceValue struct {
	values []string
	set    bool
}

func (s *StringSliceValue) String() string {
	if len(s.values) == 0 {
		return ""
	}
	return s.values[0]
}

func (s *StringSliceValue) StringSlice() []string {
	if !s.set {
		return nil
	}
	return s.values
}

func (s *StringSliceValue) IsSet() bool {
	return s.set
}

// Option defines an option that can be parsed
type Option struct {
	Name         string   // Long name (e.g., "env")
	Aliases      []string // Short aliases (e.g., ["e"])
	Usage        string   // Description for help
	DefaultValue string   // Default value
	IsSlice      bool     // Whether this option accepts multiple values
	IsBoolean    bool     // Whether this is a boolean flag (no value required)
}


// ParseResult contains the result of parsing command line arguments
type ParseResult struct {
	Options map[string]OptionValue // Parsed options
	Args    []string               // Remaining arguments (command to execute)
}

// Parser defines the interface for command line parsing
type Parser interface {
	// Parse parses the given arguments and returns options and remaining args
	Parse(ctx context.Context, args []string) (*ParseResult, error)

	// Help returns help text for all registered options
	Help() string
}

// DefaultParser implements the Parser interface
type DefaultParser struct {
	options map[string]*Option     // options by name
	aliases map[string]*Option     // aliases to options mapping
	values  map[string]OptionValue // parsed values
}

// NewParser creates a new default parser with the given options
func NewParser(opts []Option) (Parser, error) {
	p := &DefaultParser{
		options: make(map[string]*Option),
		aliases: make(map[string]*Option),
		values:  make(map[string]OptionValue),
	}

	// Initialize options
	for _, opt := range opts {
		if err := p.addOption(opt); err != nil {
			return nil, goerr.Wrap(err, "failed to add option")
		}
	}

	return p, nil
}

// addOption adds an option definition (internal method)
func (p *DefaultParser) addOption(opt Option) error {
	if opt.Name == "" {
		return goerr.New("option name cannot be empty")
	}

	if _, exists := p.options[opt.Name]; exists {
		return goerr.New("option already exists", goerr.V("name", opt.Name))
	}

	// Check for duplicate aliases within same option
	aliasMap := make(map[string]bool)
	for _, alias := range opt.Aliases {
		if alias == "" {
			continue // Skip empty aliases
		}
		if aliasMap[alias] {
			return goerr.New("duplicate alias in same option",
				goerr.V("alias", alias))
		}
		aliasMap[alias] = true
	}

	// Check for conflicting aliases
	for _, alias := range opt.Aliases {
		if alias == "" {
			continue // Skip empty aliases
		}
		if existing, exists := p.aliases[alias]; exists {
			return goerr.New("alias conflicts with existing option",
				goerr.V("alias", alias),
				goerr.V("existing", existing.Name))
		}
		if _, exists := p.options[alias]; exists {
			return goerr.New("alias conflicts with existing option name",
				goerr.V("alias", alias))
		}
	}

	// Store option
	optCopy := opt
	p.options[opt.Name] = &optCopy

	// Register aliases
	for _, alias := range opt.Aliases {
		if alias != "" {
			p.aliases[alias] = &optCopy
		}
	}

	return nil
}

// Parse parses command line arguments
func (p *DefaultParser) Parse(ctx context.Context, args []string) (*ParseResult, error) {
	result := &ParseResult{
		Options: make(map[string]OptionValue),
		Args:    []string{},
	}

	// Initialize default values
	for name, opt := range p.options {
		if opt.IsSlice {
			result.Options[name] = &StringSliceValue{values: []string{}, set: false}
		} else {
			result.Options[name] = &StringValue{value: opt.DefaultValue, set: false}
		}
	}

	i := 0
	for i < len(args) {
		arg := args[i]

		// Check for -- (end of options marker)
		if arg == "--" {
			// Everything after -- should be treated as arguments
			result.Args = append(result.Args, args[i+1:]...)
			break
		}

		// Check if this looks like an option
		if !strings.HasPrefix(arg, "-") {
			// Not an option - treat this and everything after as command arguments
			result.Args = args[i:]
			break
		}

		// Parse the option
		var optName string
		var value string
		var hasValue bool

		if after, found := strings.CutPrefix(arg, "--"); found {
			// Long option
			optName = after
			if idx := strings.Index(optName, "="); idx >= 0 {
				value = optName[idx+1:]
				optName = optName[:idx]
				hasValue = true
			}
		} else {
			// Short option
			optName = strings.TrimPrefix(arg, "-")
			if len(optName) > 1 && !strings.Contains(optName, "=") {
				// Could be combined short options (not supported yet)
				return nil, goerr.New("combined short options not supported: " + arg)
			}
			if idx := strings.Index(optName, "="); idx >= 0 {
				value = optName[idx+1:]
				optName = optName[:idx]
				hasValue = true
			}
		}

		// Find the option
		var option *Option
		if opt, exists := p.options[optName]; exists {
			option = opt
		} else if opt, exists := p.aliases[optName]; exists {
			option = opt
		} else {
			return nil, goerr.New("unknown option: " + optName)
		}

		// Get the value
		if !hasValue {
			// For boolean flags, no value is needed
			if option.IsBoolean {
				value = "true"
			} else {
				// Value should be in next argument
				if i+1 >= len(args) {
					return nil, goerr.New("option '" + optName + "' requires a value")
				}
				i++
				value = args[i]
			}
		}

		// Store the value
		if option.IsSlice {
			if existing, ok := result.Options[option.Name].(*StringSliceValue); ok {
				existing.values = append(existing.values, value)
				existing.set = true
			}
		} else {
			result.Options[option.Name] = &StringValue{value: value, set: true}
		}

		i++
	}

	// Check if help was requested
	if helpVal, exists := result.Options["help"]; exists && helpVal.IsSet() {
		return nil, ErrHelpRequested
	}

	return result, nil
}

// Help returns help text
func (p *DefaultParser) Help() string {
	var parts []string

	for _, opt := range p.options {
		var names []string

		// Add aliases
		for _, alias := range opt.Aliases {
			names = append(names, "-"+alias)
		}

		// Add long name
		names = append(names, "--"+opt.Name)

		line := fmt.Sprintf("  %s", strings.Join(names, ", "))
		if opt.Usage != "" {
			line += fmt.Sprintf("  %s", opt.Usage)
		}
		if opt.DefaultValue != "" {
			line += fmt.Sprintf(" (default: %s)", opt.DefaultValue)
		}

		parts = append(parts, line)
	}

	return strings.Join(parts, "\n")
}
