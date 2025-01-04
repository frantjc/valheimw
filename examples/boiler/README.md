# boiler

First, run `boiler`:

```sh
docker compose up -d boiler
```

Next, run the Valheim server:

```sh
docker compose up valheim
```

When this command is ran, `docker` will pull the Valheim server container image by making a series of HTTP requests to the `boiler` ran in the previous step. To satisfy those HTTP requests, `boiler` will download `steamcmd` and use it to build the container image. As a result, the pull can take some time, especially on the first run when `boiler` has not cached `steamcmd` or any Steamapps. After the pull is complete, the Valheim server will run.
