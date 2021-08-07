# zenv

`zenv` is more powerful `env` command to manage environment variables in CLI.

- Save and get secret values in/from Keychain (inspired by [envchain](https://github.com/sorah/envchain))
- Load `.env` file to import environment variables

## Install

```sh
go get github.com/m-mizutani/zenv
```

## Usage

### Save and load secret value(s) to Keychain

Save values.
```sh
$ zenv -w @aws-account AWS_SECRET_ACCESS_KEY
Value: ***
$ zenv -w @aws-account AWS_ACCESS_KEY_ID
Value: ***
```

Use values.
```sh
$ zenv @aws-account aws s3 ls
```

### List loaded environment variables

`-l` option output list of all loaded environment variables. Mask secret values with `*` if the variable is loaded from Keychain.

```sh
$ zenv -l @aws-account AWS_REGION=ap-northeast-1
AWS_ACCESS_KEY_ID=******************** (hidden)
AWS_SECRET_ACCESS_KEY=**************************************** (hidden)
AWS_REGION=ap-northeast-1
```

### Load from .env file

```sh
$ cat .env
AWS_REGION=ap-northeast-1
$ zenv -l
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
