# Valheim

Sindri boasts special support for Valheim among other Steam app servers due to its origins.

## `valheimw`

This special supports comes in the form of `valheimw`, a **Valheim** server **wrapper**. Instead of being a Steam app server that Sindri can help build into a container image, `valheimw` is a pre-built tool specifically for Valheim servers, modded or otherwise. If mods are specified, it uses [BepInEx](https://valheim.thunderstore.io/p/denikson/BepInExPack_Valheim) to load them.

It provides additional features beyond that in an HTTP server that it runs alongside the Valheim server, including:

- Download a tarball of the mods in use to distribute them to other players.
- Download the world's `.db` and `.fwl` files.
- Get information from the world's `.fwl` file, parsed on your behalf.
- Go to the world's [valheim-map.world](https://valheim-map.world/) page.

For an example of how to use `valheimw`, consider a directory with the following `docker-compose.yml`:

```yml
services:
  valheimw:
    image: ghcr.io/frantjc/valheimw
    command:
      # The name of the Valheim server's save files.
      # If they already exist, it will load them. If
      # they do not, it will create them.
      # Optional. Default "sindri".
      # - --world=hello
      # The name of the Valheim server as a player would
      # see it in-game when connecting to it.
      # Optional. Default "sindri".
      # - --name=there
      # Can be specified if you want to run a
      # pre-release version of the Valheim server.
      # Optional.
      # - --beta=public-test
      # - --beta-password=yesimadebackups
      # Browse https://valheim.thunderstore.io/ for available mods.
      # Names are case-sensitive.
      # Optional.
      # - --mod=RandyKnapp/EquipmentAndQuickSlots
    environment:
      # The password you will use to connect to your server.
      # Required. Must be at least 5 characters, and cannot
      # be contained within your world name. Default world name
      # is "sindri".
      VALHEIM_PASSWORD: hellothere
    volumes:
      # `valheimw` caches stuff here and the Valheim server's
      # save data is here by default.
      - ./saves:/home/valheimw/.cache
    ports:
      # Expose the Valheim server's port.
      # If you change the Valheim server's port from its default,
      # via `--port`, ensure that this is changed to match.
      - 2456:2456/udp
      # Expose `valheimw`'s HTTP server's port.
      # If you change `valheimw`'s HTTP server's port from its default,
      # via `--addr`, ensure that this is changed to match.
      - 8080:8080
```

To run the `valheimw` this way, run the following command in the directory that the above file is placed in.

```sh
docker compose up
```

Once `valheimw` is running, its helpful HTTP server can be used.

To get the world's seed, run the following:

```sh
curl http://localhost:8080/seed.txt
```

To go to the world's [valheim-map.world](https://valheim-map.world/) page, open [localhost:8080/map](http://localhost:8080/map).

To download the mods that the Valheim server is using (if any), run the following command:

```sh
curl http://localhost:8080/mods.gz | tar -xzf-
```

To see an exhaustive list of arguments for `valheimw`, see the following or run the help command yourself:

```sh
docker run ghcr.io/frantjc/valheimw --help
```

```
Usage:
  valheimw [flags]

Flags:
      --addr string              address (default ":8080")
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
  -h, --help                     help for valheimw
      --instance-id string       Valheim server -instanceid
  -m, --mod stringArray          Thunderstore mods (case-sensitive)
      --name string              Valheim server -name (default "sindri")
      --no-build-cost            Valheim server -setkey nobuildcost
      --no-db                    do not expose the world .db file for download
      --no-fwl                   do not expose the world .fwl file information
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
      --save-interval duration   Valheim server -saveinterval duration
      --savedir string           Valheim server -savedir (default "/home/valheimw/.cache/sindri/valheim")
  -V, --verbose count            verbosity
  -v, --version                  version for valheimw
      --world string             Valheim server -world (default "sindri")
```

## If you don't want to use `valheimw`...

[`boiler`](boiler.md) is able to help.
