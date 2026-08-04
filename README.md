# a duf fork

This is a fork of [duf](https://github.com/muesli/duf), a disk usage/free
utility for Linux, BSD, macOS, and Windows. See the
[upstream repository](https://github.com/muesli/duf) for installation
instructions and full documentation.

## about this fork

This fork keeps the upstream interface and adds:

- minimum total-capacity filtering with `--gte`;
- case-preserving mount-point filters with optional symlink dereferencing; and
- short forms for common options plus GNU/BSD `du` shorthand compatibility.

### minimum volume size

`--gte SIZE` keeps only filesystems whose total capacity is greater than or
equal to `SIZE`:

```console
duf --gte 10G
duf --json --gte 10G
```

The filter is applied before either table or JSON output. Without `--gte`, no
size filter is applied. `SIZE` is an integer byte count optionally followed by
uppercase `K`, `M`, `G`, `T`, `P`, or `E`; these suffixes scale in powers of
1024. Decimal values and suffixes such as `GB` are not accepted.

### mount-point filtering and symlinks

Values passed to `--only-mp` and `--hide-mp` preserve their original case, and
mount-point wildcard matching is case-sensitive. For example, `/Volumes/D`
remains distinct from `/volumes/d`:

```console
duf --only-mp '/Volumes/*'
```

By default, these filters match mount-point paths only. `-L` or `--dereference`
also resolves symlinks matched by the filters and includes their backing mount
points:

```console
duf -L -u /Volumes/'*'
```

Dereferencing applies to both `--only-mp` and `--hide-mp` patterns.
Mount-point filters and dereferencing affect table output; JSON output is
returned before these filters are applied.

### short options

Common filtering and output options have these short forms:

| Short | Long |
| --- | --- |
| `-L` | `--dereference` |
| `-F` | `--hide-fs` |
| `-U` | `--hide-mp` |
| `-J` | `--json` |
| `-i` | `--only` |
| `-f` | `--only-fs` |
| `-u` | `--only-mp` |
| `-o` | `--output` |
| `-R` | `--sort` |

### `du` shorthand compatibility

Commands and aliases written for GNU or BSD `du` can use these shorthands:

- `-h` selects duf's default human-readable output;
- `-D`, `-H`, and `-L` enable `--dereference`, while `-P` disables it (the last
  option wins);
- `-t SIZE` acts as `--gte SIZE`; and
- `-r` acts as `--warnings`.

The following shorthands are accepted but otherwise ignored:

```text
-a -A -b -B SIZE -c -d DEPTH -g -I MASK -k -l -m -n -s -S -x -X FILE
```

These ignored flags provide input compatibility; they do not make duf emulate
the corresponding `du` behavior.

## building

Building requires Go 1.23 or newer:

```console
go build
```

## license

duf is released under the MIT License. Portions copied and modified from
gopsutil retain their BSD license terms. See [LICENSE](LICENSE) for both texts.
