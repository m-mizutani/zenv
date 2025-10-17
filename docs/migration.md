# Migration Guide: zenv v1 to v2

This guide helps you migrate from zenv v1 to v2. **v2 encourages using `.env.yaml` files** as the primary configuration format, offering cleaner syntax and more features than traditional `.env` files.

## ⚠️ Important: What's Changed in v2

| Feature | v1 (.env) | v2 (.env.yaml) | Migration |
|---------|-----------|----------------|-----------|
| **Basic Variables** | `KEY=value` | `KEY: "value"` | Direct migration |
| **File Content (`&` prefix)** | `KEY=&/path/file` | `KEY:`<br>`  file: "/path/file"` | Use object format |
| **Command Execution (backticks)** | `KEY=`\`command\`` | `KEY:`<br>`  command: ["cmd", "arg"]` | Use object format |
| **Variable Replacement (`%`)** | `%VAR%` | `{{ .VAR }}` | Use templates or command refs |
| **Secret Management (Keychain)** | ✅ Built-in | ❌ **Removed** | Use external tools |

## Quick Start: Migrate to .env.yaml

v2 encourages using `.env.yaml` as your primary configuration:
- **Standard YAML syntax** for simple variables
- **Object format** for advanced features
- **Better readability** and maintainability
- **Clean, structured format**

## Migration Examples

### Basic Variables

#### v1 (.env)
```bash
POSTGRES_DB=dev_db
POSTGRES_USER=testuser
API_KEY=secret123
PORT=3000
```

#### v2 (.env.yaml) - Recommended
```yaml
POSTGRES_DB: "dev_db"
POSTGRES_USER: "testuser"
API_KEY: "secret123"
PORT: "3000"
```

**✅ Migration**: Use YAML key-value syntax with colons.

### File Content Loading

#### v1 (.env with `&` prefix)
```bash
SECRET_KEY=&/path/to/secret.txt
SSL_CERT=&/etc/ssl/cert.pem
```

#### v2 (.env.yaml)
```yaml
SECRET_KEY:
  file: "/path/to/secret.txt"

SSL_CERT:
  file: "/etc/ssl/cert.pem"
```

### Command Execution

#### v1 (.env with backticks)
```bash
GIT_HASH=`git rev-parse HEAD`
BUILD_DATE=`date +%Y%m%d`
```

#### v2 (.env.yaml)
```yaml
GIT_HASH:
  command: ["git", "rev-parse", "HEAD"]

BUILD_DATE:
  command: ["date", "+%Y%m%d"]
```

### Variable Replacement

#### v1 (.env with `%` prefix)
```bash
DATABASE_URL=postgresql://%DB_USER%:%DB_PASS%@%DB_HOST%/mydb
```

#### v2 (.env.yaml with templates)
```yaml
DB_USER: "admin"
DB_PASS: "secret"
DB_HOST: "localhost"

DATABASE_URL:
  value: "postgresql://{{ .DB_USER }}:{{ .DB_PASS }}@{{ .DB_HOST }}/mydb"
  refs: ["DB_USER", "DB_PASS", "DB_HOST"]
```

### Secret Management

#### v1 (Built-in Keychain)
```bash
zenv secret write @aws AWS_SECRET_ACCESS_KEY
AWS_SECRET_ACCESS_KEY=%@aws.AWS_SECRET_ACCESS_KEY  # in .env
```

#### v2 (Use External Tools)
```yaml
# Load from external secret manager or file
AWS_SECRET_ACCESS_KEY:
  file: "/run/secrets/aws_key"  # Docker secrets, K8s secrets, etc.
```

**❌ Migration**: Use external secret management (Vault, AWS Secrets Manager, K8s Secrets)

## Installation

```bash
# v1 (old)
go install github.com/m-mizutani/zenv@latest

# v2 (new - requires /v2 suffix)
go install github.com/m-mizutani/zenv/v2@latest
```

## File Format Recommendation

While v2 still supports `.env` files, **we recommend migrating to `.env.yaml`**:

```bash
# Rename your file
mv .env .env.yaml

# Update the syntax (use YAML format)
# Before: KEY=value
# After:  KEY: "value"
```

## Command Usage

```bash
# v1 (loads .env)
zenv myapp

# v2 (loads .env.yaml) - Recommended
zenv myapp  # Auto-loads .env.yaml if present

# v2 with explicit file
zenv -c config.yaml myapp
```

## Complete Migration Example

### v1 Project (.env)
```bash
# .env
APP_NAME=myapp
PORT=3000
DEBUG=true

# Advanced features
DB_PASSWORD=&/secrets/db.txt
GIT_HASH=`git rev-parse --short HEAD`
DATABASE_URL=postgresql://%DB_USER%:%DB_PASSWORD%@localhost/mydb
DB_USER=admin
```

### v2 Project (.env.yaml) - Recommended
```yaml
# Simple variables
APP_NAME: "myapp"
PORT: "3000"
DEBUG: "true"
DB_USER: "admin"

# File loading
DB_PASSWORD:
  file: "/secrets/db.txt"

# Command execution
GIT_HASH:
  command: ["git", "rev-parse", "--short", "HEAD"]

# Template for complex values
DATABASE_URL:
  value: "postgresql://{{ .DB_USER }}:{{ .DB_PASSWORD }}@localhost/mydb"
  refs: ["DB_USER", "DB_PASSWORD"]
```

## Additional Features

### Aliases (Reference Other Variables)
```yaml
# Define primary DB
PRIMARY_DB: "postgresql://primary.db.com/myapp"

# Create an alias
DATABASE_URL:
  alias: "PRIMARY_DB"
```

### Templates (Combine Variables)
```yaml
# Define components
DB_USER: "admin"
DB_HOST: "localhost"

# Combine with template
DATABASE_URL:
  value: "postgresql://{{ .DB_USER }}@{{ .DB_HOST }}/myapp"
  refs: ["DB_USER", "DB_HOST"]
```

### Conditional Configuration
```yaml
USE_STAGING: "true"

API_ENDPOINT:
  value: "{{ if .USE_STAGING }}https://staging.api.example.com{{ else }}https://api.example.com{{ end }}"
  refs: ["USE_STAGING"]
```

## Migration Checklist

1. ☐ Install v2: `go install github.com/m-mizutani/zenv/v2@latest`
2. ☐ Rename `.env` to `.env.yaml`
3. ☐ Update syntax: `KEY=value` → `KEY: "value"`
4. ☐ Convert file loading: `KEY=&/path` → `KEY:\n  file: "/path"`
5. ☐ Convert commands: `KEY=`\`cmd arg\`` → `KEY:\n  command: ["cmd", "arg"]`
6. ☐ Convert templates: `%VAR%` → `{{ .VAR }}`
7. ☐ Test: `zenv myapp`

## Quick Reference

```bash
# Remove old version (optional)
go clean -modcache

# Install v2 (MUST use /v2 suffix)
go install github.com/m-mizutani/zenv/v2@latest
```

**Critical Note**: The `/v2` suffix is **mandatory**. Using `github.com/m-mizutani/zenv@latest` will always install v1, even after v2 is released.

### Step 2: Test Current Setup

Before making changes, test your current `.env` files with v2:

```bash
# Test current .env file
zenv -e .env

# Test with your usual command
zenv -e .env your-command
```

### Step 3: Review Variable Precedence

If you rely on specific precedence behavior, verify it still works:

```bash
# Test precedence with multiple sources
export TEST_VAR=system_value
echo "TEST_VAR=env_value" > test.env
zenv -e test.env TEST_VAR=inline_value
```

Expected output order (inline_value should be used):
```
TEST_VAR=inline_value [inline]
```

### Step 4: Consider YAML Migration (Optional)

For complex configurations, consider migrating to YAML:

#### Before (.env)
```env
DATABASE_URL=postgresql://localhost/mydb
API_KEY_FILE=/path/to/api/key
GIT_COMMIT=$(git rev-parse HEAD)
```

#### After (.env.yaml)
```yaml
DATABASE_URL: "postgresql://localhost/mydb"

API_KEY:
  file: "/path/to/api/key"

GIT_COMMIT:
  command: ["git", "rev-parse", "HEAD"]
```

### Step 5: Update Scripts and CI/CD

Update any scripts or CI/CD configurations:

```bash
# Before
zenv -e .env.production deploy.sh

# After (same command, but new features available)
zenv -e .env.production -c config.yaml deploy.sh
```

## Real-World Migration Scenarios

### Scenario 1: Simple .env Usage ✅ **COMPATIBLE**

**v1 Setup:**
```bash
# .env
POSTGRES_DB=myapp_dev
POSTGRES_USER=developer

zenv -e .env psql -h localhost
```

**v2 Migration:**
```bash
# Same .env file works unchanged
POSTGRES_DB=myapp_dev
POSTGRES_USER=developer

zenv -e .env psql -h localhost  # Identical command
```

**Result**: ✅ No changes needed.

### Scenario 2: Using v1's Special Syntax ⚠️ **REQUIRES MIGRATION**

**v1 Setup:**
```bash
# .env with v1 special syntax
API_BASE_URL=https://api.myapp.com
SECRET_KEY=&/etc/myapp/secret.key
GIT_COMMIT=`git rev-parse HEAD`
DATABASE_PASSWORD=%VAULT_DATABASE_PASS
DATABASE_URL=postgresql://%DB_USER%:%DB_PASS%@%DB_HOST%:5432/myapp
```

**v2 Migration Options:**

#### Option A: Migrate to YAML
```bash
# .env (keep simple values)
API_BASE_URL=https://api.myapp.com

# .env.yaml (migrate complex values)
SECRET_KEY:
  file: "/etc/myapp/secret.key"

GIT_COMMIT:
  command: ["git", "rev-parse", "HEAD"]

# Variable replacement with template
DATABASE_PASSWORD:
  value: "{{ .VAULT_DATABASE_PASS }}"
  refs: ["VAULT_DATABASE_PASS"]

# Complex variable replacement
DATABASE_URL:
  value: "postgresql://{{ .DB_USER }}:{{ .DB_PASS }}@{{ .DB_HOST }}:5432/myapp"
  refs: ["DB_USER", "DB_PASS", "DB_HOST"]
```

#### Option B: Use shell expansion (alternative)
```bash
# Use shell to handle complex variables
export DATABASE_PASSWORD=$VAULT_DATABASE_PASS
export SECRET_KEY=$(cat /etc/myapp/secret.key)
export GIT_COMMIT=$(git rev-parse HEAD)

zenv -e .env myapp
```

### Scenario 3: macOS Keychain Secrets ❌ **CANNOT MIGRATE**

**v1 Setup:**
```bash
# Store secrets in Keychain
zenv secret write @myapp DATABASE_PASSWORD
zenv secret write @myapp API_KEY

# .env
DATABASE_URL=postgresql://user:@myapp.DATABASE_PASSWORD@localhost/db
API_KEY=%@myapp.API_KEY
```

**v2 Migration - NO DIRECT PATH:**

#### Option A: External Secret Management
```bash
# Use a secret management tool
export DATABASE_PASSWORD=$(vault kv get -field=password myapp/db)
export API_KEY=$(vault kv get -field=api_key myapp/external)

# .env
DATABASE_URL=postgresql://user:password@localhost/db

zenv -e .env myapp
```

#### Option B: File-Based Secrets
```bash
# Store secrets in files (ensure proper permissions)
echo "secret_password" > /etc/myapp/db_password  # chmod 600
echo "api_key_value" > /etc/myapp/api_key       # chmod 600

# .env.yaml
DATABASE_PASSWORD:
  file: "/etc/myapp/db_password"

API_KEY:
  file: "/etc/myapp/api_key"
```

#### Option C: Keep Using v1 for Secret Management
```bash
# Continue using v1 for projects with extensive secret management
go install github.com/m-mizutani/zenv@latest  # Install v1
zenv secret list  # Continue using v1 features
```

### Scenario 4: Complex CI/CD Pipeline ⚠️ **MIXED COMPATIBILITY**

**v1 Setup:**
```bash
# CI environment with dynamic values
# .env
BUILD_TIME=`date -u +%Y-%m-%dT%H:%M:%SZ`
GIT_BRANCH=`git rev-parse --abbrev-ref HEAD`
DOCKER_TAG=${GIT_BRANCH}-`git rev-parse --short HEAD`
SECRET_TOKEN=%@ci.SECRET_TOKEN

zenv -e .env docker build -t myapp:$DOCKER_TAG .
```

**v2 Migration Strategy:**

#### Option A: Hybrid Approach
```bash
# Generate dynamic values with shell
export BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
export DOCKER_TAG="${GIT_BRANCH}-$(git rev-parse --short HEAD)"

# .env.yaml (for command-based values)
GIT_BRANCH:
  command: ["git", "rev-parse", "--abbrev-ref", "HEAD"]

# Use external secret management for CI
export SECRET_TOKEN=$CI_SECRET_TOKEN  # from CI environment

zenv -e .env -c .env.yaml docker build -t myapp:$DOCKER_TAG .
```

#### Option B: Pure Shell Approach
```bash
# Handle all dynamic values in shell
export BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
export GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
export DOCKER_TAG="${GIT_BRANCH}-$(git rev-parse --short HEAD)"
export SECRET_TOKEN=$CI_SECRET_TOKEN

# Simple .env for static values
# .env
APP_ENV=production
LOG_LEVEL=info

zenv -e .env docker build -t myapp:$DOCKER_TAG .
```

## Troubleshooting

### Issue: Variables Not Loading as Expected

**Solution**: Check the new precedence order and use `zenv` without a command to inspect loaded variables.

```bash
zenv -e .env -c config.yaml
```

### Issue: Command Execution in YAML Not Working

**Solution**: Ensure proper YAML syntax for commands:

```yaml
# Correct - command as array (inline format recommended)
GIT_BRANCH:
  command: ["git", "rev-parse", "--abbrev-ref", "HEAD"]

# Also correct - multi-line array format
GIT_BRANCH:
  command:
    - git
    - rev-parse
    - --abbrev-ref
    - HEAD

# Incorrect - command as string
GIT_BRANCH:
  command: "git rev-parse --abbrev-ref HEAD"
```

### Issue: File Loading Not Working

**Solution**: Check file paths and permissions:

```yaml
SECRET:
  file: "/absolute/path/to/file"  # Use absolute paths
```

## Best Practices for v2

1. **Use YAML for complex configurations** that need file reading or command execution
2. **Keep .env for simple key-value pairs** that don't change often
3. **Use inline variables** for one-off overrides
4. **Test precedence** by running `zenv` without a command to see all variables
5. **Use descriptive variable names** that clearly indicate their purpose

## Getting Help

If you encounter issues during migration:

1. **Check the current behavior**: Run `zenv` without arguments to see all loaded variables
2. **Review the precedence**: Ensure your expected precedence order matches v2's behavior
3. **Test incrementally**: Migrate one configuration file at a time
4. **Use verbose output**: Check error messages for detailed information

## Should I Migrate to v2?

Use this decision matrix to determine if v2 is right for you:

### ✅ **Migrate to v2 if you:**
- Use only basic .env files with simple KEY=value pairs
- Want better YAML configuration support with structured settings
- Need clear environment variable precedence
- Can migrate variable replacement (`%VARIABLE`) to Go templates
- Can migrate file loading (`&` prefix) and command execution (backticks) to YAML
- Don't require built-in macOS Keychain integration

### ❌ **Stay with v1 if you:**
- Heavily use macOS Keychain integration (`@namespace`) for secret management
- Have extensive secret management workflows with `zenv secret` commands
- Need backwards compatibility with existing scripts using v1 syntax
- Cannot migrate to external secret management solutions

### 🔄 **Consider hybrid approach if you:**
- Want to test v2 for new projects
- Can gradually migrate simple configurations
- Have mixed complexity environments
- Want to benefit from v2's improved architecture

### Alternative: Stay with v1

v1 continues to be maintained and remains suitable for:
- **Secret management**: Built-in macOS Keychain integration with namespace support
- **Legacy projects**: Existing scripts and workflows that depend on v1 syntax
- **Simple variable replacement**: Quick `%VARIABLE` syntax without template configuration

**There's no requirement to migrate.** Choose the version that best fits your workflow:
- **v1**: For projects requiring built-in secret management and established workflows
- **v2**: For configuration-driven environments with YAML support and template capabilities

For additional support, see:
- [v1 Documentation](https://github.com/m-mizutani/zenv/blob/main/README.md) for v1 features
- [v2 Documentation](../README.md) for v2 features  
- [GitHub Issues](https://github.com/m-mizutani/zenv/issues) for bug reports and questions