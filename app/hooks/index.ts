import React from "react";
import { getSteamapp, getSteamapps, Steamapp, SteamappSummary } from "~/client";

export function useError() {
  const [error, setError] = React.useState<Error>();

  React.useEffect(() => {
    if (error) {
      alert(`${error}.`);
    }
  }, [error]);

  const handleError = React.useCallback(
    (err: unknown) => {
      if (err instanceof Error) {
        setError(err);
      } else if (err instanceof Response) {
        setError(new Error(`${err.status}: ${err.statusText}`));
      } else {
        setError(new Error(`${err}`));
      }
    },
    [setError],
  );

  return handleError;
}

export type UseSteamappsOpts = {
  handleErr?: (err: unknown) => void;
  steamapps?: Array<SteamappSummary | Steamapp>;
  token?: string;
};

export type GetSteamappDetailsOpts =
  | { appID: number; branch?: string; index?: never }
  | { appID?: never; branch?: never; index: number };

export function useSteamapps({
  steamapps: initialSteamapps = [],
  token: initialToken,
}: UseSteamappsOpts) {
  const [steamapps, setSteamapps] =
    React.useState<Array<SteamappSummary | Steamapp>>(initialSteamapps);
  const [token, setToken] = React.useState(initialToken);

  const getMoreSteamapps = React.useCallback(() => {
    return getSteamapps({ token }).then((res) => {
      setSteamapps((s) => [
        ...s,
        ...res.steamapps.filter(
          (app) =>
            !s.some(
              (existing) =>
                existing.app_id === app.app_id &&
                existing.branch === app.branch,
            ),
        ),
      ]);
      setToken(res.token);
    });
  }, [setSteamapps, setToken, token]);

  React.useEffect(() => {
    if (steamapps.length === 0) {
      getMoreSteamapps();
    }
  }, [getMoreSteamapps, steamapps]);

  const getSteamappDetails = React.useCallback(
    (opts: GetSteamappDetailsOpts) => {
      // Must explicitly check undefined because an index of 0 is valid.
      const steamapp =
        opts.index !== undefined
          ? steamapps[opts.index]
          : steamapps.find(
              (s) =>
                s.app_id === opts.appID &&
                (!opts.branch || s.branch === opts.branch),
            );

      if (steamapp && !(steamapp as Steamapp).base_image) {
        return getSteamapp(steamapp.app_id, steamapp.branch).then((s) => {
          setSteamapps((ss) => {
            if (opts.index !== undefined) {
              const newSteamapps = [...ss];
              newSteamapps[opts.index] = s;
              return newSteamapps;
            }

            return ss.map((existing) => {
              if (
                existing.app_id === s.app_id &&
                (!opts.branch || s.branch === opts.branch)
              ) {
                return s;
              }

              return existing;
            });
          });

          return s;
        });
      }

      return Promise.resolve(steamapp as Steamapp);
    },
    [steamapps, setSteamapps],
  );

  return {
    steamapps,
    getSteamappDetails,
    getMoreSteamapps: token ? getMoreSteamapps : undefined,
  };
}
