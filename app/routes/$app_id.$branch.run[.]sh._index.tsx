import { LoaderFunctionArgs } from "@remix-run/node";
import { getSteamapp } from "~/client";
import { imageRef, runScript } from "~/lib";

export function loader({ params, request }: LoaderFunctionArgs) {
  const { app_id: rawAppID, branch } = params;

  if (!rawAppID) {
    return new Response(null, {
      status: 400,
    });
  }

  const appID = parseInt(rawAppID, 10);
  const url = new URL("/", request.url);

  return getSteamapp(appID, branch)
    .then((steamapp) => {
      return runScript({
        steamapp,
        // docker cannot connect to the same localhost as the user to access the Dockerfile,
        // so we just inline it in the script in that case.
        inline: url.hostname === "localhost",
        ref: imageRef({ steamapp, registry: url.host }),
        url,
      });
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
