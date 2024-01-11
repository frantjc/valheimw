# sindri [![godoc](https://pkg.go.dev/badge/github.com/frantjc/sindri.svg)](https://pkg.go.dev/github.com/frantjc/sindri) [![goreportcard](https://goreportcard.com/badge/github.com/frantjc/sindri)](https://goreportcard.com/report/github.com/frantjc/sindri) ![license](https://shields.io/github/license/frantjc/sindri)

Easily run a dedicated Valheim server with mods from [thunderstore.io](https://valheim.thunderstore.io/) and a way to easily share those mods with Valheim clients.

## usage

```sh
Usage:
  sindri [flags]

Flags:
      --addr string              address for sindri (default ":8080")
      --admin int64Slice         Valheim server admin Steam IDs (default [])
      --backup-long duration     Valheim server -backuplong duration
      --backup-short duration    Valheim server -backupshort duration
      --backups int              Valheim server -backup amount
      --ban int64Slice           Valheim server banned Steam IDs (default [])
      --beta string              Steam beta branch
      --beta-password string     Steam beta password
      --combat-modifier          Valheim server -modifier combat
      --crossplay                Valheim server enable -crossplay
      --death-penalty-modifier   Valheim server -modifier deathpenalty
  -h, --help                     help for sindri
      --instance-id string       Valheim server -instanceid
  -m, --mod stringArray          Thunderstore mods (case-sensitive)
      --mods-only                do not redownload Valheim
      --name string              Valheim server -name (default "sindri")
      --no-build-cost            Valheim server -setkey nobuildcost
      --no-download              do not redownload Valheim or mods
      --no-map                   Valheim server -setkey nomap
      --passive-mobs             Valheim server -setkey passivemobs
      --permit int64Slice        Valheim server permitted Steam IDs (default [])
      --player-events            Valheim server -setkey playerevents
      --port int                 Valheim server -port (0 to use default)
      --portal-modifier          Valheim server -modifier portals
      --preset                   Valheim server -preset
      --public                   Valheim server make -public
      --raid-modifier            Valheim server -modifier raids
      --resource-modifier        Valheim server -modifier resources
      --rm stringArray           Thunderstore mods to remove (case-sensitive)
  -r, --root string              root directory for sindri (-savedir resides here) (default "~/.local/share/sindri")
      --save-interval duration   Valheim server -saveinterval duration
  -s, --state string             state directory for sindri (default "~/.local/share/sindri")
  -V, --verbose count            verbosity for sindri
  -v, --version                  version for sindri
      --world string             Valheim server -world (default "sindri")

```

### Root directory

Sindri is tied to its root directory, meaning that if you stop Sindri and then start it back up with the same root directory, it will pick back up where it left off: same mod list, same world, etc.

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
    ghcr.io/frantjc/sindri:1.3.0 \
        --root /var/lib/sindri
```

### Mods

Sindri downloads its mods from thunderstore.io. Mods can be referenced a number of ways. For example, [FarmGrid](https://valheim.thunderstore.io/package/Nexus/FarmGrid/) can be referenced by its full name, `Nexus-FarmGrid-0.2.0` (note that the version is optional--if omitted, the latest version is used); by a [distribution-spec](https://github.com/opencontainers/distribution-spec)-like reference, `Nexus/FarmGrid:0.2.0`; or by a GitHub-Actions-like reference, `Nexus/FarmGrid@0.2.0`. Note that these are all case-sensitive.

The desired list of mods can be passed to Sindri via `--mod`.

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    --publish 2456:2456/udp \
    ghcr.io/frantjc/sindri:1.3.0 \
        --root /var/lib/sindri \
        --mod Nexus/FarmGrid
```

### Distributing mods to clients

After Sindri is running, if it has any mods, you can download them from it (the exact versions!) for your Valheim client. It listens on `:8080` by default which can be overridden with `--addr`.

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    --publish 2456:2456/udp \
    --publish 8080:8080 \
    ghcr.io/frantjc/sindri:1.3.0 \
        --root /var/lib/sindri
```

Then you can use your HTTP client of choice to download a `.tar` with the mods.

```powershell
cd "C:\Program Files (x86)\Steam\steamapps\common\Valheim"
curl -fSs http://your-sindri-address/mods.gz | tar -xzf -
```

After the initial install, Sindri supplies some helpful scripts to use to update and uninstall it, respectively.

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
    # Make sure to publish the correct port if you change it.
    --publish 3567:3567/udp \
    --publish 8080:8080 \
    --env VALHEIM_PASSWORD=atleast5chars \
    ghcr.io/frantjc/sindri:1.3.0 \
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
    --env VALHEIM_PASSWORD=atleast5chars \
    ghcr.io/frantjc/sindri:1.3.0 \
        --root /var/lib/sindri \
        --beta public-test \
        --beta-password yesimadebackups
```

### Make it faster

Sindri can be made to start up faster on subsequent runs by skipping redownloading Valheim and/or mods by using `--no-download` and/or `--mods-only`.

```sh
docker run \
    --volume $(pwd)/sindri:/var/lib/sindri \
    --publish 3567:3567/udp \
    --publish 8080:8080 \
    --env VALHEIM_PASSWORD=atleast5chars \
    ghcr.io/frantjc/sindri:1.3.0 \
        --root /var/lib/sindri \
        --no-download
```
