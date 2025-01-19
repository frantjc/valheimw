# Boiler

`boiler` is Sindri's crown jewel. Inspired by [Nixery](https://nixery.dev/), it is a read-only container registry for pulling images with Steam apps pre-installed on them. The base of the images is `debian:stable-slim`. Images are non-root and `steamcmd` is never installed on them, so there's no leftover files from it on the image's filesystem or in its layers. Images are built on-demand rather than being stored, waiting to be pulled.

The image's name refers to a Steam app ID. Check out [SteamDB](https://steamdb.info/) to find your Steam app ID if you do not already know it.

The image's tag maps to the Steam app's branch, except the specific case of the default tag "latest" which maps to the default Steam app branch "public".

Layers and manifests are cached after being pulled via tag so that subsequent pulls via digest will function and be snappy. Subsequent pulls via tag will always cause `boiler` to rebuild the container image in case a new build has been released on the given branch, albiet a bit faster if a new build has not been released since no cacheing need take place in said scenario.

Taking the Valheim server as an example of how `boiler` could be used, consider a directory with the following `docker-compose.yml`:

```yml
services:
  boiler:
    image: ghcr.io/frantjc/boiler
    ports:
      - 5000:5000
  valheim:
    # The default tag is "latest" which gets mapped to the Steam app branch
    # "public". "896660" refers to the Steam app ID of the Valheim server.
    # 
    image: localhost:5000/896660
    # start_server_xterm.sh just execs start_server.sh with xterm,
    # so avoid the extra dependency on xterm by directly execing
    # start_server.sh.
    entrypoint:
      - /home/boil/steamapp/start_server.sh
    ports:
      - 2456:2456/udp
    depends_on:
      - boiler
```

First, run `boiler` in the background. We will use it to pre-build a container image with the Valheim server installed:

```sh
docker compose up --detach boiler
```

Next, build and run the Valheim server. This will pull a minimal container image with it pre-installed from `boiler`:

```sh
docker compose up --detach valheim
```

When this command is ran, `docker` will pull the Valheim server container image by making a series of HTTP requests to the `boiler` ran in the previous step. To satisfy those HTTP requests, `boiler` will download `steamcmd` and use it to build and cache the various manifests and blobs of the container image. As a result, the pull can take some time, especially on the first run when `boiler` has not cached `steamcmd` or the Steam app. After the pull is complete, the Valheim server will run.
