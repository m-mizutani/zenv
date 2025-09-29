package model

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/m-mizutani/goerr/v2"
)

type TOMLConfig map[string]TOMLValue

type TOMLValue struct {
	Value    *string  `toml:"value,omitempty"`
	File     *string  `toml:"file,omitempty"`
	Command  *string  `toml:"command,omitempty"`
	Args     []string `toml:"args,omitempty"`
	Alias    *string  `toml:"alias,omitempty"`
	Template *string  `toml:"template,omitempty"`
	Refs     []string `toml:"refs,omitempty"`
}

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

	if count == 0 {
		return goerr.New("no value specified")
	}
	if count > 1 {
		return goerr.New("multiple value types specified (only one of value, file, command, alias, or template can be specified)")
	}

	// Refs can only be used with templates
	if len(v.Refs) > 0 && v.Template == nil {
		return goerr.New("refs can only be used with template")
	}

	return nil
}

// UnmarshalTOML implements the toml.Unmarshaler interface to support both
// simple format (KEY = "value") and section format ([KEY])
func (c *TOMLConfig) UnmarshalTOML(data any) error {
	// dataをmap[string]anyとして受け取る
	raw, ok := data.(map[string]any)
	if !ok {
		return goerr.New("invalid TOML format")
	}

	*c = make(TOMLConfig)

	for key, value := range raw {
		switch v := value.(type) {
		case string:
			// シンプル形式: KEY = "value"
			(*c)[key] = TOMLValue{Value: &v}
		case map[string]any:
			// セクション形式: [KEY] ...
			// 一度エンコードして再度デコードすることで型安全に変換
			var buf bytes.Buffer
			if err := toml.NewEncoder(&buf).Encode(v); err != nil {
				return goerr.Wrap(err, "failed to re-encode section",
					goerr.V("key", key))
			}

			var tv TOMLValue
			if _, err := toml.NewDecoder(&buf).Decode(&tv); err != nil {
				return goerr.Wrap(err, "failed to decode section",
					goerr.V("key", key))
			}

			// Validate the decoded value
			if err := tv.Validate(); err != nil {
				return goerr.Wrap(err, "invalid section configuration",
					goerr.V("key", key))
			}

			(*c)[key] = tv
		default:
			return goerr.New("unsupported type for key",
				goerr.V("key", key),
				goerr.V("type", fmt.Sprintf("%T", value)))
		}
	}
	return nil
}
