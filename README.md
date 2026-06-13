# zenv [![CI](https://github.com/m-mizutani/zenv/actions/workflows/test.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/test.yml) [![Security Scan](https://github.com/m-mizutani/zenv/actions/workflows/gosec.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/gosec.yml) [![Vuln scan](https://github.com/m-mizutani/zenv/actions/workflows/trivy.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/trivy.yml) <!-- omit in toc -->

`zenv` is enhanced `env` command to manage environment variables in CLI.

```yaml
# .env.yaml - Powerful environment variable management
DB_USER: "admin"
DB_HOST: "localhost"

DB_PASSWORD:
  file: "/path/to/db_secret"  # Load from file

API_KEY:
  file: "/path/to/api_key"

DATABASE_URL:
  value: "postgresql://{{ .DB_USER }}:{{ .DB_PASSWORD }}@{{ .DB_HOST }}/mydb"
  refs:
    - DB_USER
    - DB_PASSWORD
    - DB_HOST  # Build from variables
  profile:
    dev: "sqlite://local.db"  # Override with profile

CONFIG_DATA:
  command:
    - curl
    - -H
    - "Authorization: Bearer {{ .API_KEY }}"
    - https://api.example.com/config
  refs:
    - API_KEY  # Fetch data from API
```

- Load environment variables from multiple sources:
    - `.env` files with static values, file content reading, and command execution
    - YAML configuration files with advanced features
    - Inline environment variable specification (KEY=value format)
    - System environment variables
- Replace command line argument with loaded environment variable
- Variable precedence: System < .env < YAML < Inline (later sources override earlier ones)

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
- `-c, --config FILE`: Load environment variables from YAML file (can be specified multiple times)
- `-p, --profile NAME`: Select profile from YAML configuration (e.g., dev, staging, prod)
- `--redact`: Mask `secret: true` values in the child process's stdout/stderr. Also enabled by setting `ZENV_REDACT=1`. Disabled by default — see [Secret Redaction](#secret-redaction).

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

### Load from YAML configuration files

Both `.env.yaml` and `.env.yml` file extensions are supported. If both files exist, they will be automatically merged.

```sh
$ cat .env.yaml
DATABASE_URL: "postgresql://localhost/mydb"
PORT: "3000"

$ zenv -c .env.yaml myapp
# myapp runs with DATABASE_URL and PORT set from .env.yaml (or .env.yml)
```

### Multiple files and precedence

You can load from multiple sources. Variables are merged with the following precedence (later sources override earlier ones):

1. System environment variables
2. `.env` files (in order specified)
3. YAML files (in order specified)
4. Inline variables (KEY=value)

```sh
# Load from multiple sources
$ zenv -e base.env -e override.env -c config.yaml DATABASE_URL=sqlite://local.db myapp
```

### List environment variables

Run without a command to see all loaded environment variables:

```sh
$ zenv
DATABASE_URL=postgresql://localhost/mydb [.yaml]
PORT=3000 [.yaml]
API_SECRET=secret_from_file [.yaml]
CURRENT_BRANCH=main [.yaml]
PATH=/usr/bin:/bin [system]
...

# List with specific configuration
$ zenv -e production.env -c config.yaml
```

## YAML Configuration Format

### Basic Usage

YAML files use standard key-value pairs for environment variables:

```yaml
DATABASE_URL: "postgresql://localhost/mydb"
API_KEY: "secret-key-123"
PORT: "3000"
DEBUG: "true"
```

### Advanced Features

For capabilities beyond simple strings, use the object format:

#### File Content Reading
Load values from files:
```yaml
SECRET_KEY:
  file: "/path/to/secret/file"

SSL_CERT:
  file: "/etc/ssl/certs/app.pem"
```

#### Command Execution
Execute commands and use their output:
```yaml
GIT_COMMIT:
  command:
    - git
    - rev-parse
    - HEAD

BUILD_TIME:
  command:
    - date
    - "+%Y-%m-%d"
```

#### Secret Manager References (AWS / GCP)
Fetch values directly from AWS Secrets Manager or GCP Secret Manager. Each
reference is a single path string; authentication uses the standard credential
chain of the respective SDK (AWS: environment variables / shared config / IAM
role, GCP: Application Default Credentials). No credential options are
configured in `zenv` itself.

AWS secrets **must be referenced by their full ARN** (a bare secret name is
rejected) so the target region and account are always explicit.

```yaml
# AWS Secrets Manager — full ARN required; the region is taken from the ARN
DB_PASSWORD:
  aws_secret: "arn:aws:secretsmanager:ap-northeast-1:123456789012:secret:prod/db/password"
  secret: true

# GCP Secret Manager (full resource path including the version)
API_TOKEN:
  gcp_secret: "projects/my-project/secrets/api-token/versions/latest"
  secret: true
```

When the stored secret is a JSON document, append `#<field>` to extract a single
string field:

```yaml
# Secret value: {"host":"db.example.com","password":"s3cret"}
DB_HOST:
  aws_secret: "arn:aws:secretsmanager:ap-northeast-1:123456789012:secret:prod/db/conn#host"      # -> db.example.com

DB_PASSWORD:
  aws_secret: "arn:aws:secretsmanager:ap-northeast-1:123456789012:secret:prod/db/conn#password"  # -> s3cret
  secret: true
```

`aws_secret` / `gcp_secret` also support `refs`, so the path itself can
interpolate other variables:

```yaml
REGION: "ap-northeast-1"

DB_PASSWORD:
  aws_secret: "arn:aws:secretsmanager:{{ .REGION }}:123456789012:secret:prod/db/password"
  refs:
    - REGION
```

Notes:
- AWS: the reference must be a full ARN (`arn:aws:secretsmanager:...`); a bare
  secret name is rejected. A version-pinned secret is selected by passing a
  version-qualified ARN; otherwise `AWSCURRENT` is used.
- GCP: the path must include `/versions/<version>` (use `latest` for the most
  recent version). A path without a version defaults to `latest`.
- These value sources are available in both YAML and HCL configuration files.

#### Variable References (Alias)
Reference other variables or system environment variables:
```yaml
APP_HOME:
  alias: "HOME"  # References system environment variable

PRIMARY_DB: "postgresql://primary.example.com/maindb"

DATABASE_URL:
  alias: "PRIMARY_DB"  # References another YAML variable
```

#### Templates (Variable Interpolation)
Combine multiple variables using Go's text/template syntax by adding `refs`:
```yaml
DB_USER: "admin"

DB_HOST: "localhost"

DB_NAME: "myapp"

# Simple interpolation
DATABASE_URL:
  value: "postgresql://{{ .DB_USER }}@{{ .DB_HOST }}/{{ .DB_NAME }}"
  refs:
    - DB_USER
    - DB_HOST
    - DB_NAME

# Conditional logic
USE_STAGING: "true"

API_ENDPOINT:
  value: "{{ if eq .USE_STAGING \"true\" }}https://staging.api.example.com{{ else }}https://api.example.com{{ end }}"
  refs:
    - USE_STAGING
```

**Template Features:**
- Use `{{ .VAR_NAME }}` to reference variables
- Support conditional logic: `{{ if }}`/`{{ else }}`/`{{ end }}`
- Can reference system environment variables, .env variables, and YAML variables
- Both `value` and `command` support templates with `refs`

#### Profile Support
Manage different configurations for different environments (dev, staging, prod, etc.):

```yaml
# Basic profile usage
API_URL: "https://api.example.com"

DATABASE_HOST: "prod-db.example.com"

# Unset variable in specific profile (null value)
DEBUG_MODE: "false"

# Profile with different value types
SSL_CERT:
  file: "/etc/ssl/prod.pem"

profiles:
  dev:
    API_URL: "http://localhost:8080"
    DATABASE_HOST: "localhost"
    DEBUG_MODE: "true"
    SSL_CERT: "-----BEGIN CERTIFICATE-----\ndev-cert\n-----END CERTIFICATE-----"

  staging:
    API_URL: "https://staging-api.example.com"
    DATABASE_HOST: "staging-db.example.com"
    SSL_CERT:
      file: "/etc/ssl/staging.pem"

  prod:
    DEBUG_MODE: null  # Unset DEBUG_MODE in prod
```

To use a specific profile, run:
```bash
# Use dev profile
zenv -c config.yaml -p dev myapp

# Use staging profile
zenv -c config.yaml --profile staging deploy
```

#### Secret Redaction
Add `secret: true` to mark a variable as sensitive. In the variable list (`zenv` with no command) it is always masked with `*`:
```yaml
DB_PASSWORD:
  file: "/path/to/db_secret"
  secret: true
```

Profile values inherit `secret: true` from their base configuration.

##### Redacting secrets in the child process's output

By default, `zenv` attaches the child process's `stdin`/`stdout`/`stderr` directly to the parent terminal so that the child sees a real TTY (preserving color, progress bars, interactive prompts, terminal size, etc.). In this mode the child's output is **not** filtered, so a leaked secret value would appear verbatim.

Pass `--redact` (or set `ZENV_REDACT=1`) to opt into output redaction. Any occurrence of a `secret: true` value in the child's stdout/stderr is replaced with `*****`:

```sh
$ zenv --redact -c .env.yaml my-app
# any prints of $DB_PASSWORD inside my-app appear as ***** in your terminal
```

How `--redact` connects the child's streams:

| Platform | Parent stdout is a TTY | Path                                                                        |
| -------- | ---------------------- | --------------------------------------------------------------------------- |
| Unix     | yes                    | pseudo-terminal (pty) + redactor — child still sees a TTY                  |
| Unix     | no (e.g. `> file`)     | anonymous pipe + redactor — child sees a pipe (color libraries usually off) |
| Windows  | any                    | anonymous pipe + redactor (pty path is not supported)                       |

In the pty path the child's stdout/stderr share a single pty, so any ANSI control sequences will reach a redirected file (`zenv --redact app > out.log`) unless the child itself disables them (e.g. `NO_COLOR=1`).

## Configuration Rules

**Value Types** (only one can be specified per variable):
- `value`: Direct string value (becomes a template when used with `refs`)
- `file`: Read content from a file path
- `command`: Execute command and use output
- `alias`: Reference another variable
- `aws_secret`: Fetch a value from AWS Secrets Manager (`<full_arn>[#<json_key>]`)
- `gcp_secret`: Fetch a value from GCP Secret Manager (`projects/.../versions/<version>[#<json_key>]`)

**Additional Options:**
- `refs`: List of variables to reference in templates (used with `value`, `command`, `aws_secret`, or `gcp_secret`)
- `profiles`: Environment-specific overrides (dev, staging, prod, etc.)

**Important Notes:**
- Circular references (e.g., A→B→A) will result in an error
- Profile values override defaults when selected with `-p/--profile`
- Null profile value unsets the variable for that environment

## Migration from v1 to v2

For detailed migration instructions, see [Migration Guide](docs/migration.md).

### Quick Migration Summary

1. **Update installation**: `go install github.com/m-mizutani/zenv/v2@latest`
2. **Review precedence**: New order is System < .env < YAML < Inline
3. **Test existing setup**: Most `.env` usage continues to work unchanged
4. **Consider YAML**: Optionally migrate complex configurations to YAML for advanced features

## License

Apache License 2.0
