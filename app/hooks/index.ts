import React from "react";
import { getSteamapps, SteamappSummary } from "~/client";

export function useSteamapps(limit: number = 10): [Array<SteamappSummary>, Error | undefined, boolean, () => void, boolean] {
    const [steamapps, setSteamapps] = React.useState<Array<SteamappSummary>>([]);
    const [cont, setContinue] = React.useState<string>();
    const [loading, setLoading] = React.useState(true);
    const [init, setInit] = React.useState(false);
    const [err, setErr] = React.useState<Error>();

    const more = React.useCallback(() => {
        setLoading(true);

        return getSteamapps({ continue: cont, limit })
            .then(res => {
                setSteamapps(s => [...s, ...res.steamapps]);
                setContinue(res.continue);
            })
            .catch((err) => {
                if (err instanceof Error) {
                    setErr(err);
                } else if (err instanceof Response) {
                    setErr(new Error(`${err.status}: ${err.statusText}`));
                } else {
                    setErr(new Error(err));
                }
            })
            .finally(() => {
                setLoading(false);
            });
    }, [setLoading, cont, limit, setSteamapps, setContinue]);

    React.useEffect(() => {
        if (init) {
            more();
        } else {
            setInit(true)
        }
    }, [init, more, setInit]);

    return [steamapps, err, !!cont, more, loading]
}
