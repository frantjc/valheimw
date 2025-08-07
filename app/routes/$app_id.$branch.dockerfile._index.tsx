import { LoaderFunctionArgs } from "@remix-run/node";
import { getSteamapp } from "~/client";
import { dockerfileFromSteamapp } from "~/lib";

export function loader({ params }: LoaderFunctionArgs) {
  const { app_id: rawAppID, branch = "public" } = params;

  if (!rawAppID) {
    return new Response(null, {
      status: 400,
    });
  }

  const appID = parseInt(rawAppID, 10);

  return getSteamapp(appID, branch)
    .then(dockerfileFromSteamapp)
    .then((dockerfile) => {
      return new Response(dockerfile.toString().concat("\n"), {
        headers: {
          "Content-Disposition": "attachment; filename=Dockerfile",
          "Content-Type": "text/plain",
        },
      });
    });
}
