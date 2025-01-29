# Palworld

Consider a directory with the following `docker-compose.yml`:

```yml
services:
  buildkitd:
    image: moby/buildkit
    privileged: true
    command:
      - --addr
      - tcp://0.0.0.0:1234
  boiler:
    image: ghcr.io/frantjc/boiler
    ports:
      - 5000:5000
    depends_on:
      - buildkitd
  palworld:
    image: localhost:5000/2394010
    ports:
      - 8211:8211/udp
    depends_on:
      - boiler
```

> The Palworld server is one of a few Steam apps that is included in the hardcoded database, so it works out of the box.

> "2394010" refers to the Steam app ID of the Palworld server.

To run the Palworld server this way, run the following commands in the directory that the above files are placed in.

First, run [`boiler`](boiler.md) in the background. We will use it to pre-build a container image with the Palworld server installed:

```sh
docker compose up --detach boiler
```

Next, build and run the Palworld server. This will pull a minimal container image with it pre-installed from `boiler` and then run the Palworld server container:

```sh
docker compose up --detach palworld
```

Finally, `boiler` can be stopped:

```sh
docker compose down boiler
```
