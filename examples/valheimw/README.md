# valheimw

First update the command to have your list of desired thunderstore.io mods and the `VALHEIM_PASSWORD` to your desired password in [`docker-compose.yml`](docker-compose.yml).

Next, Run `valheimw`:

```sh
docker compose up
```

`valheimw` will cache the Valheim server's save data, the Valheim server itself, and the mods in `./hack`.

> If you run into errors with, you have likely ran into a permissions issue. Run `chmod -R 777 ./hack` and try again.
