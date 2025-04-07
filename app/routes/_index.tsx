import type { MetaFunction } from "@remix-run/node";
import React from "react";
import { BsClipboard, BsClipboardCheck } from 'react-icons/bs';
import { useSteamapps } from "~/hooks";

export const meta: MetaFunction = () => {
  return [
    { title: "Sindri" },
  ];
};

export default function Index() {
  const [steamapps, err, _, loading] = useSteamapps();
  const [index, setIndex] = React.useState(0);

  React.useEffect(() => {
    if (steamapps.length && steamapps.length > 1) {
      const timeout = setInterval(
        () => setIndex(i => (i+1)%steamapps.length),
        2000,
      );

      return () => clearTimeout(timeout);
    }
  }, [steamapps, setIndex]);

  const tag = steamapps && steamapps.length > 0 && steamapps[index].branch || "latest";
  const branch = tag === "latest" ? "public" : tag;
  const steamapp = steamapps && steamapps.length > 0 && steamapps[index];
  const command = steamapp && `docker run sindri.frantjc.cc/${steamapp.app_id.toString()}:${tag}`

  const [copied, setCopied] = React.useState(false);

  const handleCopy = () => {
    if (command) {
      navigator.clipboard.writeText(command);
      setCopied(true);
      const timeout = setTimeout(() => setCopied(false), 1331);
      return () => clearTimeout(timeout);
    }
  };

  return (
    <div className="grid grid-cols-1">
      <div className="flex items-end h-24">
        <p>Run the...</p>
      </div>
    </div>
  );
}
