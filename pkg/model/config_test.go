package model_test

import (
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func TestTOMLConfigUnmarshalTOML_SimpleFormat(t *testing.T) {
	// シンプル形式のみのテスト
	input := `
DATABASE_URL = "postgres://localhost/mydb"
API_KEY = "secret123"
PORT = "3000"
`

	var config model.TOMLConfig
	_, err := toml.Decode(input, &config)
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

func TestTOMLConfigUnmarshalTOML_SectionFormat(t *testing.T) {
	// セクション形式のみのテスト
	input := `
[DATABASE_URL]
value = "postgres://localhost/mydb"

[SSL_CERT]
file = "/path/to/cert.pem"

[AUTH_TOKEN]
command = "aws"
args = ["secretsmanager", "get-secret-value"]
`

	var config model.TOMLConfig
	_, err := toml.Decode(input, &config)
	gt.NoError(t, err)

	// DATABASE_URL
	gt.V(t, config["DATABASE_URL"].Value).NotNil()
	gt.V(t, *config["DATABASE_URL"].Value).Equal("postgres://localhost/mydb")

	// SSL_CERT
	gt.V(t, config["SSL_CERT"].File).NotNil()
	gt.V(t, *config["SSL_CERT"].File).Equal("/path/to/cert.pem")

	// AUTH_TOKEN
	gt.V(t, config["AUTH_TOKEN"].Command).NotNil()
	gt.V(t, *config["AUTH_TOKEN"].Command).Equal("aws")
	gt.V(t, len(config["AUTH_TOKEN"].Args)).Equal(2)
	gt.V(t, config["AUTH_TOKEN"].Args[0]).Equal("secretsmanager")
	gt.V(t, config["AUTH_TOKEN"].Args[1]).Equal("get-secret-value")
}

func TestTOMLConfigUnmarshalTOML_MixedFormat(t *testing.T) {
	// 混在形式のテスト
	// TOML仕様: セクション定義後はそのセクション内の設定となるため、
	// トップレベルのキー・バリューペアは全てセクション定義の前に配置する必要がある
	input := `
# シンプル形式（トップレベル）
PORT = "3000"
ENV = "development"
API_KEY = "secret123"

# セクション形式
[DATABASE_URL]
value = "postgres://localhost/mydb"

[SSL_CERT]
file = "/path/to/cert.pem"

[AUTH_TOKEN]
command = "aws"
args = ["secretsmanager", "get-secret-value"]
`

	var config model.TOMLConfig
	_, err := toml.Decode(input, &config)
	gt.NoError(t, err)

	// Check the actual keys in the config
	gt.V(t, len(config)).Equal(6) // PORT, ENV, API_KEY, DATABASE_URL, SSL_CERT, AUTH_TOKEN

	// シンプル形式
	gt.V(t, config["PORT"].Value).NotNil()
	gt.V(t, *config["PORT"].Value).Equal("3000")
	gt.V(t, config["ENV"].Value).NotNil()
	gt.V(t, *config["ENV"].Value).Equal("development")
	gt.V(t, config["API_KEY"].Value).NotNil()
	gt.V(t, *config["API_KEY"].Value).Equal("secret123")

	// セクション形式
	gt.V(t, config["DATABASE_URL"].Value).NotNil()
	gt.V(t, *config["DATABASE_URL"].Value).Equal("postgres://localhost/mydb")
	gt.V(t, config["SSL_CERT"].File).NotNil()
	gt.V(t, *config["SSL_CERT"].File).Equal("/path/to/cert.pem")
	gt.V(t, config["AUTH_TOKEN"].Command).NotNil()
	gt.V(t, *config["AUTH_TOKEN"].Command).Equal("aws")
}

func TestTOMLConfigUnmarshalTOML_InvalidType(t *testing.T) {
	// 数値型はサポートしない
	input := `
INVALID = 123
`

	var config model.TOMLConfig
	_, err := toml.Decode(input, &config)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("unsupported type")
}

func TestTOMLConfigUnmarshalTOML_InvalidSectionFormat(t *testing.T) {
	// セクション形式で複数のタイプを指定（無効）
	input := `
[INVALID]
value = "test"
file = "/path/to/file"
`

	var config model.TOMLConfig
	_, err := toml.Decode(input, &config)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("multiple value types specified")
}

func TestTOMLConfigUnmarshalTOML_ComplexTypes(t *testing.T) {
	// より複雑なセクション形式のテスト
	input := `
# エイリアスのテスト
[BASE_URL]
value = "https://api.example.com"

[API_ENDPOINT]
alias = "BASE_URL"

# テンプレートのテスト
[GREETING]
template = "Hello, {{.USER}}!"
refs = ["USER"]

[USER]
value = "Alice"
`

	var config model.TOMLConfig
	_, err := toml.Decode(input, &config)
	gt.NoError(t, err)

	// BASE_URL
	gt.V(t, config["BASE_URL"].Value).NotNil()
	gt.V(t, *config["BASE_URL"].Value).Equal("https://api.example.com")

	// API_ENDPOINT (alias)
	gt.V(t, config["API_ENDPOINT"].Alias).NotNil()
	gt.V(t, *config["API_ENDPOINT"].Alias).Equal("BASE_URL")

	// GREETING (template)
	gt.V(t, config["GREETING"].Template).NotNil()
	gt.V(t, *config["GREETING"].Template).Equal("Hello, {{.USER}}!")
	gt.V(t, len(config["GREETING"].Refs)).Equal(1)
	gt.V(t, config["GREETING"].Refs[0]).Equal("USER")

	// USER
	gt.V(t, config["USER"].Value).NotNil()
	gt.V(t, *config["USER"].Value).Equal("Alice")
}
