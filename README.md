# zenv [![CI](https://github.com/m-mizutani/zenv/actions/workflows/test.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/test.yml) [![Security Scan](https://github.com/m-mizutani/zenv/actions/workflows/gosec.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/gosec.yml) [![Vuln scan](https://github.com/m-mizutani/zenv/actions/workflows/trivy.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/trivy.yml) <!-- omit in toc -->

> **⚠️ Breaking Changes in v2**  
> Version 2 introduces significant changes including new TOML configuration support, modified CLI options, and updated variable precedence. See [Migration Guide](docs/migration.md) for detailed migration instructions.

`zenv` is enhanced `env` command to manage environment variables in CLI.

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

For capabilities beyond simple strings, use the section format:

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
command = "git"
args = ["rev-parse", "HEAD"]

[BUILD_TIME]
command = "date"
args = ["+%Y-%m-%d"]
```

#### Alias (Reference Another Variable)
Reference existing variables:
```toml
# Reference system environment variable
[APP_HOME]
alias = "HOME"

# Reference another TOML variable
[PRIMARY_DB]
value = "postgresql://primary.example.com/maindb"

[DATABASE_URL]
alias = "PRIMARY_DB"
```

#### Template (Combine Variables)
Build values from multiple variables using Go templates:
```toml
[DB_HOST]
value = "localhost"

[DB_PORT]
value = "5432"

[DB_NAME]
value = "myapp"

[DATABASE_URL]
template = "postgresql://user:pass@{{ .DB_HOST }}:{{ .DB_PORT }}/{{ .DB_NAME }}"
refs = ["DB_HOST", "DB_PORT", "DB_NAME"]

# Conditional logic
[USE_STAGING]
value = "true"

[API_ENDPOINT]
template = "{{ if .USE_STAGING }}https://staging.api.example.com{{ else }}https://api.example.com{{ end }}"
refs = ["USE_STAGING"]

# Template can reference aliases and system environment variables
[LOG_PATH]
template = "{{ .HOME }}/logs/{{ .APP_NAME }}.log"
refs = ["HOME", "APP_NAME"]
```

**Note**: 
- Only one of `value`, `file`, `command`, `alias`, or `template` can be specified per variable
- Templates use Go's `text/template` syntax
- The `refs` field is required when using `template` and must list all variables referenced in the template
- Circular references (e.g., A→B→A) will result in an error

## Migration from v1 to v2

For detailed migration instructions, see [Migration Guide](docs/migration.md).

### Quick Migration Summary

1. **Update installation**: `go install github.com/m-mizutani/zenv/v2@latest`
2. **Review precedence**: New order is System < .env < TOML < Inline
3. **Test existing setup**: Most `.env` usage continues to work unchanged
4. **Consider TOML**: Optionally migrate complex configurations to TOML for advanced features

## License

Apache License 2.0
