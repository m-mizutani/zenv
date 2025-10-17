package model_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/model"
	"gopkg.in/yaml.v3"
)

func TestYAMLConfigUnmarshal_SimpleFormat(t *testing.T) {
	// Test for simple format only
	input := `
DATABASE_URL: "postgres://localhost/mydb"
API_KEY: "secret123"
PORT: "3000"
`

	var config model.YAMLConfig
	err := yaml.Unmarshal([]byte(input), &config)
	gt.NoError(t, err)

	// DATABASE_URL
	gt.V(t, config).NotNil()
	gt.V(t, config["DATABASE_URL"].Value).NotNil()
	gt.V(t, *config["DATABASE_URL"].Value).Equal("postgres://localhost/mydb")
	gt.V(t, config["DATABASE_URL"].File).Nil()
	gt.V(t, config["DATABASE_URL"].Command).Nil()

	// API_KEY
	gt.V(t, config["API_KEY"].Value).NotNil()
	gt.V(t, *config["API_KEY"].Value).Equal("secret123")

	// PORT
	gt.V(t, config["PORT"].Value).NotNil()
	gt.V(t, *config["PORT"].Value).Equal("3000")
}

func TestYAMLConfigUnmarshal_StructuredFormat(t *testing.T) {
	// Test for structured format only
	input := `
DATABASE_URL:
  value: "postgres://localhost/mydb"

SSL_CERT:
  file: "/path/to/cert.pem"

AUTH_TOKEN:
  command: ["aws", "secretsmanager", "get-secret-value"]
`

	var config model.YAMLConfig
	err := yaml.Unmarshal([]byte(input), &config)
	gt.NoError(t, err)

	// DATABASE_URL
	gt.V(t, config["DATABASE_URL"].Value).NotNil()
	gt.V(t, *config["DATABASE_URL"].Value).Equal("postgres://localhost/mydb")

	// SSL_CERT
	gt.V(t, config["SSL_CERT"].File).NotNil()
	gt.V(t, *config["SSL_CERT"].File).Equal("/path/to/cert.pem")

	// AUTH_TOKEN
	gt.V(t, len(config["AUTH_TOKEN"].Command)).Equal(3)
	gt.V(t, config["AUTH_TOKEN"].Command[0]).Equal("aws")
	gt.V(t, config["AUTH_TOKEN"].Command[1]).Equal("secretsmanager")
	gt.V(t, config["AUTH_TOKEN"].Command[2]).Equal("get-secret-value")
}

func TestYAMLConfigUnmarshal_MixedFormat(t *testing.T) {
	// Test for mixed format - YAML allows mixing unlike TOML
	input := `
# Simple format
PORT: "3000"
ENV: "development"
API_KEY: "secret123"

# Structured format
DATABASE_URL:
  value: "postgres://localhost/mydb"

SSL_CERT:
  file: "/path/to/cert.pem"

AUTH_TOKEN:
  command: ["aws", "secretsmanager", "get-secret-value"]
`

	var config model.YAMLConfig
	err := yaml.Unmarshal([]byte(input), &config)
	gt.NoError(t, err)

	// Check the actual keys in the config
	gt.V(t, len(config)).Equal(6) // PORT, ENV, API_KEY, DATABASE_URL, SSL_CERT, AUTH_TOKEN

	// Simple format
	gt.V(t, config["PORT"].Value).NotNil()
	gt.V(t, *config["PORT"].Value).Equal("3000")
	gt.V(t, config["ENV"].Value).NotNil()
	gt.V(t, *config["ENV"].Value).Equal("development")
	gt.V(t, config["API_KEY"].Value).NotNil()
	gt.V(t, *config["API_KEY"].Value).Equal("secret123")

	// Structured format
	gt.V(t, config["DATABASE_URL"].Value).NotNil()
	gt.V(t, *config["DATABASE_URL"].Value).Equal("postgres://localhost/mydb")
	gt.V(t, config["SSL_CERT"].File).NotNil()
	gt.V(t, *config["SSL_CERT"].File).Equal("/path/to/cert.pem")
	gt.V(t, config["AUTH_TOKEN"].Command).NotNil()
	gt.V(t, len(config["AUTH_TOKEN"].Command)).Equal(3)
}

func TestYAMLValue_Validate(t *testing.T) {
	t.Run("valid value", func(t *testing.T) {
		value := "test"
		v := model.YAMLValue{Value: &value}
		gt.NoError(t, v.Validate())
	})

	t.Run("valid file", func(t *testing.T) {
		file := "/path/to/file"
		v := model.YAMLValue{File: &file}
		gt.NoError(t, v.Validate())
	})

	t.Run("valid command", func(t *testing.T) {
		v := model.YAMLValue{Command: []string{"echo", "hello"}}
		gt.NoError(t, v.Validate())
	})

	t.Run("valid alias", func(t *testing.T) {
		alias := "OTHER_VAR"
		v := model.YAMLValue{Alias: &alias}
		gt.NoError(t, v.Validate())
	})

	t.Run("multiple types error", func(t *testing.T) {
		value := "test"
		file := "/path/to/file"
		v := model.YAMLValue{Value: &value, File: &file}
		err := v.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("multiple value types specified")
	})

	t.Run("no value specified error", func(t *testing.T) {
		v := model.YAMLValue{}
		err := v.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("no value specified")
	})

	t.Run("refs with value is valid", func(t *testing.T) {
		value := "hello {{ .NAME }}"
		v := model.YAMLValue{Value: &value, Refs: []string{"NAME"}}
		gt.NoError(t, v.Validate())
	})

	t.Run("refs without value or command error", func(t *testing.T) {
		v := model.YAMLValue{Refs: []string{"NAME"}}
		err := v.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("refs can only be used with value or command")
	})
}

func TestYAMLValue_IsEmpty(t *testing.T) {
	t.Run("empty value", func(t *testing.T) {
		v := &model.YAMLValue{}
		gt.True(t, v.IsEmpty())
	})

	t.Run("with value", func(t *testing.T) {
		value := "test"
		v := &model.YAMLValue{Value: &value}
		gt.False(t, v.IsEmpty())
	})

	t.Run("with file", func(t *testing.T) {
		file := "/path"
		v := &model.YAMLValue{File: &file}
		gt.False(t, v.IsEmpty())
	})

	t.Run("with command", func(t *testing.T) {
		v := &model.YAMLValue{Command: []string{"echo"}}
		gt.False(t, v.IsEmpty())
	})

	t.Run("with alias", func(t *testing.T) {
		alias := "VAR"
		v := &model.YAMLValue{Alias: &alias}
		gt.False(t, v.IsEmpty())
	})

	t.Run("with refs", func(t *testing.T) {
		v := &model.YAMLValue{Refs: []string{"NAME"}}
		gt.False(t, v.IsEmpty())
	})

	t.Run("with profile", func(t *testing.T) {
		v := &model.YAMLValue{Profile: map[string]*model.YAMLValue{"dev": nil}}
		gt.False(t, v.IsEmpty())
	})
}

func TestYAMLValue_GetValueForProfile(t *testing.T) {
	t.Run("no profile specified", func(t *testing.T) {
		value := "default"
		v := &model.YAMLValue{Value: &value}
		result := v.GetValueForProfile("")
		gt.V(t, result).Equal(v)
	})

	t.Run("profile exists", func(t *testing.T) {
		value := "default"
		devValue := "dev-value"
		v := &model.YAMLValue{
			Value: &value,
			Profile: map[string]*model.YAMLValue{
				"dev": {Value: &devValue},
			},
		}
		result := v.GetValueForProfile("dev")
		gt.V(t, result.Value).NotNil()
		gt.V(t, *result.Value).Equal("dev-value")
	})

	t.Run("profile does not exist", func(t *testing.T) {
		value := "default"
		v := &model.YAMLValue{
			Value:   &value,
			Profile: map[string]*model.YAMLValue{},
		}
		result := v.GetValueForProfile("prod")
		gt.V(t, result).Equal(v)
	})

	t.Run("profile is nil (unset)", func(t *testing.T) {
		value := "default"
		v := &model.YAMLValue{
			Value: &value,
			Profile: map[string]*model.YAMLValue{
				"prod": nil,
			},
		}
		result := v.GetValueForProfile("prod")
		gt.V(t, result).Nil()
	})
}

func TestYAMLValue_UnmarshalYAML(t *testing.T) {
	t.Run("simple string", func(t *testing.T) {
		input := `MY_VAR: "simple value"`
		var config model.YAMLConfig
		err := yaml.Unmarshal([]byte(input), &config)
		gt.NoError(t, err)
		gt.V(t, config["MY_VAR"].Value).NotNil()
		gt.V(t, *config["MY_VAR"].Value).Equal("simple value")
	})

	t.Run("structured value", func(t *testing.T) {
		input := `
MY_VAR:
  value: "structured value"
  refs:
    - OTHER_VAR
`
		var config model.YAMLConfig
		err := yaml.Unmarshal([]byte(input), &config)
		gt.NoError(t, err)
		gt.V(t, config["MY_VAR"].Value).NotNil()
		gt.V(t, *config["MY_VAR"].Value).Equal("structured value")
		gt.V(t, len(config["MY_VAR"].Refs)).Equal(1)
		gt.V(t, config["MY_VAR"].Refs[0]).Equal("OTHER_VAR")
	})

	t.Run("with profile", func(t *testing.T) {
		input := `
API_URL:
  value: "https://api.example.com"
  profile:
    dev: "http://localhost:8080"
    staging: "https://staging-api.example.com"
`
		var config model.YAMLConfig
		err := yaml.Unmarshal([]byte(input), &config)
		gt.NoError(t, err)
		gt.V(t, config["API_URL"].Value).NotNil()
		gt.V(t, *config["API_URL"].Value).Equal("https://api.example.com")
		gt.V(t, len(config["API_URL"].Profile)).Equal(2)

		devProfile := config["API_URL"].Profile["dev"]
		gt.V(t, devProfile).NotNil()
		gt.V(t, devProfile.Value).NotNil()
		gt.V(t, *devProfile.Value).Equal("http://localhost:8080")
	})
}
