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
        // Errors from the API _should_ look like '{"error":"error description"}'.
        throw new Response(null, {
          status: res.status,
          statusText: err.error || res.statusText,
        });
      });
  }

  return res;
}

export type SteamappSummary = {
	app_id: number;
  name: string;
  branch?: string;
  icon_url: string;
	date_created: Date;
  locked: boolean;
}

export type SteamappDetail = {
	base_image: string;
	apt_packages: Array<string>;
	launch_type: string;
	platform_type: string;
	execs: Array<string>;
	entrypoint: Array<string>;
	cmd: Array<string>;
}

export type Steamapp = SteamappSummary & SteamappDetail;

export type SteamappList = {
  continue?: string;
  steamapps: Array<SteamappSummary>;
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
    `http://localhost:5050`,
  ).toString();
}

export function getSteamapp(id: number, branch?: string): Promise<Steamapp> {
  return fetch(
    getUrl(`/api/v1/steamapps/${id}`.concat(branch ? `/branch=${branch}` : "")),
    {
      headers: {
        "Content-Type": "application/json",
      },
    },
  )
    .then(handleError)
    .then((res) => {
      return res.json() as Promise<Steamapp>;
    });
}

export function getSteamapps(
  {
    continue: cont,
    limit = 10,
  }: {
    continue?: string;
    limit?: number;
  } = {}
): Promise<SteamappList> {
  return fetch(
    getUrl(
      `/api/v1/steamapps?${
        new URLSearchParams(
          Object.entries({ continue: cont, limit })
            .reduce(
              (acc, [k, v]) =>
                v && v.toString()
                  ? {
                      ...acc,
                      [k]: v.toString(),
                    }
                  : acc,
              {},
            ),
        )
      }`,
    ),
    {
      headers: {
        "Content-Type": "application/json",
      },
    },
  )
    .then(handleError)
    .then(async (res) => {
      return res.json().then((steamapps) => {
        return {
          continue: res.headers.get("X-Continue-Token") || undefined,
          steamapps,
        }
      });
    });
}
