import type { MetaFunction } from "@remix-run/node";
import React from "react";
import { getSteamapps, SteamappList } from "~/client";

export const meta: MetaFunction = () => {
  return [
    { title: "Sindri" },
  ];
};

export default function Index() {
  const [data, setData] = React.useState<SteamappList>();

  React.useEffect(() => {
    getSteamapps({ continue: data?.continue })
      .then(res => setData(cur => {
        return {
          continue: res.continue,
          steamapps: cur?.steamapps.concat(res.steamapps) || res.steamapps,
        }
      }));
  }, [setData, data?.continue])

  return (
    <div>
      {JSON.stringify(data?.steamapps || [])}
    </div>
  );
}
