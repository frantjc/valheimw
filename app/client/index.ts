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

export type Steamapp = {
  name: string;
  icon_url: string;
  locked: boolean;
	app_id: number;
	date_created: Date;
	date_updated: Date;
	base_image: string;
	apt_packages: Array<string>;
	launch_type: string;
	platform_type: string;
	execs: Array<string>;
	entrypoint: Array<string>;
	cmd: Array<string>;
}

export type Steamapps = {
  offset: number;
  limit: number;
  steamapps: Array<Pick<Steamapp, "app_id" | "date_created" | "date_updated" | "name" | "icon_url" | "locked">>;
}

// getUrl takes a path and returns the full URL
// that that resource can be accessed at. This
// cleverly works both in the browser and in NodeJS.
export function getUrl(path: string) {
  if (typeof process !== "object") {
    return path;
  } else if (process.env.STOKER_API_URL) {
    return new URL(path, process.env.STOKER_API_URL).toString();
  }

  return new URL(
    path,
    `http://localhost:${process.env.PORT || 3000}`,
  ).toString();
}

export function getSteamapp(id: number): Promise<Steamapp> {
  return fetch(getUrl(`/api/v1/steamapps/${id}`))
    .then(handleError)
    .then((res) => {
      return res.json() as Promise<Steamapp>;
    });
}

export function getSteamapps(offset: number = 0, limit: number = 10): Promise<Steamapps> {
  return fetch(getUrl(`/api/v1/steamapps${
    new URLSearchParams(Object.entries({ offset, limit }).reduce(
      (acc, [k, v]) =>
        v && v.toString()
          ? {
              ...acc,
              [k]: v.toString(),
            }
          : acc,
      {},
    ))
  }`))
    .then(handleError)
    .then((res) => {
      return res.json() as Promise<Steamapps>;
    });
}
