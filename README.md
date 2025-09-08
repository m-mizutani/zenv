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

## License

Apache License 2.0
