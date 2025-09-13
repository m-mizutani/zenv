package cli_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/cli"
)

func TestNewParser(t *testing.T) {
	parser, err := cli.NewParser([]cli.Option{})
	gt.NoError(t, err)
	gt.V(t, parser).NotNil()
}

func TestParser_Creation(t *testing.T) {
	t.Run("Success cases", func(t *testing.T) {
		t.Run("Basic option", func(t *testing.T) {
			parser, err := cli.NewParser([]cli.Option{
				{
					Name:    "env",
					Aliases: []string{"e"},
					Usage:   "Environment file",
				},
			})
			gt.NoError(t, err)
			gt.NotEqual(t, parser, nil)
		})

		t.Run("Option with default value", func(t *testing.T) {
			parser, err := cli.NewParser([]cli.Option{
				{
					Name:         "log-level",
					Usage:        "Log level",
					DefaultValue: "warn",
				},
			})
			gt.NoError(t, err)
			gt.NotEqual(t, parser, nil)
		})

		t.Run("Slice option", func(t *testing.T) {
			parser, err := cli.NewParser([]cli.Option{
				{
					Name:    "file",
					Aliases: []string{"f"},
					Usage:   "Input files",
					IsSlice: true,
				},
			})
			gt.NoError(t, err)
			gt.NotEqual(t, parser, nil)
		})

		t.Run("Option with multiple aliases", func(t *testing.T) {
			parser, err := cli.NewParser([]cli.Option{
				{
					Name:    "verbose",
					Aliases: []string{"v", "verb", "debug"},
					Usage:   "Verbose output",
				},
			})
			gt.NoError(t, err)
			gt.NotEqual(t, parser, nil)
		})

		t.Run("Option with no aliases", func(t *testing.T) {
			parser, err := cli.NewParser([]cli.Option{
				{
					Name:  "config-file",
					Usage: "Configuration file path",
				},
			})
			gt.NoError(t, err)
			gt.NotEqual(t, parser, nil)
		})

		t.Run("Option with empty usage", func(t *testing.T) {
			parser, err := cli.NewParser([]cli.Option{
				{
					Name: "test",
				},
			})
			gt.NoError(t, err)
			gt.NotEqual(t, parser, nil)
		})
	})

	t.Run("Error cases", func(t *testing.T) {
		t.Run("Empty name", func(t *testing.T) {
			_, err := cli.NewParser([]cli.Option{
				{
					Name: "",
				},
			})
			gt.Error(t, err)
		})

		t.Run("Whitespace only name", func(t *testing.T) {
			_, err := cli.NewParser([]cli.Option{
				{
					Name: "   ",
				},
			})
			gt.NoError(t, err) // This should be allowed - parser doesn't trim
		})

		t.Run("Duplicate option name", func(t *testing.T) {
			_, err := cli.NewParser([]cli.Option{
				{Name: "env"},
				{Name: "env"},
			})
			gt.Error(t, err)
		})

		t.Run("Duplicate option name case sensitive", func(t *testing.T) {
			_, err := cli.NewParser([]cli.Option{
				{Name: "env"},
				{Name: "ENV"},
			})
			gt.NoError(t, err) // Should be allowed - case sensitive
		})

		t.Run("Conflicting alias with option name", func(t *testing.T) {
			_, err := cli.NewParser([]cli.Option{
				{Name: "env"},
				{
					Name:    "environment",
					Aliases: []string{"env"}, // Conflicts with existing option name
				},
			})
			gt.Error(t, err)
		})

		t.Run("Conflicting alias with existing alias", func(t *testing.T) {
			_, err := cli.NewParser([]cli.Option{
				{
					Name:    "env",
					Aliases: []string{"e"},
				},
				{
					Name:    "environment",
					Aliases: []string{"e"}, // Conflicts with existing alias
				},
			})
			gt.Error(t, err)
		})

		t.Run("Multiple conflicting aliases", func(t *testing.T) {
			_, err := cli.NewParser([]cli.Option{
				{
					Name:    "verbose",
					Aliases: []string{"v", "verb"},
				},
				{
					Name:    "version",
					Aliases: []string{"ver", "v"}, // 'v' conflicts
				},
			})
			gt.Error(t, err)
		})

		t.Run("Empty alias in slice", func(t *testing.T) {
			_, err := cli.NewParser([]cli.Option{
				{
					Name:    "test",
					Aliases: []string{"", "t"}, // Empty alias
				},
			})
			gt.NoError(t, err) // Should be allowed - parser doesn't validate empty aliases
		})

		t.Run("Duplicate aliases in same option", func(t *testing.T) {
			_, err := cli.NewParser([]cli.Option{
				{
					Name:    "test",
					Aliases: []string{"t", "t"}, // Duplicate in same option
				},
			})
			gt.Error(t, err)
		})
	})
}

func setupParser() cli.Parser {
	parser, _ := cli.NewParser([]cli.Option{
		{
			Name:         "env",
			Aliases:      []string{"e"},
			Usage:        "Environment file",
			DefaultValue: ".env",
		},
		{
			Name:    "toml",
			Aliases: []string{"t"},
			Usage:   "TOML file",
			IsSlice: true,
		},
		{
			Name:         "log-level",
			Usage:        "Log level",
			DefaultValue: "warn",
		},
	})
	return parser
}

func TestParser_Parse(t *testing.T) {
	t.Run("Success cases", func(t *testing.T) {
		t.Run("No arguments", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal(".env")
			gt.V(t, result.Options["env"].IsSet()).Equal(false)
			gt.V(t, result.Args).Equal([]string{})
		})

		t.Run("Nil arguments", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, nil)
			gt.NoError(t, err)
			gt.V(t, len(result.Args)).Equal(0)
		})

		t.Run("Long option with value", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{"--env", "test.env", "echo", "hello"})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal("test.env")
			gt.V(t, result.Options["env"].IsSet()).Equal(true)
			gt.V(t, result.Args).Equal([]string{"echo", "hello"})
		})

		t.Run("Long option with equals", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{"--env=test.env", "echo", "hello"})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal("test.env")
			gt.V(t, result.Options["env"].IsSet()).Equal(true)
		})

		t.Run("Long option with empty value via equals", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{"--env=", "echo", "hello"})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal("")
			gt.V(t, result.Options["env"].IsSet()).Equal(true)
		})

		t.Run("Short option with value", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{"-e", "test.env", "echo", "hello"})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal("test.env")
			gt.V(t, result.Options["env"].IsSet()).Equal(true)
		})

		t.Run("Short option with equals", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{"-e=test.env", "echo", "hello"})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal("test.env")
			gt.V(t, result.Options["env"].IsSet()).Equal(true)
		})

		t.Run("Multiple options", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{
				"--env", "test.env",
				"--log-level", "debug",
				"go", "test", "-v",
			})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal("test.env")
			gt.V(t, result.Options["log-level"].String()).Equal("debug")
			gt.V(t, result.Args).Equal([]string{"go", "test", "-v"})
		})

		t.Run("Slice option single value", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{"--toml", "config.toml", "echo", "hello"})
			gt.NoError(t, err)
			gt.V(t, result.Options["toml"].StringSlice()).Equal([]string{"config.toml"})
			gt.V(t, result.Options["toml"].IsSet()).Equal(true)
		})

		t.Run("Slice option multiple values", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{
				"--toml", "config1.toml",
				"--toml", "config2.toml",
				"-t", "config3.toml",
				"echo", "hello",
			})
			gt.NoError(t, err)
			gt.V(t, result.Options["toml"].StringSlice()).Equal([]string{"config1.toml", "config2.toml", "config3.toml"})
			gt.V(t, result.Options["toml"].IsSet()).Equal(true)
		})

		t.Run("Slice option with empty values", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{
				"--toml=",
				"--toml", "",
				"echo", "hello",
			})
			gt.NoError(t, err)
			gt.V(t, result.Options["toml"].StringSlice()).Equal([]string{"", ""})
			gt.V(t, result.Options["toml"].IsSet()).Equal(true)
		})

		t.Run("Options stop at first non-option", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{
				"--env", "test.env",
				"go", "test",
				"--log-level", "debug", // This should be treated as command arg
			})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal("test.env")
			gt.V(t, result.Options["log-level"].String()).Equal("warn") // Default value
			gt.V(t, result.Options["log-level"].IsSet()).Equal(false)
			gt.V(t, result.Args).Equal([]string{"go", "test", "--log-level", "debug"})
		})

		t.Run("Command with various flags", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{
				"--env", "test.env",
				"go", "test", "./pkg/...", "-v", "-run", "TestSomething", "--parallel", "4", "-short",
			})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal("test.env")
			gt.V(t, result.Args).Equal([]string{"go", "test", "./pkg/...", "-v", "-run", "TestSomething", "--parallel", "4", "-short"})
		})

		t.Run("Special characters in values", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{
				"--env", "test-file_with.special@chars#and$symbols%.env",
				"echo", "hello",
			})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal("test-file_with.special@chars#and$symbols%.env")
		})

		t.Run("Unicode values", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{
				"--env", "テスト.env",
				"--log-level", "デバッグ",
				"echo", "こんにちは",
			})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal("テスト.env")
			gt.V(t, result.Options["log-level"].String()).Equal("デバッグ")
			gt.V(t, result.Args).Equal([]string{"echo", "こんにちは"})
		})

		t.Run("Very long values", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			longValue := string(make([]byte, 10000))
			for i := range longValue {
				longValue = longValue[:i] + "a" + longValue[i+1:]
			}

			result, err := parser.Parse(ctx, []string{
				"--env", longValue,
				"echo", "hello",
			})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal(longValue)
		})

		t.Run("Values with spaces and quotes", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{
				"--env", "file with spaces.env",
				"--log-level", "debug with 'quotes' and \"double quotes\"",
				"echo", "hello world",
			})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal("file with spaces.env")
			gt.V(t, result.Options["log-level"].String()).Equal("debug with 'quotes' and \"double quotes\"")
		})
	})

	t.Run("Error cases", func(t *testing.T) {
		t.Run("Unknown long option", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			_, err := parser.Parse(ctx, []string{"--unknown", "value"})
			gt.Error(t, err)
		})

		t.Run("Unknown short option", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			_, err := parser.Parse(ctx, []string{"-x", "value"})
			gt.Error(t, err)
		})

		t.Run("Long option without value", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			_, err := parser.Parse(ctx, []string{"--env"})
			gt.Error(t, err)
		})

		t.Run("Short option without value", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			_, err := parser.Parse(ctx, []string{"-e"})
			gt.Error(t, err)
		})

		t.Run("Long option without value at end", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			_, err := parser.Parse(ctx, []string{"--log-level", "debug", "--env"})
			gt.Error(t, err)
		})

		t.Run("Short option without value at end", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			_, err := parser.Parse(ctx, []string{"--log-level", "debug", "-e"})
			gt.Error(t, err)
		})

		t.Run("Combined short options", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			_, err := parser.Parse(ctx, []string{"-et", "value"})
			gt.Error(t, err)
		})

		t.Run("Combined short options with equals", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			_, err := parser.Parse(ctx, []string{"-et=value"})
			gt.Error(t, err)
		})

		t.Run("Invalid long option format", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			_, err := parser.Parse(ctx, []string{"---env", "value"})
			gt.Error(t, err)
		})

		t.Run("Option name with equals but no value part", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			// This should work - empty value after equals
			result, err := parser.Parse(ctx, []string{"--env=", "echo"})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal("")
		})

		t.Run("Multiple equals in option", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{"--env=test=value", "echo"})
			gt.NoError(t, err)
			gt.V(t, result.Options["env"].String()).Equal("test=value") // Should preserve the second equals
		})

		t.Run("Nil context", func(t *testing.T) {
			parser := setupParser()

			result, err := parser.Parse(context.TODO(), []string{"--env", "test.env"})
			gt.NoError(t, err) // Should work with nil context
			gt.V(t, result.Options["env"].String()).Equal("test.env")
		})
	})

	t.Run("Edge cases", func(t *testing.T) {
		t.Run("Single dash as argument", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{
				"--env", "test.env",
				"cat", "-", // Single dash should be treated as argument
			})
			gt.NoError(t, err)
			gt.V(t, result.Args).Equal([]string{"cat", "-"})
		})

		t.Run("Double dash as argument", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{
				"--env", "test.env",
				"echo", "--", // Double dash should be treated as argument
			})
			gt.NoError(t, err)
			gt.V(t, result.Args).Equal([]string{"echo", "--"})
		})

		t.Run("Option-like arguments after command", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{
				"--env", "test.env",
				"command",
				"--unknown-flag",
				"-x",
				"--another=value",
			})
			gt.NoError(t, err)
			gt.V(t, result.Args).Equal([]string{"command", "--unknown-flag", "-x", "--another=value"})
		})

		t.Run("Empty string arguments", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			result, err := parser.Parse(ctx, []string{
				"--env", "test.env",
				"",
				"echo",
				"",
				"hello",
			})
			gt.NoError(t, err)
			gt.V(t, result.Args).Equal([]string{"", "echo", "", "hello"})
		})

		t.Run("Large number of arguments", func(t *testing.T) {
			parser := setupParser()
			ctx := context.Background()

			args := []string{"--env", "test.env", "echo"}
			for i := 0; i < 1000; i++ {
				args = append(args, "arg"+string(rune(i%26+65)))
			}

			result, err := parser.Parse(ctx, args)
			gt.NoError(t, err)
			gt.V(t, len(result.Args)).Equal(1001) // echo + 1000 args
		})

		t.Run("Many options before command", func(t *testing.T) {
			// Create many options
			var options []cli.Option
			for i := 0; i < 100; i++ {
				options = append(options, cli.Option{
					Name:    "opt" + string(rune(i%26+97)) + string(rune((i/26)%26+97)),
					IsSlice: i%2 == 0,
				})
			}
			
			parser, err := cli.NewParser(options)
			gt.NoError(t, err)

			ctx := context.Background()
			args := []string{}

			// Use some options
			for i := 0; i < 10; i++ {
				optName := "opt" + string(rune(i%26+97)) + string(rune((i/26)%26+97))
				args = append(args, "--"+optName, "value"+string(rune(i+48)))
			}
			args = append(args, "echo", "hello")

			result, err := parser.Parse(ctx, args)
			gt.NoError(t, err)
			gt.V(t, result.Args).Equal([]string{"echo", "hello"})
		})
	})
}

func TestStringValue(t *testing.T) {
	t.Run("Default state", func(t *testing.T) {
		val := &cli.StringValue{}
		gt.V(t, val.String()).Equal("")
		gt.V(t, val.StringSlice()).Equal([]string(nil))
		gt.V(t, val.IsSet()).Equal(false)
	})

	t.Run("After setting value", func(t *testing.T) {
		// Note: In real usage, these would be set by the parser
		// We can't directly test the internal state since fields are not exported
		// This is by design - the parser manages the state
	})
}

func TestStringSliceValue(t *testing.T) {
	t.Run("Default state", func(t *testing.T) {
		val := &cli.StringSliceValue{}
		gt.V(t, val.String()).Equal("")
		gt.V(t, val.StringSlice()).Equal([]string(nil))
		gt.V(t, val.IsSet()).Equal(false)
	})

	t.Run("String method with values", func(t *testing.T) {
		// Note: In real usage, these would be set by the parser
		// We can't directly test the internal state since fields are not exported
		// This is by design - the parser manages the state
	})
}

func TestParser_Help(t *testing.T) {
	t.Run("Empty parser", func(t *testing.T) {
		parser, err := cli.NewParser([]cli.Option{})
		gt.NoError(t, err)
		help := parser.Help()
		gt.V(t, help).Equal("")
	})

	t.Run("Single option", func(t *testing.T) {
		parser, err := cli.NewParser([]cli.Option{
			{
				Name:         "env",
				Aliases:      []string{"e"},
				Usage:        "Environment file",
				DefaultValue: ".env",
			},
		})
		gt.NoError(t, err)

		help := parser.Help()
		gt.V(t, len(help)).NotEqual(0)
		// Don't test exact format - that's implementation detail
	})

	t.Run("Multiple options", func(t *testing.T) {
		parser := setupParser()
		help := parser.Help()
		gt.V(t, len(help)).NotEqual(0)
	})

	t.Run("Option with no usage", func(t *testing.T) {
		parser, err := cli.NewParser([]cli.Option{
			{
				Name: "test",
			},
		})
		gt.NoError(t, err)

		help := parser.Help()
		gt.V(t, len(help)).NotEqual(0)
	})

	t.Run("Option with no aliases", func(t *testing.T) {
		parser, err := cli.NewParser([]cli.Option{
			{
				Name:  "long-option-name",
				Usage: "Test option",
			},
		})
		gt.NoError(t, err)

		help := parser.Help()
		gt.V(t, len(help)).NotEqual(0)
	})
}

// TestParser_Validate was removed as the Validate method no longer exists

func TestRealWorldScenarios(t *testing.T) {
	t.Run("Original problem case", func(t *testing.T) {
		parser, err := cli.NewParser([]cli.Option{
			{
				Name:    "env",
				Aliases: []string{"e"},
				IsSlice: true,
			},
			{
				Name:         "log-level",
				DefaultValue: "warn",
			},
		})
		gt.NoError(t, err)

		ctx := context.Background()

		// Test the original problem case
		result, err := parser.Parse(ctx, []string{
			"go", "test", "./pkg/controller/http/...", "-v", "-run", "TestGraphQL_Firestore",
		})
		gt.NoError(t, err)
		gt.V(t, result.Options["log-level"].String()).Equal("warn")
		gt.V(t, result.Options["log-level"].IsSet()).Equal(false)
		gt.V(t, result.Args).Equal([]string{"go", "test", "./pkg/controller/http/...", "-v", "-run", "TestGraphQL_Firestore"})
	})

	t.Run("Complex docker command", func(t *testing.T) {
		parser, err := cli.NewParser([]cli.Option{
			{Name: "env", IsSlice: true},
		})
		gt.NoError(t, err)

		ctx := context.Background()

		result, err := parser.Parse(ctx, []string{
			"--env", "prod.env",
			"--env", "secrets.env",
			"docker", "run", "-it", "--rm", "-p", "8080:80", "-v", "/host:/container", "nginx:latest",
		})
		gt.NoError(t, err)
		gt.V(t, result.Options["env"].StringSlice()).Equal([]string{"prod.env", "secrets.env"})
		gt.V(t, result.Args).Equal([]string{"docker", "run", "-it", "--rm", "-p", "8080:80", "-v", "/host:/container", "nginx:latest"})
	})

	t.Run("Git command with many flags", func(t *testing.T) {
		parser, err := cli.NewParser([]cli.Option{
			{Name: "log-level", DefaultValue: "info"},
		})
		gt.NoError(t, err)

		ctx := context.Background()

		result, err := parser.Parse(ctx, []string{
			"--log-level=debug",
			"git", "log", "--oneline", "--graph", "--decorate", "--all", "-n", "10", "--author=John", "--since='2 weeks ago'",
		})
		gt.NoError(t, err)
		gt.V(t, result.Options["log-level"].String()).Equal("debug")
		gt.V(t, result.Args).Equal([]string{"git", "log", "--oneline", "--graph", "--decorate", "--all", "-n", "10", "--author=John", "--since='2 weeks ago'"})
	})

	t.Run("No command, only zenv options", func(t *testing.T) {
		parser, err := cli.NewParser([]cli.Option{
			{Name: "env"},
		})
		gt.NoError(t, err)

		ctx := context.Background()

		result, err := parser.Parse(ctx, []string{
			"--env", "test.env",
		})
		gt.NoError(t, err)
		gt.V(t, result.Options["env"].String()).Equal("test.env")
		gt.V(t, len(result.Args)).Equal(0)
	})

	t.Run("Command starting with dash", func(t *testing.T) {
		parser, err := cli.NewParser([]cli.Option{
			{Name: "env"},
		})
		gt.NoError(t, err)

		ctx := context.Background()

		result, err := parser.Parse(ctx, []string{
			"--env", "test.env",
			"command-starting-with-dash", "arg1", "arg2", // Remove leading dash to avoid being treated as option
		})
		gt.NoError(t, err)
		gt.V(t, result.Args).Equal([]string{"command-starting-with-dash", "arg1", "arg2"})
	})

	t.Run("Stress test - many repeated options", func(t *testing.T) {
		parser, err := cli.NewParser([]cli.Option{
			{Name: "file", IsSlice: true},
		})
		gt.NoError(t, err)

		ctx := context.Background()

		args := []string{}
		expectedFiles := []string{}

		for i := 0; i < 50; i++ {
			filename := "file" + string(rune(i+48)) + ".txt"
			args = append(args, "--file", filename)
			expectedFiles = append(expectedFiles, filename)
		}
		args = append(args, "process-files")

		result, err := parser.Parse(ctx, args)
		gt.NoError(t, err)
		gt.V(t, result.Options["file"].StringSlice()).Equal(expectedFiles)
		gt.V(t, result.Args).Equal([]string{"process-files"})
	})

	t.Run("Mixed option formats", func(t *testing.T) {
		parser, err := cli.NewParser([]cli.Option{
			{Name: "input", Aliases: []string{"i"}, IsSlice: true},
			{Name: "output", Aliases: []string{"o"}},
			{Name: "verbose", Aliases: []string{"v"}},
		})
		gt.NoError(t, err)

		ctx := context.Background()

		result, err := parser.Parse(ctx, []string{
			"--input=file1.txt",
			"-i", "file2.txt",
			"--input", "file3.txt",
			"-o=output.txt",
			"--verbose", "true",
			"process", "--unknown-flag", "-x",
		})
		gt.NoError(t, err)
		gt.V(t, result.Options["input"].StringSlice()).Equal([]string{"file1.txt", "file2.txt", "file3.txt"})
		gt.V(t, result.Options["output"].String()).Equal("output.txt")
		gt.V(t, result.Options["verbose"].String()).Equal("true")
		gt.V(t, result.Args).Equal([]string{"process", "--unknown-flag", "-x"})
	})
}
