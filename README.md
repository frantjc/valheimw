# sindri [![godoc](https://pkg.go.dev/badge/github.com/frantjc/sindri.svg)](https://pkg.go.dev/github.com/frantjc/sindri) [![goreportcard](https://goreportcard.com/badge/github.com/frantjc/sindri)](https://goreportcard.com/report/github.com/frantjc/sindri)

Sindri is a toolkit for turning Steam app servers into containers. This repository also houses tools built from this toolkit.

## valheimw

`valheimw` is a wrapper around the Valheim server.

On start up, it installs the latest version of the specified branch of the Valheim server (public by default), any given [thunderstore.io](https://valheim.thunderstore.io/) mods and BepInEx to load them. Then it executes the Valheim server using those mods, if any.

It also runs an HTTP server alongside the Valheim server which provides endpoints to do a number of helpful things.

- Download a tarball of the mods in use.
- Download the world's `.db` and `.fwl` files.
- Get information from the world's `.fwl` file.
- Go to the world's [valheim-map.world](https://valheim-map.world/) page.

Lastly, it documents all arguments that can be passed to Valheim's server.

```sh
docker run --rm ghcr.io/frantjc/valheimw --help
```

See [examples/valheimw](examples/valheimw).

## boiler

`boiler` is a read-only container registry for pulling images with Steam apps installed on them. The base of the images is `debian:stable-slim`. Images are non-root and `steamcmd` is never installed on them, so there's no leftover files from it on the image's filesystem or in its layers. Images are built on-demand rather than being stored, waiting to be pulled.

The image's tag maps to the Steam app's branch, except the specific case of the default tag "latest" which maps to the default Steam app branch "public".
See [examples/boiler](examples/boiler).


## mist

`mist` is a CLI intended for use in Dockerfiles to install Steam apps, Steam Workshop items, and [thunderstore.io](https://thunderstore.io/) mods. See [examples/mist](examples/mist).

## corekeeper

`corekeeper` is a container image built by `boiler` and layered upon to satisfy the Core Keeper server's additional dependecies. See [examples/corekeeper](examples/corekeeper).
