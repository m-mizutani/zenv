# zenv ![CI](https://github.com/m-mizutani/zenv/actions/workflows/test.yml/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/m-mizutani/zenv)](https://goreportcard.com/report/github.com/m-mizutani/zenv)

`zenv` is enhanced `env` command to manage environment variables in CLI.

- Load environment variable from `.env` file by
    - Static values
    - Reading file content
    - Executing command
- Securely save, generate and get secret values with Keychain, inspired by [envchain](https://github.com/sorah/envchain) (supported only macOS)
- Replace command line argument with loaded environment variable

## Install

```sh
go install github.com/m-mizutani/zenv@latest
```

## Usage

### Set environment variable with `env` command



### Save and load secret value(s) to Keychain

Save values.
```sh
$ zenv secret write @aws-account AWS_SECRET_ACCESS_KEY
Value: ***
$ zenv secret @aws-account AWS_ACCESS_KEY_ID
Value: ***
```

Use values.
```sh
$ zenv @aws-account aws s3 ls
```

### Generate random secure value

`generate` subcommand can generate random value like token and save to KeyChain.

```sh
$ zenv secret generate @my-project MYSQL_PASS
$ zenv secret generate @my-project -n 8 TMP_TOKEN # set length to 8
$ zenv list @my-project
MYSQL_PASS=******************************** (hidden)
TMP_TOKEN=******** (hidden)
```

### List loaded environment variables

`list` subcommand output list of all loaded environment variables. Mask secret values with `*` if the variable is loaded from Keychain.

```sh
$ zenv list @aws-account AWS_REGION=ap-northeast-1
AWS_ACCESS_KEY_ID=******************** (hidden)
AWS_SECRET_ACCESS_KEY=**************************************** (hidden)
AWS_REGION=ap-northeast-1
```

### Load from .env file

```sh
$ cat .env
AWS_REGION=ap-northeast-1
$ zenv list
AWS_REGION=ap-northeast-1
```

`.env` is default dotenv file name and zenv loads environment variables from the file if existing. Also, `-e` option specifies a file used as `.env`.

### Mixing keychain profile to .env file

```sh
$ cat .env
@aws-account
AWS_REGION=ap-northeast-1
$ zenv -l
AWS_ACCESS_KEY_ID=******************** (hidden)
AWS_SECRET_ACCESS_KEY=**************************************** (hidden)
AWS_REGION=ap-northeast-1
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
