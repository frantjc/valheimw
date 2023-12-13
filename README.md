# sindri [![godoc](https://pkg.go.dev/badge/github.com/frantjc/sindri.svg)](https://pkg.go.dev/github.com/frantjc/sindri) [![goreportcard](https://goreportcard.com/badge/github.com/frantjc/sindri)](https://goreportcard.com/report/github.com/frantjc/sindri) ![license](https://shields.io/github/license/frantjc/sindri)

Easily run a dedicated Valheim server with mods from [thunderstore.io](https://valheim.thunderstore.io/) and a way to easily share those mods with Valheim clients.

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
      --rm stringArray         Thunderstore mods to remove (case-sensitive)
  -r, --root string            root directory for Sindri. Valheim savedir resides here (default "~/.local/share/sindri/root")
  -s, --state string           state directory for Sindri (default "~/.local/share/sindri/state")
  -V, --verbose count          verbosity for Sindri
  -v, --version                version for sindri
      --world string           world for Valheim (default "sindri")
```

### Root directory

Sindri is tied to its root directory, meaning that if you stop Sindri and then start it back up with the same root directory, it will pick back up where it left off: same mod list, same world, etc. However, if any manual changes are made to this directory, all bets are off.

By default, the root directory will be `$XDG_DATA_HOME/sindri` per the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html). This can be overridden with `--root`.

```sh
docker run \
    # Make sure to map a volume to the --root directory
    # or important data such as your world saves will be
    # tied to the container's filesystem, making it easy
    # to lose if the container is destroyed.
    --volume $(pwd)/sindri:/var/lib/sindri \
    # Valheim listens on port 2456 for UDP traffic by default.
    --publish 2456:2456/udp \
    ghcr.io/frantjc/sindri:0.7.1 \
        --root /var/lib/sindri
```

### State directory

Sindri also uses a state directory (default `$XDG_RUNTIME_DIR/sindri`) for ephemeral data. This is not important to be kept the same nor does it need to be kept around.

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    --publish 2456:2456/udp \
    ghcr.io/frantjc/sindri:0.7.1 \
        --root /var/lib/sindri \
        --state /run/sindri
```

### Mods

Sindri downloads its mods from thunderstore.io. Mods can be referenced a number of ways. For example, [FarmGrid](https://valheim.thunderstore.io/package/Nexus/FarmGrid/) can be referenced by its full name, `Nexus-FarmGrid-0.2.0` (note that the version is optional--if omitted, the latest version is used); by a [distribution-spec](https://github.com/opencontainers/distribution-spec)-like reference, `Nexus/FarmGrid:0.2.0`; or by a GitHub-Actions-like reference, `Nexus/FarmGrid@0.2.0`. Note that these are all case-sensitive.

The desired list of mods can be passed to Sindri via `--mod`.

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    --publish 2456:2456/udp \
    ghcr.io/frantjc/sindri:0.7.1 \
        --root /var/lib/sindri \
        --mod Nexus/FarmGrid
```

Sindri can remove mods from a previous run as well.

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    --publish 2456:2456/udp \
    ghcr.io/frantjc/sindri:0.7.1 \
        --root /var/lib/sindri \
        --rm Nexus/FarmGrid
```

### Distributing mods to clients

After Sindri is running, if it has any mods, you can download them from it (the exact versions!) for your Valheim client. It listens on `:8080` by default which can be overridden with `--addr`.

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    --publish 2456:2456/udp \
    --publish 8080:8080 \
    ghcr.io/frantjc/sindri:0.7.1 \
        --root /var/lib/sindri
```

Then you can use your HTTP client of choice to download a `.tar` with the mods.

```powershell
cd "C:\Program Files (x86)\Steam\steamapps\common\Valheim"
curl -fSs http://your-sindri-address/mods.tar.gz | tar -xzf -
```

After the initial install, you Sindri supplies some helpful scripts to use to update and uninstall it, respectively.

```powershell
cd "C:\Program Files (x86)\Steam\steamapps\common\Valheim"
update-sindri
```

```powershell
cd "C:\Program Files (x86)\Steam\steamapps\common\Valheim"
uninstall-sindri
```

### Valheim options

Valheim arguments other than `-savedir` (sourced from `--root`) and `-password` (required and sourced from the environment variable `VALHEIM_PASSWORD`) can be passed through Sindri by flags of the same name. If not provided, `--world` and `--name` default to "sindri", while `--port` will not be passed to Valheim if not provided. Lastly, `--public` only needs to be defined, it does not need the value of "1".

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    # Make sure to publish the correct port if you change
    --publish 3567:3567/udp \
    --publish 8080:8080 \
    --env VALHEIM_PASSWORD=mustbe5chars \
    ghcr.io/frantjc/sindri:0.7.1 \
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
    --publish 3567:3567/udp \
    --publish 8080:8080 \
    --env VALHEIM_PASSWORD=mustbe5chars \
    ghcr.io/frantjc/sindri:0.7.1 \
        --root /var/lib/sindri \
        --beta public-test \
        --beta-password yesimadebackups
```

### Make it faster

Sindri can be made to start up faster on subsequent runs by skipping redownloading Valheim and/or mods by using `--airgap` or `--mods-only`.

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    --publish 3567:3567/udp \
    --publish 8080:8080 \
    --env VALHEIM_PASSWORD=mustbe5chars \
    ghcr.io/frantjc/sindri:0.7.1 \
        --root /var/lib/sindri \
        --airgap
```
