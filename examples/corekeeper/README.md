# corekeeper

Run the Core Keeper server:

```sh
docker compose up
```

The server's save data will be stored in `./hack`.

> If Core Keeper errors with `Segmentation fault (core dumped)`, you have likely ran into a permissions issue. Run `chmod -R 777 ./hack` and try again.
