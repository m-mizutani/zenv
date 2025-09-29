package model

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/m-mizutani/goerr/v2"
)

// TOMLConfig represents a TOML configuration file with environment variables
type TOMLConfig map[string]TOMLValue

// TOMLValue represents a single environment variable configuration with multiple source options
type TOMLValue struct {
	// Value is a direct string value
	Value *string `toml:"value,omitempty"`
	// File specifies a file path to read the value from
	File *string `toml:"file,omitempty"`
	// Command specifies a command to execute to get the value
	Command *string `toml:"command,omitempty"`
	// Args are arguments for the command
	Args []string `toml:"args,omitempty"`
	// Alias references another environment variable
	Alias *string `toml:"alias,omitempty"`
	// Template is a Go template string that can reference other variables
	Template *string `toml:"template,omitempty"`
	// Refs lists the environment variables referenced in the template
	Refs []string `toml:"refs,omitempty"`
	// Profile contains profile-specific configurations
	Profile map[string]*TOMLValue `toml:"profile,omitempty"`
}

// IsEmpty checks if TOMLValue represents an empty object.
// This is used to determine if a profile configuration should unset the variable.
func (v *TOMLValue) IsEmpty() bool {
	return v != nil &&
		v.Value == nil &&
		v.File == nil &&
		v.Command == nil &&
		v.Alias == nil &&
		v.Template == nil &&
		len(v.Args) == 0 &&
		len(v.Refs) == 0 &&
		len(v.Profile) == 0
}

// GetValueForProfile returns the TOMLValue for the specified profile.
// If the profile exists and is not nil, it returns the profile-specific value.
// Otherwise, it returns the base TOMLValue (self).
func (v *TOMLValue) GetValueForProfile(profile string) *TOMLValue {
	if profile != "" && v.Profile != nil {
		if profileValue, exists := v.Profile[profile]; exists {
			return profileValue
		}
	}
	// Return the default (self) if no profile match
	return v
}

// Validate checks that the TOMLValue configuration is valid.
// Rules:
// - Only one of value, file, command, alias, or template can be specified
// - Refs can only be used with template
// - Nested profiles are not allowed
func (v TOMLValue) Validate() error {
	// Refs should only be used with template (check this first to give more specific error)
	if v.Template == nil && len(v.Refs) > 0 {
		return goerr.New("refs can only be used with template")
	}

	count := 0
	if v.Value != nil {
		count++
	}
	if v.File != nil {
		count++
	}
	if v.Command != nil {
		count++
	}
	if v.Alias != nil {
		count++
	}
	if v.Template != nil {
		count++
	}

	// Allow empty values only if profile is present
	if count == 0 && len(v.Profile) == 0 {
		return goerr.New("no value specified")
	}
	if count > 1 {
		return goerr.New("multiple value types specified (only one of value, file, command, alias, or template can be specified)")
	}

	// Refs can only be used with templates
	if len(v.Refs) > 0 && v.Template == nil {
		return goerr.New("refs can only be used with template")
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

// UnmarshalTOML implements the toml.Unmarshaler interface for TOMLValue.
// It supports multiple formats:
// - Direct string: KEY = "value" or profile.dev = "value"
// - Structured map: [KEY] with fields or profile.dev = {value = "x"}
func (v *TOMLValue) UnmarshalTOML(data any) error {
	switch val := data.(type) {
	case string:
		// Direct string value: KEY = "value" or profile.dev = "value"
		*v = TOMLValue{Value: &val}
		return nil
	case map[string]any:
		// Structured value: [KEY] with fields or profile.dev = {value = "x"}
		// Use type alias to prevent infinite recursion
		type tomlValueAlias TOMLValue
		var temp tomlValueAlias

		// Re-encode and decode to leverage TOML library's type handling
		var buf bytes.Buffer
		if err := toml.NewEncoder(&buf).Encode(val); err != nil {
			return goerr.Wrap(err, "failed to encode TOMLValue")
		}

		if _, err := toml.NewDecoder(&buf).Decode(&temp); err != nil {
			return goerr.Wrap(err, "failed to decode TOMLValue")
		}

		*v = TOMLValue(temp)
		return nil
	default:
		return goerr.New("unsupported type for TOMLValue", goerr.V("type", fmt.Sprintf("%T", data)))
	}
}

// mergeTOMLValues merges two TOMLValue structures.
// The 'from' values take precedence over 'base' values.
// This is used to handle self-referencing TOML configurations where
// KEY.field = value appears inside [KEY] section.
func mergeTOMLValues(base, from *TOMLValue) *TOMLValue {
	if from == nil {
		return base
	}
	if base == nil {
		return from
	}

	result := &TOMLValue{}

	// Copy all fields, with 'from' taking precedence
	if from.Value != nil {
		result.Value = from.Value
	} else {
		result.Value = base.Value
	}

	if from.File != nil {
		result.File = from.File
	} else {
		result.File = base.File
	}

	if from.Command != nil {
		result.Command = from.Command
	} else {
		result.Command = base.Command
	}

	if from.Alias != nil {
		result.Alias = from.Alias
	} else {
		result.Alias = base.Alias
	}

	if from.Template != nil {
		result.Template = from.Template
	} else {
		result.Template = base.Template
	}

	if len(from.Args) > 0 {
		result.Args = from.Args
	} else {
		result.Args = base.Args
	}

	if len(from.Refs) > 0 {
		result.Refs = from.Refs
	} else {
		result.Refs = base.Refs
	}

	// Merge profiles
	if base.Profile != nil || from.Profile != nil {
		result.Profile = make(map[string]*TOMLValue)
		// Copy from base
		for k, v := range base.Profile {
			result.Profile[k] = v
		}
		// Override with from
		for k, v := range from.Profile {
			result.Profile[k] = v
		}
	}

	return result
}

// UnmarshalTOML implements the toml.Unmarshaler interface for TOMLConfig.
// It supports multiple TOML formats:
//
// 1. Simple format:
//    KEY = "value"
//
// 2. Section format:
//    [KEY]
//    value = "something"
//    profile.dev = "dev-value"
//
// 3. Self-referencing format (special handling):
//    [KEY]
//    value = "default"
//    KEY.profile.dev = "dev-value"  # Self-reference inside [KEY] section
//
// The self-referencing format requires special handling because the TOML
// library creates a nested structure that needs to be flattened.
func (c *TOMLConfig) UnmarshalTOML(data any) error {
	// Use type alias to avoid infinite recursion
	type configAlias map[string]TOMLValue

	// First, try normal decoding
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(data); err != nil {
		return goerr.Wrap(err, "failed to encode config")
	}

	// Check if it contains self-reference by looking at the raw structure
	needsSelfRefHandling := false
	if raw, ok := data.(map[string]any); ok {
		for key, value := range raw {
			if mapVal, ok := value.(map[string]any); ok {
				if _, hasSelfRef := mapVal[key]; hasSelfRef {
					needsSelfRefHandling = true
					break
				}
			}
		}
	}

	if !needsSelfRefHandling {
		// Simple case - decode normally
		var temp configAlias
		if _, err := toml.NewDecoder(&buf).Decode(&temp); err != nil {
			return goerr.Wrap(err, "failed to decode config")
		}

		*c = TOMLConfig(temp)

		// Validate all values
		for key, tv := range *c {
			if err := tv.Validate(); err != nil {
				return goerr.Wrap(err, "invalid configuration", goerr.V("key", key))
			}
		}
		return nil
	}

	// Self-reference case - needs special handling
	// We can't avoid map[string]any here because TOML library gives us this
	raw := data.(map[string]any)
	*c = make(TOMLConfig)

	for key, value := range raw {
		var tv TOMLValue

		switch v := value.(type) {
		case string:
			tv.Value = &v
		case map[string]any:
			// Handle potential self-reference
			if selfRef, hasSelfRef := v[key]; hasSelfRef {
				// Remove self-reference temporarily
				delete(v, key)

				// Decode main value
				if err := tv.UnmarshalTOML(v); err != nil {
					return goerr.Wrap(err, "failed to unmarshal value", goerr.V("key", key))
				}

				// Decode and merge self-reference
				var selfRefTV TOMLValue
				if err := selfRefTV.UnmarshalTOML(selfRef); err != nil {
					return goerr.Wrap(err, "failed to unmarshal self-reference", goerr.V("key", key))
				}

				// Merge (self-ref takes precedence)
				merged := mergeTOMLValues(&tv, &selfRefTV)
				tv = *merged
			} else {
				// No self-reference, decode normally
				if err := tv.UnmarshalTOML(v); err != nil {
					return goerr.Wrap(err, "failed to unmarshal value", goerr.V("key", key))
				}
			}
		default:
			return goerr.New("unsupported type", goerr.V("key", key), goerr.V("type", fmt.Sprintf("%T", value)))
		}

		// Validate
		if err := tv.Validate(); err != nil {
			return goerr.Wrap(err, "invalid configuration", goerr.V("key", key))
		}

		(*c)[key] = tv
	}

	return nil
}
