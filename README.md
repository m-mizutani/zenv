# zenv

`zenv` is more powerful `env` command to manage environment variables in CLI.

- Save and get secret values in/from Keychain (inspired by [envchain](https://github.com/sorah/envchain))
- Load different environment variables depending on current working directory

## Install

```sh
go get github.com/m-mizutani/zenv
```

## Usage

### Save a secret value to Keychain

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

### Load environment variable by working directory

Create `~/.zenv` configuration file.

```toml
[workdir.proj1]
dirpath = "/Users/mizutani/works/github.com/m-mizutani/proj1"
vars = ["DBNAME=proj1"]
```

Then,

```sh
$ cd $HOME
$ zenv env | grep DBNAME
(no output)
$ cd /Users/mizutani/works/github.com/m-mizutani/proj1
$ zenv env | grep DBNAME
DBNAME=proj1
```

## License

MIT License
