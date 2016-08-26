# emp

A CLI for Empire.

## Installation

You can always download the latest version with:

```console
$ curl -L https://github.com/remind101/empire/releases/download/v0.11.0/emp-`uname -s`-`uname -m` \
  > /usr/local/bin/emp
```

Or by using homebrew:

```console
$ brew update
$ brew install emp
```

If you have a working Go 1.5+ environment, you can do the following:

```console
$ go get -u github.com/remind101/empire/cmd/emp
```

Otherwise, you can find a complete list of releases [here](https://github.com/remind101/empire/releases).

## Usage

The basic usage of emp is:

```
Usage: EMPIRE_API_URL=<empire api> emp <command> [-a app] [options] [arguments]
```
