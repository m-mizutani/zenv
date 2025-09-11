# zenv [![CI](https://github.com/m-mizutani/zenv/actions/workflows/test.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/test.yml) [![Security Scan](https://github.com/m-mizutani/zenv/actions/workflows/gosec.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/gosec.yml) [![Vuln scan](https://github.com/m-mizutani/zenv/actions/workflows/trivy.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/trivy.yml) <!-- omit in toc -->

> **⚠️ Breaking Changes in v2**  
> Version 2 introduces significant changes including new TOML configuration support, modified CLI options, and updated variable precedence. See [Migration Guide](docs/migration.md) for detailed migration instructions.

`zenv` is enhanced `env` command to manage environment variables in CLI.

- Load environment variables from multiple sources:
    - `.env` files with static values, file content reading, and command execution
    - TOML configuration files with advanced features
    - Inline environment variable specification (KEY=value format)
    - System environment variables
- Securely save, generate and get secret values with Keychain, inspired by [envchain](https://github.com/sorah/envchain) (supported only macOS)
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

TOML files provide advanced configuration options including file content reading and command execution.

```sh
$ cat .env.toml
[DATABASE_URL]
value = "postgresql://localhost/mydb"

[API_SECRET]
file = "/path/to/secret.txt"

[CURRENT_BRANCH]
command = "git"
args = ["rev-parse", "--abbrev-ref", "HEAD"]

[MULTILINE_CONFIG]
value = """
line1
line2
line3
"""

$ zenv -t .env.toml myapp
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
API_SECRET=secret_from_file [.toml]
CURRENT_BRANCH=main [.toml]
CUSTOM_VAR=inline_value [inline]
PATH=/usr/bin:/bin [system]
...

# List with specific configuration
$ zenv -e production.env -t config.toml
```

## TOML Configuration Format

TOML files support four types of value specification:

### Static Values
```toml
[VARIABLE_NAME]
value = "static string value"
```

### File Content Reading
```toml
[SECRET_KEY]
file = "/path/to/secret/file"
```

### Command Execution
```toml
[GIT_COMMIT]
command = "git"
args = ["rev-parse", "HEAD"]

[SIMPLE_COMMAND]
command = "date"
```

### Alias (Reference to Another Variable)
```toml
# Reference a system environment variable
[APP_HOME]
alias = "HOME"

# Reference another variable defined in the same TOML file
[PRIMARY_DB]
value = "postgresql://primary.example.com/maindb"

[DATABASE_URL]
alias = "PRIMARY_DB"

# Alias takes precedence over other value types if multiple are specified
[SECONDARY_DB]
value = "postgresql://secondary.example.com/backupdb"

[BACKUP_DB]
alias = "SECONDARY_DB"  # This will be used
value = "ignored_value"  # This will be ignored
```

**Note**: Only one of `value`, `file`, `command`, or `alias` can be specified per variable. Circular references (e.g., A→B→A) will result in an error.

## Migration from v1 to v2

For detailed migration instructions, see [Migration Guide](docs/migration.md).

### Quick Migration Summary

1. **Update installation**: `go install github.com/m-mizutani/zenv/v2@latest`
2. **Review precedence**: New order is System < .env < TOML < Inline
3. **Test existing setup**: Most `.env` usage continues to work unchanged
4. **Consider TOML**: Optionally migrate complex configurations to TOML for advanced features

## License

Apache License 2.0
