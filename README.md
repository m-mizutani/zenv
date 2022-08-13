# zenv [![CI](https://github.com/m-mizutani/zenv/actions/workflows/test.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/test.yml) [![Security Scan](https://github.com/m-mizutani/zenv/actions/workflows/gosec.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/gosec.yml) [![Vuln scan](https://github.com/m-mizutani/zenv/actions/workflows/trivy.yml/badge.svg)](https://github.com/m-mizutani/zenv/actions/workflows/trivy.yml) <!-- omit in toc -->

`zenv` is enhanced `env` command to manage environment variables in CLI.

- Load environment variable from `.env` file by
    - Static values
    - Reading file content
    - Executing command
- Securely save, generate and get secret values with Keychain, inspired by [envchain](https://github.com/sorah/envchain) (supported only macOS)
- Replace command line argument with loaded environment variable

## Install <!-- omit in toc -->

```sh
go install github.com/m-mizutani/zenv@latest
```

## Basic Usage

### Set by CLI argument

Can set environment variable in same manner with `env` command

```sh
$ zenv POSTGRES_DB=your_local_dev_db psql
```

### Load from `.env` file

Automatically load `.env` file and

```sh
$ cat .env
POSTGRES_DB=your_local_db
POSTGRES_USER=test_user
PGDATA=/var/lib/db
$ zenv psql -h localhost -p 15432
# connecting to your_local_db on localhost:15432 as test_user
```

### Save and load secret values

```sh
# save a secret value
$ zenv secret write @aws-account AWS_SECRET_ACCESS_KEY
Value: # no echo
$ zenv secret write @aws-account AWS_ACCESS_KEY_ID
Value: # no echo

# load a secret value and execute command "aws s3 ls"
$ zenv @aws-account aws s3 ls
2020-06-19 03:53:13 my-bucket1
2020-04-18 06:45:44 my-bucket2
...
```

`secret write` command format is `zenv secret write <Namespace> <Key>` to save a secret value. In above case, `@aws-account` is *Namespace* and `AWS_SECRET_ACCESS_KEY` & `AWS_ACCESS_KEY_ID` are *Key (Environment variable name)*. *Namespace* **must** have `@` prefix.

`zenv <Namespace> <Command>` executes `<Command>` with loaded secret value(s) from `<Namespace>` as environment variables. If multiple environment variables are saved in the `<Namespace>`, all variables are loaded.

### Mixing CLI, `.env` and secret

All of CLI argument, loading `.env` and secret can be used in parallel. An example is following.

```sh
$ zenv secret write @aws-account AWS_SECRET_ACCESS_KEY
Value: # no echo
$ cat .env
AWS_ACCESS_KEY_ID=abcdefghijklmn
$ zenv @aws-account AWS_REGION=jp-northeast-1 aws s3 ls
# access to S3 with AWS_SECRET_ACCESS_KEY, AWS_SECRET_ACCESS_KEY and AWS_REGION
```

Also, `-e` option specifies a file used as `.env`.

### List loaded variables

You can see loaded environment variable by zenv with `zenv list <...>` command.

```sh
$ zenv list @aws-account AWS_REGION=jp-northeast-1
AWS_REGION=jp-northeast-1
AWS_ACCESS_KEY_ID=abcdefghijklmn
AWS_SECRET_ACCESS_KEY=******************************** (hidden)
```

You can specify arguments to specify loading environment in same manner with executing command. Of curse secret values loaded from Keychain will be masked.

## Advanced Usage

### Generate random secure value

`secret generate` subcommand can generate random value like token and save to KeyChain.

```sh
$ zenv secret generate @my-project MYSQL_PASS
$ zenv secret generate @my-project -n 8 TMP_TOKEN # set length to 8
$ zenv list @my-project
MYSQL_PASS=******************************** (hidden)
TMP_TOKEN=******** (hidden)
```

### List namespaces

`secret list` subcommand shows list of namespaces.

```sh
$ zenv secret list
@aws
@local-db
@staging-db
```

### Put namespace into .env file

You can also can put *Namespace* for secret values into `.env` file. Then, `zenv` always loads secret values without *Namespace* argument.

```sh
$ zenv secret write @aws AWS_SECRET_ACCESS_KEY
Value # <- input

$ cat .env
@aws
AWS_REGION=jp-northeast-1
AWS_ACCESS_KEY_ID=abcdefghijklmn

$ zenv list
AWS_REGION=jp-northeast-1
AWS_ACCESS_KEY_ID=abcdefghijklmn
AWS_SECRET_ACCESS_KEY=******************************** (hidden)
```

### Replace value in environment variable with another one

`zenv` replaces words having `%` prefix with existing another environment variable.

```sh
$ cat .env
MYTOOL_DB_PASSWD=abc123
PGPASSWORD=%MYTOOL_DB_PASSWD
$ zenv list
MYTOOL_DB_PASSWD=abc123
PGPASSWORD=abc123
```

### Replace value in arguments with loaded environment variable

`zenv` replaces words having `%` prefix with loaded environment variable.

```sh
$ cat .env
TOKEN=abc123
$ zenv curl -v -H "Authorization: bearer %TOKEN" http://localhost:1234
(snip)
> GET /api/v1/alert HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/7.64.1
> Accept: */*
> Authorization: bearer abc123
(snip)
```

### Loading file content to environment variable

Sometime, we need to load large content into environment variable. For example, Google OAuth2 credential file is slightly large to write in `.env` file and complicated. `zenv` can load file content into environment variable with `&` prefix.

```sh
$ cat .env
GOOGLE_OAUTH_DATA=&tmp/client_secret_00000-abcdefg.apps.googleusercontent.com.json
$ zenv list
GOOGLE_OAUTH_DATA={"web":{"client_id":"00000...(snip)..."}}
$ zenv ./some-oauth-server
```

### Execute command in `.env` file

`zenv` recognizes environment value as command by surrounding with `` ` `` backquote (backtick). The feature is useful to set short live token that provided CLI command. `zenv` set standard output as value of environment variable.

```sh
$ cat .env
GOOGLE_TOKEN=`gcloud auth print-identity-token`
$ zenv list
GOOGLE_TOKEN=eyJhbGciOiJS...(snip)
$ zenv ./some-app-requires-token
```

### Backup and restore secrets

For example, when migrating PC, we need to transfer every data including secrets. So backup and restore features are required. `zenv` provides export and import command as following.

```sh
$ zenv secret write @aws AWS_SECRET_ACCESS_KEY
Value: # <- input secret
$ zenv secret export -o backup.json
input passphrase: # <- input passphrase
exported secrets to backup.json
$ cat backup.json
{
  "CreatedAt": "2022-03-27T13:37:06.577827+09:00",
  "Encrypted": "wr/s6Z5T4diP6Ihu1318tL2tRA2Ch2LImAB1QEJi0...(snip)..."
}
```

`secret export` command dumps encrypted all secrets to JSON file. You can filter dumped namespace by `-n` option.

After that, move `backup.json` to new machine and import it.

```sh
$ zenv secret import backup.json
input passphrase: # <- input passphrase
$ zenv list @aws
AWS_SECRET_ACCESS_KEY=******************************** (hidden)
```

Passphrase must be same when exporting and importing.

## License

Apache License 2.0
