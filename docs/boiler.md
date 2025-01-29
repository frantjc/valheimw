# Boiler

`boiler` is Sindri's crown jewel. Inspired by [Nixery](https://nixery.dev/), it is a read-only container registry for pulling images with Steam apps pre-installed on them. The base of the images is `debian:stable-slim`. Images are non-root and `steamcmd` is never installed on them, so there's no leftover files from it on the image's filesystem or in its layers. Images are built on-demand rather than being stored, waiting to be pulled.

The image's name refers to a Steam app ID. Check out [SteamDB](https://steamdb.info/) to find your Steam app ID if you do not already know it.

The image's tag maps to the Steam app's branch, except the specific case of the default tag "latest" which maps to the default Steam app branch "public".

Layers and manifests are cached after being pulled via tag so that subsequent pulls via digest will function and be snappy. Subsequent pulls via tag will cause `boiler` to rebuild the container image to check if a new build has been released on the given branch. Such pulls are still faster than the first, especially if a new build has not been released because no cacheing would need to be done.

Steam apps sometimes have entrypoints that are non-configurable without editing files that they provide and frequently have additional system dependencies that need to be installed. `boiler` uses a hardcoded database to automatically fix its container images of some Steam apps before returning the images to the puller.

Although I am open to the idea, there is currently no public instance of `boiler`. I would like one to exist with a non-hardcoded, community-maintained database to automatically fix the container images of all Steam apps whose out-of-the-box configuration is lacking. Until then, you must run your own. Thankfully, doing so is easy.

Taking the Valheim server as an example of how `boiler` could be used, consider a directory with the following `docker-compose.yml`:

```yml
services:
  boiler:
    image: ghcr.io/frantjc/boiler
    ports:
      - 5000:5000
  valheim:
    image: localhost:5000/896660
    ports:
      - 2456:2456/udp
    depends_on:
      - boiler
```

> The Valheim server is one of a few Steam apps that is included in the hardcoded database, so it works out of the box.

> "896660" refers to the Steam app ID of the Valheim server.

First, run `boiler` in the background. We will use it to pre-build a container image with the Valheim server installed:

```sh
docker compose up --detach boiler
```

Next, build and run the Valheim server. This will pull a minimal container image with it pre-installed from `boiler` and then run the Palworld server container:

```sh
docker compose up --detach valheim
```

When this command is ran, `docker` will pull the Valheim server container image by making a series of HTTP requests to the `boiler` ran in the previous step. To satisfy those HTTP requests, `boiler` will download `steamcmd` and use it to build and cache the various manifests and blobs of the container image. As a result, the pull can take some time, especially on the first run when `boiler` has not cached `steamcmd` or the Steam app. After the pull is complete, the Valheim server will run.
