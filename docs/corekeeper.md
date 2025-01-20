# Core Keeper

Sindri has a pre-built container image for the Core Keeper server for my own use because, unlike Docker, Kubernetes does not support building container images, only running them.

This container image gets built using [`boiler`](boiler.md) and takes care of installing the Core Keeper server's additional dependencies, smoothing out some of its quirks and ensuring that it does not run as root.

Consider a directory with the following `docker-compose.yml`:

```yml
services:
  corekeeper:
    image: ghcr.io/frantjc/corekeeper
    volumes:
      - ./save:/home/boil/.config/unity3d/Pugstorm/Core Keeper/DedicatedServer
```

This `docker-compose.yml` runs the Core Keeper server. To use it, place it in a directory and run the following command there:

```sh
docker compose up
```

> If Core Keeper errors with `Segmentation fault (core dumped)`, you have likely ran into a permissions issue. Run `chmod -R 777 ./save` and try again.

The server's save data will be stored in `./save`.

Notably, the Core Keeper server does not any ports exposed, instead using _magic_ to allow players to connect to the server.

Once the container finishes starting up, the game ID will be in its logs and can be used to connect to the server.
