import type { MetaFunction } from "@remix-run/node";
import React from "react";

export const meta: MetaFunction = () => {
  return [
    { title: "Sindri" },
    { name: "description", content: "Welcome to Sindri!" },
  ];
};


function isSuccess(res: Response) {
  return 200 <= res.status && res.status < 300;
}

function isError(res: Response) {
  return !isSuccess(res);
}

function handleError(res: Response) {
  if (isError(res)) {
    return res
      .json()
      .catch(() => {
        throw new Response(null, {
          status: res.status,
          statusText: res.statusText,
        });
      })
      .then((err) => {
        // Errors from the API _should_ look like '{"message":"error description"}'.
        throw new Response(null, {
          status: res.status,
          statusText: err.message || res.statusText,
        });
      });
  }

  return res;
}

type Steamapp = {
	appID: number;
	dateCreated: Date;
	dateUpdated: Date;
	baseImage: string;
	aptPackages: Array<string>;
	launchType: string;
	platformType: string;
	execs: Array<string>;
	entrypoint: Array<string>;
	cmd: Array<string>;
}

function getSteamapp(id: number): Promise<Steamapp> {
  return fetch(`/api/v1/steamapps/${id}`)
    .then(handleError)
    .then((res) => {
      return res.json() as Promise<Steamapp>;
    });
}

export default function Index() {
  const [steamapp, setSteamapp] = React.useState<Steamapp>();

  React.useEffect(() => {
    getSteamapp(896660).then(setSteamapp);
  }, [setSteamapp]);

  return (
    <div>
      {JSON.stringify(steamapp)}
    </div>
  );
}
