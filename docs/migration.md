# Migration Guide: zenv v1 to v2

This guide helps you migrate from zenv v1 to v2. **v2 is a complete rewrite** that focuses on configuration files and environment variable management, but **removes some advanced features** from v1.

## ⚠️ Important: What's NOT Available in v2

Before migrating, understand that v2 **removes** several v1 features:

| Feature | v1 | v2 | Migration Path |
|---------|----|----|----------------|
| **macOS Keychain Integration** | ✅ Full support | ❌ **Removed** | Use external tools or TOML file references |
| **Secret Management Commands** | ✅ `secret write/read/list` | ❌ **Removed** | Manual secret management |
| **Variable Replacement (`%` prefix)** | ✅ `%MYTOOL_DB_PASSWD` | ❌ **Removed** | Use shell variable expansion |
| **File Content Loading (`&` prefix)** | ✅ `KEY=&/path/file` | ✅ **Replaced with TOML** | Migrate to TOML `file = "path"` |
| **Command Execution (backticks)** | ✅ `KEY=`\`command\`` | ✅ **Replaced with TOML** | Migrate to TOML `command/args` |
| **Namespace-based secrets** | ✅ `@namespace` | ❌ **Removed** | No equivalent |

## What v2 Focuses On

v2 is designed for **configuration-driven** environment management:
- **TOML configuration files** with structured settings
- **Multiple file support** with clear precedence
- **Clean CLI interface** without complex secret management
- **Better architecture** for simple use cases

## Detailed Feature Comparison

### Basic Environment Variable Loading

#### v1 (.env files)
```bash
# .env file
POSTGRES_DB=dev_db
POSTGRES_USER=testuser

# Usage
zenv psql  # Automatically loads .env
zenv -e production.env psql  # Load specific file
```

#### v2 (.env files - Compatible)
```bash
# Same .env file works
POSTGRES_DB=dev_db
POSTGRES_USER=testuser

# Usage (same commands work)
zenv psql  # Automatically loads .env
zenv -e production.env psql  # Load specific file
```

**✅ Migration**: No changes needed for basic .env usage.

### Advanced Variable Loading

#### v1 (Advanced Syntax in .env)
```bash
# .env with v1 special syntax
SECRET_KEY=&/path/to/secret.txt         # File content
API_TOKEN=`gcloud auth print-access-token`  # Command execution
DB_PASSWORD=%VAULT_DB_PASS              # Variable replacement
```

#### v2 (TOML Configuration)
```toml
# .env.toml
[SECRET_KEY]
file = "/path/to/secret.txt"

[API_TOKEN]
command = "gcloud"
args = ["auth", "print-access-token"]

# DB_PASSWORD=%VAULT_DB_PASS - NO EQUIVALENT in v2
```

**⚠️ Migration**: 
- File loading: ✅ Migrate to TOML
- Command execution: ✅ Migrate to TOML  
- Variable replacement: ❌ No equivalent - use shell expansion

### Secret Management

#### v1 (Full Keychain Integration)
```bash
# Store secrets in macOS Keychain
zenv secret write @aws AWS_SECRET_ACCESS_KEY
zenv secret generate @project API_TOKEN 32
zenv secret list

# Use in .env
AWS_SECRET_ACCESS_KEY=%@aws
API_TOKEN=%@project
```

#### v2 (No Secret Management)
```bash
# NO EQUIVALENT - Feature completely removed
```

**❌ Migration**: No path available. Must use:
- External secret management tools
- Manual environment variable setting
- File-based secrets with appropriate permissions

## Breaking Changes

### 1. Installation and Module Path

#### v1
```bash
go install github.com/m-mizutani/zenv@latest
```

#### v2
```bash
go install github.com/m-mizutani/zenv/v2@latest
```

**⚠️ Important**: Even after v2.0.0 is released, `go install github.com/m-mizutani/zenv@latest` will still install **v1** due to Go modules major version handling. You **must** use the `/v2` suffix to get v2.

### 2. Environment Variable Precedence

#### v1 Precedence
The exact precedence in v1 varied, but generally:
- System environment variables
- `.env` files
- Command-line arguments

#### v2 Precedence (Explicit)
```
System < .env < TOML < Inline
```
Where `<` means "is overridden by"

**What This Means:**
- **System environment variables** have the **lowest** priority
- **Inline variables** (KEY=value) have the **highest** priority
- **TOML files** override `.env` files
- **`.env` files** override system variables

**Action Required**: Review your environment variable setup to ensure the new precedence order works for your use case.

### 3. CLI Interface Changes

#### New Options in v2
- `-t, --toml FILE`: Load environment variables from TOML file
- Automatic environment variable listing when no command is specified

#### Behavioral Changes
- **v1**: Required explicit command or specific flags to show variables
- **v2**: Shows environment variables by default when no command is provided

```bash
# v2 - Shows all loaded environment variables
zenv -e .env.production

# v2 - Execute command with environment variables
zenv -e .env.production npm start
```

## New Features in v2

### 1. TOML Configuration Files

v2 introduces powerful TOML configuration support:

#### Basic Static Values
```toml
[DATABASE_URL]
value = "postgresql://localhost/mydb"

[API_KEY]
value = "your-api-key"
```

#### File Content Loading
```toml
[SECRET_KEY]
file = "/path/to/secret/file"

[SSL_CERT]
file = "/etc/ssl/certs/app.pem"
```

#### Command Execution
```toml
[GIT_COMMIT]
command = "git"
args = ["rev-parse", "HEAD"]

[BUILD_TIME]
command = "date"
args = ["+%Y-%m-%d %H:%M:%S"]
```

#### Multiline Values
```toml
[CONFIG_JSON]
value = """
{
  "database": {
    "host": "localhost",
    "port": 5432
  }
}
"""
```

#### Alias Support (NEW in v2)
```toml
# Reference system environment variables
[USER_HOME]
alias = "HOME"

# Reference other variables in the same file
[PRIMARY_DB]
value = "postgresql://primary.db.com/myapp"

[DATABASE_URL]
alias = "PRIMARY_DB"

# Create alternative names for existing variables
[DB_CONNECTION]
alias = "DATABASE_URL"
```

#### Template Support (NEW in v2)
```toml
# Combine multiple variables using Go templates
[DB_USER]
value = "admin"

[DB_PASS]
file = "/secrets/db_password"

[DB_HOST]
value = "localhost"

[DB_PORT]
value = "5432"

[DB_NAME]
value = "myapp"

[DATABASE_URL]
template = "postgresql://{{ .DB_USER }}:{{ .DB_PASS }}@{{ .DB_HOST }}:{{ .DB_PORT }}/{{ .DB_NAME }}"
refs = ["DB_USER", "DB_PASS", "DB_HOST", "DB_PORT", "DB_NAME"]

# Conditional configuration
[USE_STAGING]
value = "true"

[API_ENDPOINT]
template = "{{ if .USE_STAGING }}https://staging.api.example.com{{ else }}https://api.example.com{{ end }}"
refs = ["USE_STAGING"]

# Combine paths
[LOG_PATH]
template = "{{ .HOME }}/logs/{{ .APP_NAME }}.log"
refs = ["HOME", "APP_NAME"]
```

### 2. Multiple File Support

```bash
# Load from multiple sources with clear precedence
zenv -e base.env -e override.env -t config.toml KEY=value command
```

### 3. Enhanced Error Handling

v2 provides better error messages and handles edge cases more gracefully.

## Step-by-Step Migration

### Step 1: Update Installation

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

### Step 4: Consider TOML Migration (Optional)

For complex configurations, consider migrating to TOML:

#### Before (.env)
```env
DATABASE_URL=postgresql://localhost/mydb
API_KEY_FILE=/path/to/api/key
GIT_COMMIT=$(git rev-parse HEAD)
```

#### After (.env.toml)
```toml
[DATABASE_URL]
value = "postgresql://localhost/mydb"

[API_KEY]
file = "/path/to/api/key"

[GIT_COMMIT]
command = "git"
args = ["rev-parse", "HEAD"]
```

### Step 5: Update Scripts and CI/CD

Update any scripts or CI/CD configurations:

```bash
# Before
zenv -e .env.production deploy.sh

# After (same command, but new features available)
zenv -e .env.production -t config.toml deploy.sh
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
```

**v2 Migration Options:**

#### Option A: Migrate to TOML
```bash
# .env (keep simple values)
API_BASE_URL=https://api.myapp.com

# .env.toml (migrate complex values)
[SECRET_KEY]
file = "/etc/myapp/secret.key"

[GIT_COMMIT]
command = "git"
args = ["rev-parse", "HEAD"]

# Note: DATABASE_PASSWORD=%VAULT_DATABASE_PASS has no equivalent
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

# .env.toml
[DATABASE_PASSWORD]
file = "/etc/myapp/db_password"

[API_KEY]
file = "/etc/myapp/api_key"
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

# .env.toml (for command-based values)
[GIT_BRANCH]
command = "git"
args = ["rev-parse", "--abbrev-ref", "HEAD"]

# Use external secret management for CI
export SECRET_TOKEN=$CI_SECRET_TOKEN  # from CI environment

zenv -e .env -t .env.toml docker build -t myapp:$DOCKER_TAG .
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
zenv -e .env -t config.toml
```

### Issue: Command Execution in TOML Not Working

**Solution**: Ensure proper TOML syntax for commands:

```toml
# Correct
[GIT_BRANCH]
command = "git"
args = ["rev-parse", "--abbrev-ref", "HEAD"]

# Incorrect
[GIT_BRANCH]
command = "git rev-parse --abbrev-ref HEAD"
```

### Issue: File Loading Not Working

**Solution**: Check file paths and permissions:

```toml
[SECRET]
file = "/absolute/path/to/file"  # Use absolute paths
```

## Best Practices for v2

1. **Use TOML for complex configurations** that need file reading or command execution
2. **Keep .env for simple key-value pairs** that don't change often
3. **Use inline variables** for one-off overrides
4. **Test precedence** by running `zenv` without a command to see all variables
5. **Use descriptive TOML section names** that clearly indicate the variable purpose

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
- Want better TOML configuration support
- Need clear environment variable precedence
- Prefer configuration files over secret management
- Don't rely on v1's special syntax (`%`, `&`, backticks)
- Don't use macOS Keychain integration

### ❌ **Stay with v1 if you:**
- Heavily use macOS Keychain integration (`@namespace`)
- Rely on variable replacement (`%VARIABLE`)
- Use extensive file loading with `&` prefix in .env
- Have complex secret management workflows
- Need backwards compatibility with existing scripts
- Use namespace-based secret organization

### 🔄 **Consider hybrid approach if you:**
- Want to test v2 for new projects
- Can gradually migrate simple configurations
- Have mixed complexity environments
- Want to benefit from v2's improved architecture

### Alternative: Stay with v1

v1 continues to be maintained and has more advanced features. If your use case depends on:
- **Secret management**: v1's Keychain integration is more sophisticated
- **Dynamic variables**: v1's `%` syntax is more flexible
- **Complex workflows**: v1 supports more advanced use cases

**There's no requirement to migrate.** Choose the version that best fits your workflow.

For additional support, see:
- [v1 Documentation](https://github.com/m-mizutani/zenv/blob/main/README.md) for v1 features
- [v2 Documentation](../README.md) for v2 features  
- [GitHub Issues](https://github.com/m-mizutani/zenv/issues) for bug reports and questions