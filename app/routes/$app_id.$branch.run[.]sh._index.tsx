import { LoaderFunctionArgs } from "@remix-run/node";
import { getSteamapp } from "~/client";

export function loader({ params }: LoaderFunctionArgs) {
  const { app_id: rawAppID, branch } = params;

  if (!rawAppID) {
    return new Response(null, {
      status: 400,
    });
  }

  // TODO(frantjc): Source from Host header.
  const scheme = "http";
  const host = "localhost:3000";
  const tag = `${host}/${rawAppID}${branch ? `:${branch}` : ""}`;

  const appID = parseInt(rawAppID, 10);

  return getSteamapp(appID, branch).then((steamapp) => {
    const script = [
      "#!/bin/sh",
      `docker build --file ${scheme}://${host}/${rawAppID}/${branch}/dockerfile --tag ${tag} .`,
      `docker run ${tag}`,
    ]
      .join("\n")
      .concat("\n");

    return new Response(script, {
      headers: {
        "Content-Disposition": "attachment; filename=run.sh",
        "Content-Type": "text/plain",
      },
    });
  });
}
