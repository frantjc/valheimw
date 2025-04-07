import React from "react";
import { getSteamapps, SteamappSummary } from "~/client";

export function useSteamapps(limit: number = 10): [Array<SteamappSummary>, Error | undefined, () => void, boolean] {
    const [steamapps, setSteamapps] = React.useState<Array<SteamappSummary>>([]);
    const [cont, setContinue] = React.useState<string>();
    const [loading, setLoading] = React.useState(true);
    const [init, setInit] = React.useState(false);
    const [err, setErr] = React.useState<Error>();

    const more = React.useCallback(() => {
        if (!init || cont) {
            setLoading(true);
    
            getSteamapps({ continue: cont, limit })
                .then(res => {
                    setSteamapps(s => [...s, ...res.steamapps]);
                    setContinue(res.continue);
                    setInit(true);
                })
                .catch(setErr)
                .finally(() => {
                    setLoading(false);
                });
        }
    }, [setLoading, setSteamapps, setContinue, setInit, cont, limit]);

    React.useEffect(() => {
        if (!init) {
            more();
        }
    }, [init, more])

    return [steamapps, err, more, loading]
}
