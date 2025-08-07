import { LoaderFunctionArgs } from "@remix-run/node";
import { getSteamapp } from "~/client";
import { dockerfileFromSteamapp } from "~/lib";

export function loader({ params, request }: LoaderFunctionArgs) {
  const { app_id: rawAppID, branch } = params;

  if (!rawAppID) {
    return new Response(null, {
      status: 400,
    });
  }

  const appID = parseInt(rawAppID, 10);
  const url = new URL(request.url.split("/").slice(0, -1).join("/"));
  const tag = branch ? `:${branch}` : "";
  const ref = `${url.host}/${appID}${tag}`;

  return getSteamapp(appID, branch)
    .then((steamapp) => {
      // docker cannot connect to the same localhost as the user to access the Dockerfile,
      // so we just inline it in the script.
      if (url.hostname === "localhost") {
        const dockerfile = dockerfileFromSteamapp(steamapp);

        return [
          "#!/bin/sh",
          `cat <<EOF | docker build -f- -t ${ref} .`,
          dockerfile,
          "EOF",
          "docker run"
            .concat(
              steamapp.ports
                ? steamapp.ports
                    .map((port) => ` -p ${port.port}:${port.port}`)
                    .join("")
                : "",
            )
            .concat(` ${ref}`),
        ]
          .join("\n")
          .concat("\n");
      }

      return [
        "#!/bin/sh",
        `docker build -f ${url}/dockerfile -t ${ref} .`,
        "docker run"
          .concat(
            steamapp.ports
              ? steamapp.ports
                  .map((port) => ` -p ${port.port}:${port.port}`)
                  .join("")
              : "",
          )
          .concat(` ${ref}`),
      ]
        .join("\n")
        .concat("\n");
    })
    .then((script) => {
      return new Response(script, {
        headers: {
          "Content-Disposition": "attachment; filename=run.sh",
          "Content-Type": "text/plain",
        },
      });
    });
}
