Sindri began as a tool specifically for managing a modded Valheim server in a container. As such, it's name originated from [Norse mythology](https://en.wikipedia.org/wiki/Sindri_(mythology)).

Since then, it has grown into a more generalized form as a toolkit for turning Steam app servers into a container image--modded or otherwise. While it still boasts a tool that is an especially helpful wrapper around the Valheim server, it has several other tools that should support efforts in building a minimal container images for any Steam app as well as Steam app servers of other games for my own use.

These tools include:

- [`boiler`](boiler.md), a read-only container registry for pulling images with Steam apps pre-installed on them.
- [`valheimw`](valheim.md), a container image containing a wrapper around the Valheim server that manages its mods via [thunderstore.io](https://valheim.thunderstore.io/) and runs an HTTP server alongside it to provide additional functionality.
- [`corekeeper`](corekeeper.md), a container image containing the Core Keeper server.
- [`mist`](mist.md), a CLI tool for use in Dockerfiles to install Steam apps, Steam Workshop items, and [thunderstore.io](https://thunderstore.io/) mods.
- _and more_.
