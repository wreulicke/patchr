# patchr

This is patch tool using comment directive.

## Motivation

This aims to make scaffolding more easily and safely.

Sometimes we can see some templates that don't work after generating from them.
We should build and test those templates on CI pipelines, but, it is hard to test templates correctness because templates usually does'nt work before generation.
So, this `patchr` aims to create a `living` scaffolding template.

## Installation

Download from [Releases](https://github.com/wreulicke/patchr/releases)

```bash
# Mac
curl -o patchr -sL "https://github.com/wreulicke/patchr/releases/download/v0.0.1/patchr_0.0.1_darwin_arm64"
curl -o patchr -sL "https://github.com/wreulicke/patchr/releases/download/v0.0.1/patchr_0.0.1_darwin_amd64"

# Linux
curl -o patchr -sL "https://github.com/wreulicke/patchr/releases/download/v0.0.1/patchr_0.0.1_linux_amd64"
curl -o patchr -sL "https://github.com/wreulicke/patchr/releases/download/v0.0.1/patchr_0.0.1_linux_arm64"

# Windows
curl -o patchr -sL https://github.com/wreulicke/patchr/releases/download/v0.0.1/patchr_0.0.1_windows_amd64.exe
curl -o patchr -sL https://github.com/wreulicke/patchr/releases/download/v0.0.1/patchr_0.0.1_windows_arm64.exe

chmod +x patchr
mv patchr /usr/local/bin
```

## Usage

```bash
$ patchr
patchr is a tool to apply patches to source code using comment directives

Usage:
  patchr [file or directory] [flags]
  patchr [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version     show version

Flags:
  -p, --comment-prefix string   overrides comment prefix
  -d, --dry-run                 dry run
  -h, --help                    help for patchr
  -v, --values string           values file for template

Use "patchr [command] --help" for more information about a command.
```

## Example

See [testdata](./testdata/).

## License

MIT License