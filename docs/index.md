Sindri began as a tool specifically for managing a modded Valheim server in a container. As such, it's name originated from [Norse mythology](https://en.wikipedia.org/wiki/Sindri_(mythology)).

Since then, it has grown into a more generalized form as a toolkit for turning Steam app servers into container images--modded or otherwise. Sindri evolved this way because every open source container images I've came across for Steam app servers is bloated with all kinds of additional, unnecessary software with their own nuances to tease out and I am tired of building and pushing my own container images. While it still boasts a tool that is an especially helpful wrapper around the Valheim server, it also includes other tools to support efforts building minimal container images for any Steam app servers.

These tools include:

- [`boiler`](boiler.md), a container image for a read-only container registry for pulling images with Steam apps pre-installed on them, built on-demand.
- [`valheimw`](valheim.md), a container image containing a wrapper around the Valheim server that manages its mods via [thunderstore.io](https://valheim.thunderstore.io/) and runs an HTTP server alongside it to provide additional functionality.
- [`mist`](mist.md), a CLI tool for use in `Dockerfile`s to install Steam apps, Steam Workshop items, and [thunderstore.io](https://thunderstore.io/) mods.
