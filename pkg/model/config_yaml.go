package model

import (
	"github.com/m-mizutani/goerr/v2"
	"gopkg.in/yaml.v3"
)

// YAMLConfig represents a YAML configuration file with environment variables
type YAMLConfig map[string]YAMLValue

// YAMLValue represents a single environment variable configuration with multiple source options
type YAMLValue struct {
	// Value is a direct string value
	Value *string `yaml:"value,omitempty"`
	// File specifies a file path to read the value from
	File *string `yaml:"file,omitempty"`
	// Command specifies a command to execute to get the value
	Command []string `yaml:"command,omitempty"`
	// Alias references another environment variable
	Alias *string `yaml:"alias,omitempty"`
	// Refs lists the environment variables referenced in value or command templates
	Refs []string `yaml:"refs,omitempty"`
	// Profile contains profile-specific configurations
	Profile map[string]*YAMLValue `yaml:"profile,omitempty"`
}

// IsEmpty checks if YAMLValue represents an empty object.
// This is used to determine if a profile configuration should unset the variable.
func (v *YAMLValue) IsEmpty() bool {
	return v != nil &&
		v.Value == nil &&
		v.File == nil &&
		len(v.Command) == 0 &&
		v.Alias == nil &&
		len(v.Refs) == 0 &&
		len(v.Profile) == 0
}

// GetValueForProfile returns the YAMLValue for the specified profile.
// If the profile exists and is not nil, it returns the profile-specific value.
// Otherwise, it returns the base YAMLValue (self).
func (v *YAMLValue) GetValueForProfile(profile string) *YAMLValue {
	if profile != "" && v.Profile != nil {
		if profileValue, exists := v.Profile[profile]; exists {
			return profileValue
		}
	}
	// Return the default (self) if no profile match
	return v
}

// Validate checks that the YAMLValue configuration is valid.
// Rules:
// - Only one of value, file, command, or alias can be specified
// - Refs can only be used with value or command
// - Nested profiles are not allowed
func (v YAMLValue) Validate() error {
	// Refs should only be used with value or command (check this first to give more specific error)
	if v.Value == nil && len(v.Command) == 0 && len(v.Refs) > 0 {
		return goerr.New("refs can only be used with value or command")
	}

	count := 0
	if v.Value != nil {
		count++
	}
	if v.File != nil {
		count++
	}
	if len(v.Command) > 0 {
		count++
	}
	if v.Alias != nil {
		count++
	}

	// Allow empty values only if profile is present
	if count == 0 && len(v.Profile) == 0 {
		return goerr.New("no value specified")
	}
	if count > 1 {
		return goerr.New("multiple value types specified (only one of value, file, command, or alias can be specified)")
	}

	// Validate profile values
	for profileName, profileValue := range v.Profile {
		if profileValue == nil {
			continue // Allow nil for unset
		}
		// Empty object is allowed (for unset)
		if profileValue.IsEmpty() {
			continue
		}
		// Profile within profile is not allowed
		if len(profileValue.Profile) > 0 {
			return goerr.New("nested profile is not allowed",
				goerr.V("profile", profileName))
		}
		// Validate non-empty profile value
		if err := profileValue.Validate(); err != nil {
			return goerr.Wrap(err, "invalid profile configuration",
				goerr.V("profile", profileName))
		}
	}

	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for YAMLValue.
// It supports multiple formats:
// - Direct string: KEY: "value" or dev: "value"
// - Structured map: KEY: {value: "x"} or dev: {value: "x"}
func (v *YAMLValue) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		// Direct string value: KEY: "value"
		*v = YAMLValue{Value: &node.Value}
		return nil
	case yaml.MappingNode:
		// Structured value: KEY: {value: "x", file: "y", ...}
		// Use type alias to prevent infinite recursion
		type yamlValueAlias YAMLValue
		var temp yamlValueAlias

		if err := node.Decode(&temp); err != nil {
			return goerr.Wrap(err, "failed to decode YAMLValue")
		}

		*v = YAMLValue(temp)
		return nil
	default:
		return goerr.New("unsupported type for YAMLValue", goerr.V("kind", node.Kind))
	}
}
