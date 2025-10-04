# zenv [![CI](https://github.com/m-mizutani/zenv/actions/workflows/test.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/test.yml) [![Security Scan](https://github.com/m-mizutani/zenv/actions/workflows/gosec.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/gosec.yml) [![Vuln scan](https://github.com/m-mizutani/zenv/actions/workflows/trivy.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/trivy.yml) <!-- omit in toc -->

> **⚠️ Breaking Changes in v2**  
> Version 2 introduces significant changes including new TOML configuration support, modified CLI options, and updated variable precedence. See [Migration Guide](docs/migration.md) for detailed migration instructions.

`zenv` is enhanced `env` command to manage environment variables in CLI.

```toml
# .env.toml - Powerful environment variable management
DB_USER = "admin"
DB_HOST = "localhost"

[DB_PASSWORD]
file = "/path/to/db_secret"  # Load from file

[API_KEY]
file = "/path/to/api_key"

[DATABASE_URL]
value = "postgresql://{{ .DB_USER }}:{{ .DB_PASSWORD }}@{{ .DB_HOST }}/mydb"
refs = ["DB_USER", "DB_PASSWORD", "DB_HOST"]  # Build from variables

[CONFIG_DATA]
command = ["curl", "-H", "Authorization: Bearer {{ .API_KEY }}", "https://api.example.com/config"]
refs = ["API_KEY"]  # Fetch data from API

[DATABASE_URL.profile]
dev = "sqlite://local.db"  # Override with profile
```

- Load environment variables from multiple sources:
    - `.env` files with static values, file content reading, and command execution
    - TOML configuration files with advanced features
    - Inline environment variable specification (KEY=value format)
    - System environment variables
- Replace command line argument with loaded environment variable
- Variable precedence: System < .env < TOML < Inline (later sources override earlier ones)

## Install <!-- omit in toc -->

```sh
# Install v2 (the /v2 suffix is required)
go install github.com/m-mizutani/zenv/v2@latest

# Note: github.com/m-mizutani/zenv@latest will install v1 even after v2 release
```

## Command Line Options

```sh
zenv [OPTIONS] [ENVIRONMENT_VARIABLES] [COMMAND] [ARGS...]
```

### Options

- `-e, --env FILE`: Load environment variables from .env file (can be specified multiple times)
- `-t, --toml FILE`: Load environment variables from TOML file (can be specified multiple times)
- `-p, --profile NAME`: Select profile from TOML configuration (e.g., dev, staging, prod)

## Basic Usage

### Set by CLI argument

Can set environment variable in same manner with `env` command

```sh
$ zenv POSTGRES_DB=your_local_dev_db psql
```

### Load from `.env` file

Automatically loads `.env` file from current directory. You can also specify custom files with `-e` option.

```sh
$ cat .env
POSTGRES_DB=your_local_db
POSTGRES_USER=test_user
PGDATA=/var/lib/db

$ zenv psql -h localhost -p 15432
# connecting to your_local_db on localhost:15432 as test_user

# Or specify custom .env file
$ zenv -e production.env psql
```

### Load from TOML configuration files

```sh
$ cat .env.toml
DATABASE_URL = "postgresql://localhost/mydb"
PORT = "3000"

$ zenv -t .env.toml myapp
# myapp runs with DATABASE_URL and PORT set from .env.toml
```

### Multiple files and precedence

You can load from multiple sources. Variables are merged with the following precedence (later sources override earlier ones):

1. System environment variables
2. `.env` files (in order specified)
3. TOML files (in order specified)  
4. Inline variables (KEY=value)

```sh
# Load from multiple sources
$ zenv -e base.env -e override.env -t config.toml DATABASE_URL=sqlite://local.db myapp
```

### List environment variables

Run without a command to see all loaded environment variables:

```sh
$ zenv
DATABASE_URL=postgresql://localhost/mydb [.toml]
PORT=3000 [.toml]
API_SECRET=secret_from_file [.toml]
CURRENT_BRANCH=main [.toml]
PATH=/usr/bin:/bin [system]
...

# List with specific configuration
$ zenv -e production.env -t config.toml
```

## TOML Configuration Format

### Basic Usage

TOML files use standard key-value pairs for environment variables:

```toml
DATABASE_URL = "postgresql://localhost/mydb"
API_KEY = "secret-key-123"
PORT = "3000"
DEBUG = "true"
```

### Advanced Features

For capabilities beyond simple strings, use the section format `[VARIABLE_NAME]`:

#### File Content Reading
Load values from files:
```toml
[SECRET_KEY]
file = "/path/to/secret/file"

[SSL_CERT]
file = "/etc/ssl/certs/app.pem"
```

#### Command Execution
Execute commands and use their output:
```toml
[GIT_COMMIT]
command = ["git", "rev-parse", "HEAD"]

[BUILD_TIME]
command = ["date", "+%Y-%m-%d"]
```

#### Variable References (Alias)
Reference other variables or system environment variables:
```toml
[APP_HOME]
alias = "HOME"  # References system environment variable

[PRIMARY_DB]
value = "postgresql://primary.example.com/maindb"

[DATABASE_URL]
alias = "PRIMARY_DB"  # References another TOML variable
```

#### Templates (Variable Interpolation)
Combine multiple variables using Go's text/template syntax by adding `refs`:
```toml
[DB_USER]
value = "admin"

[DB_HOST]
value = "localhost"

[DB_NAME]
value = "myapp"

# Simple interpolation
[DATABASE_URL]
value = "postgresql://{{ .DB_USER }}@{{ .DB_HOST }}/{{ .DB_NAME }}"
refs = ["DB_USER", "DB_HOST", "DB_NAME"]

# Conditional logic
[USE_STAGING]
value = "true"

[API_ENDPOINT]
value = "{{ if eq .USE_STAGING \"true\" }}https://staging.api.example.com{{ else }}https://api.example.com{{ end }}"
refs = ["USE_STAGING"]
```

**Template Features:**
- Use `{{ .VAR_NAME }}` to reference variables
- Support conditional logic: `{{ if }}`/`{{ else }}`/`{{ end }}`
- Can reference system environment variables, .env variables, and TOML variables
- Both `value` and `command` support templates with `refs`

#### Profile Support
Manage different configurations for different environments (dev, staging, prod, etc.):

```toml
# Basic profile usage - inline table format
[API_URL]
value = "https://api.example.com"
profile = { dev = "http://localhost:8080", staging = "https://staging-api.example.com" }

# Dotted key format within section
[DATABASE_HOST]
value = "prod-db.example.com"
profile.dev = "localhost"
profile.staging = "staging-db.example.com"

# Unset variable in specific profile (empty object)
[DEBUG_MODE]
value = "false"
profile = { dev = "true", prod = {} }  # prod profile will unset DEBUG_MODE

# Profile with different value types
[SSL_CERT]
file = "/etc/ssl/prod.pem"

[SSL_CERT.profile.dev]
value = "-----BEGIN CERTIFICATE-----\ndev-cert\n-----END CERTIFICATE-----"

[SSL_CERT.profile.staging]
file = "/etc/ssl/staging.pem"
```

To use a specific profile, run:
```bash
# Use dev profile
zenv -t config.toml -p dev myapp

# Use staging profile
zenv -t config.toml --profile staging deploy
```

## Configuration Rules

**Value Types** (only one can be specified per variable):
- `value`: Direct string value (becomes a template when used with `refs`)
- `file`: Read content from a file path
- `command`: Execute command and use output
- `alias`: Reference another variable

**Additional Options:**
- `refs`: List of variables to reference in templates (used with `value` or `command`)
- `profile`: Environment-specific overrides (dev, staging, prod, etc.)

**Important Notes:**
- Circular references (e.g., A→B→A) will result in an error
- Profile values override defaults when selected with `-p/--profile`
- Empty profile `{}` unsets the variable for that environment

## Migration from v1 to v2

For detailed migration instructions, see [Migration Guide](docs/migration.md).

### Quick Migration Summary

1. **Update installation**: `go install github.com/m-mizutani/zenv/v2@latest`
2. **Review precedence**: New order is System < .env < TOML < Inline
3. **Test existing setup**: Most `.env` usage continues to work unchanged
4. **Consider TOML**: Optionally migrate complex configurations to TOML for advanced features

## License

Apache License 2.0
