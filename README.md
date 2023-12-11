# sindri [![godoc](https://pkg.go.dev/badge/github.com/frantjc/sindri.svg)](https://pkg.go.dev/github.com/frantjc/sindri) [![goreportcard](https://goreportcard.com/badge/github.com/frantjc/sindri)](https://goreportcard.com/report/github.com/frantjc/sindri) ![license](https://shields.io/github/license/frantjc/sindri)

Easily run a dedicated Valheim server with mods from [thunderstore.io](https://valheim.thunderstore.io/).

## usage

```
Usage:
  sindri [flags]

Flags:
      --addr string            address for Sindri (default ":8080")
      --airgap                 do not redownload Valheim or mods
      --beta string            Steam beta branch
      --beta-password string   Steam beta password
  -h, --help                   help for sindri
  -m, --mod stringArray        Thunderstore mods (case-sensitive)
      --mods-only              do not redownload Valheim
      --name string            name for Valheim (default "sindri")
      --port int               port for Valheim (0 to use default)
      --public                 make Valheim server public
  -r, --root string            root directory for Sindri. Valheim savedir resides here (default "~/.local/share/sindri/root")
  -s, --state string           state directory for Sindri (default "~/.local/share/sindri/state")
  -V, --verbose count          verbosity for Sindri
  -v, --version                version for sindri
      --world string           world for Valheim (default "sindri")
```

### Root directory

Sindri is tied to its root directory, meaning that if you stop Sindri and then start it back up with the same root directory, it will pick back up where it left off: same mod list (plus any that have been added), same world, etc. If any manual changes are made to this directory, all bets are off.

By default, the root directory will be `$XDG_DATA_HOME/sindri` per the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html). This can be overridden with `--root`.

```sh
docker run \
    # Make sure to map a volume to the --root directory
    # or important data such as your world saves will be
    # tied to the container's filesystem, making it easy
    # to lose if the container is destroyed.
    --volume $(pwd)/sindri:/var/lib/sindri \
    ghcr.io/frantjc/sindri:0.7.0 \
        --root /var/lib/sindri
```

### State directory

Sindri also uses a state directory (default `$XDG_RUNTIME_DIR/sindri`) for ephemeral data. This is not important to be kept the same nor does it need to be kept around.

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    ghcr.io/frantjc/sindri:0.7.0 \
        --root /var/lib/sindri \
        --state /run/sindri
```

### Mods

Sindri downloads its mods from thunderstore.io. Mods can be referenced a number of ways. For example, [FarmGrid](https://valheim.thunderstore.io/package/Nexus/FarmGrid/) can be referenced by its full name, `Nexus-FarmGrid-0.2.0` (note that the version is optional--if omitted, the latest version is used); by a [distribution-spec](https://github.com/opencontainers/distribution-spec)-like reference, `Nexus/FarmGrid:0.2.0`; or by a GitHub-Actions-like reference, `Nexus/FarmGrid@0.2.0`. Note that these are all case-sensitive.

The desired list of mods can be passed to Sindri via `--mod`.

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    ghcr.io/frantjc/sindri:0.7.0 \
        --root /var/lib/sindri \
        --state /run/sindri \
        --mod Nexus/FarmGrid
```

After Sindri is running, you can download the mods from it (the exact versions!) for your Valheim client.

```sh
cd "C:\Program Files (x86)\Steam\steamapps\common\Valheim"
curl http://your-sindri-address/mods.tar.gz | tar -xzf -
```

Sindri can remove mods from a previous run as well.

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    ghcr.io/frantjc/sindri:0.7.0 \
        --root /var/lib/sindri \
        --rm Nexus/FarmGrid
```

### Valheim options

Valheim arguments other than `-savedir` (sourced from `--root`) and `-password` (required and sourced from the environment variable `VALHEIM_PASSWORD`) can be passed through Sindri by flags of the same name. If not provided, `--world` and `--name` default to "sindri", while `--port` will not be passed to Valheim if not provided. Lastly, `--public` only needs to be defined, it does not need the value of "1".

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    --env VALHEIM_PASSWORD=mustbe5chars \
    ghcr.io/frantjc/sindri:0.7.0 \
        --root /var/lib/sindri \
        --mod Nexus/FarmGrid \
        --port 3567 \
        --world "My world" \
        --name "My world" \
        --public
```

### Beta versions

Sindri can run a beta version of Valheim by using `--beta` and `--beta-password`.

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    --env VALHEIM_PASSWORD=mustbe5chars \
    ghcr.io/frantjc/sindri:0.7.0 \
        --root /var/lib/sindri \
        --beta public-test \
        --beta-password yesimadebackups
```

### Make it faster

Sindri can start up faster on subsequent runs, by skipping redownloading Valheim and mods by using `--mods-only` or `--airgap`.

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    --env VALHEIM_PASSWORD=mustbe5chars \
    ghcr.io/frantjc/sindri:0.7.0 \
        --root /var/lib/sindri \
        --airgap
```
