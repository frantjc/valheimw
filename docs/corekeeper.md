# Core Keeper

Consider a directory with the following `docker-compose.yml`:

```yml
services:
  buildkitd:
    image: moby/buildkit:rootless
    security_opt:
      - seccomp=unconfined
      - apparmor=unconfined
    command:
      - --addr
      - tcp://0.0.0.0:1234
      - --oci-worker-no-process-sandbox
  boiler:
    image: ghcr.io/frantjc/boiler
    command:
      - --buildkitd=tcp://buildkitd:1234
    ports:
      - 5000:5000
    depends_on:
      - buildkitd
  corekeeper:
    image: localhost:5000/1963720
    volumes:
      - ./saves:/home/steam/.config/unity3d/Pugstorm/Core Keeper/DedicatedServer
    depends_on:
      - boiler
```

> The Core Keeper server is one of a few Steamapps that is included in the hardcoded database, so it works out of the box.

> "1963720" refers to the Steamapp ID of the Core Keeper server.

This `docker-compose.yml` runs the Core Keeper server. To use it, place it in a directory and run the following command there:

```sh
docker compose up --detach boiler
```

Next, build and run the Core Keeper server. This will pull a minimal container image with it pre-installed from Boiler and then run the Core Keeper server container:

```sh
docker compose up --detach corekeeper
```

> If Core Keeper errors with `Segmentation fault (core dumped)`, you have likely ran into a permissions issue. Run `chmod -R 777 ./save` and try again.

The server's save data will be stored in `./saves`.

Notably, the Core Keeper server does not need any ports exposed, instead using _magic_ to allow players to connect to the server.

Once the container finishes starting up, the game ID will be in its logs and can be used to connect to the server.
