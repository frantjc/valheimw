Sindri began as a tool specifically for managing a modded Valheim server in a container. As such, it's name originated from [Norse mythology](https://en.wikipedia.org/wiki/Sindri_(mythology)).

Since then, it has grown into a more generalized form as a toolkit for turning Steam app servers into container images--modded or otherwise. While it still boasts a tool that is an especially helpful wrapper around the Valheim server, it also contains several other tools that support efforts in building minimal container images for other Steam apps as well as Steam app servers for other games.

These tools include:

- [`boiler`](boiler.md), a read-only container registry for pulling images with Steam apps pre-installed on them on-demand.
- [`valheimw`](valheim.md), a container image containing a wrapper around the Valheim server that manages its mods via [thunderstore.io](https://valheim.thunderstore.io/) and runs an HTTP server alongside it to provide additional functionality.
- [`corekeeper`](corekeeper.md), a container image containing the Core Keeper server.
- [`mist`](mist.md), a CLI tool for use in `Dockerfile`s to install Steam apps, Steam Workshop items, and [thunderstore.io](https://thunderstore.io/) mods.
