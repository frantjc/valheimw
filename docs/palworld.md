# Palworld

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
  palworld:
    image: localhost:5000/2394010
    ports:
      - 8211:8211/udp
    depends_on:
      - boiler
```

> The Palworld server is one of a few Steamapps that is included in the hardcoded database, so it works out of the box.

> "2394010" refers to the Steamapp ID of the Palworld server.

To run the Palworld server this way, run the following commands in the directory that the above files are placed in.

First, run [Boiler](boiler.md) in the background. We will use it to pre-build a container image with the Palworld server installed:

```sh
docker compose up --detach boiler
```

Next, build and run the Palworld server. This will pull a minimal container image with it pre-installed from Boiler and then run the Palworld server container:

```sh
docker compose up --detach palworld
```

Finally, Boiler can be stopped:

```sh
docker compose down boiler
```
