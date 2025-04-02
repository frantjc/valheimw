import type { MetaFunction } from "@remix-run/node";
import React from "react";
import { getSteamapps, SteamappSummary } from "~/client";
import { Command } from "~/components";

export const meta: MetaFunction = () => {
  return [
    { title: "Sindri" },
  ];
};

export default function Main() {
  const [steamapps, setSteamapps] = React.useState<Array<SteamappSummary>>();
  const [index, setIndex] = React.useState(0);

  React.useEffect(() => {
    getSteamapps()
      .then(res => setSteamapps(res.steamapps));
  }, [setSteamapps])

  React.useEffect(() => {
    if (steamapps?.length && steamapps.length > 1) {
      const timeout = setTimeout(
        () => setIndex(i => i ? (i+1)%steamapps.length : 0),
        2000,
      );

      return () => clearTimeout(timeout);
    }
  }, [steamapps, setIndex])

  return (
    <main className="flex flex-col items-center justify-center p-4 min-h-screen bg-gray-900 text-white">
      {steamapps?.length && (
        <div className="bg-gray-800 text-white p-4 rounded-lg">
          <p className="text-lg">Run the {steamapps[index].name}:</p>
          <Command>{`docker run sindri.frantjc.cc/${steamapps[index].app_id.toString()}`}</Command>
        </div>
      )}
    </main>
  );
}
