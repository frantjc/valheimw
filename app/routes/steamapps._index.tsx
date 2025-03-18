import type { MetaFunction } from "@remix-run/node";
import React from "react";
import { getSteamapps, Steamapps } from "~/client";

export const meta: MetaFunction = () => {
  return [
    { title: "Sindri" },
  ];
};

export default function Main() {
  const [data, setData] = React.useState<Pick<Steamapps, "steamapps"> & { next?: number }>();

  React.useEffect(() => {
    getSteamapps({ offset: data?.next || 0 })
      .then(res => setData(cur => {
        return {
          continue: res.steamapps.length < res.limit ? undefined : res.offset + 1,
          steamapps: cur?.steamapps.concat(res.steamapps) || res.steamapps,
        }
      }));
  }, [setData, data?.next])

  return (
    <main>
      {JSON.stringify(data?.steamapps || [])}
    </main>
  );
}
