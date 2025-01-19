# Palworld

Sindri provides no special support for the Palworld server beyond this document describing how one could use Sindri to build and run the Palworld server in a container.

Consider a directory with the following `docker-compose.yml` and `Dockerfile`:

```yml
services:
  boiler:
    image: ghcr.io/frantjc/boiler
    ports:
      - 5000:5000
  palworld:
    build: .
    ports:
      - 8211:8211/udp
    depends_on:
      - boiler
```

```Dockerfile
FROM localhost:5000/2394010
USER root
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        xdg-utils \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean
USER boil
```

> "2394010" refers to the Steam app ID of the Palworld server.

To run the Palworld server this way, run the following commands in the directory that the above files are placed in.

First, run [`boiler`](boiler.md) in the background. We will use it to pre-build a container image with the Palworld server installed:

```sh
docker compose up --detach boiler
```

Next, build and run the Palworld server. This will pull a minimal container image with it pre-installed from `boiler`, install extra dependencies via the `Dockerfile` and then run the Palworld server container:

```sh
docker compose up --detach palworld
```

Finally, `boiler` can be stopped:

```sh
docker compose down boiler
```
