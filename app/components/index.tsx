import React from "react";
import { BsClipboard, BsClipboardCheck } from "react-icons/bs";
import { Steamapp } from "~/client";

type CodeModalProps = {
  open: boolean;
  onClose: () => void;
  steamapp?: Steamapp;
  lines?: number;
}

export function CodeModal({ steamapp, open, onClose, lines = 16 }: CodeModalProps) {
  const [copied, setCopied] = React.useState(false);

  const defn = generateContainerDefinition(steamapp);
  const codeLines: string[] = defn.split("\n");
  while (codeLines.length < lines) codeLines.push("");

  const handleCopy = () => {
    navigator.clipboard.writeText(codeLines.join("\n"));
    setCopied(true);
    const timeout = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(timeout);
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50">
      <div className="bg-white rounded shadow-lg min-w-[750px] max-w-[90vw]">
        <div className="flex justify-between items-center px-4 py-2 border-b border-gray-200">
          <span className="font-bold text-lg">Code View</span>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700 font-bold px-2 py-1 rounded"
          >
            Close
          </button>
        </div>
        <div className="relative flex">
          <pre className="select-none text-right text-gray-400 bg-gray-100 py-4 pl-4 pr-2 rounded-bl rounded-tl">
            {codeLines.map((_, i) => (
              <div key={i} className="h-5 leading-5">{i + 1}</div>
            ))}
          </pre>
          <pre className="relative bg-black text-white font-mono py-4 px-4 rounded-br rounded-tr overflow-x-auto w-full">
            <code
              className="block outline-none"
              contentEditable={false}
              style={{ userSelect: "text" }}
            >
              {codeLines.join("\n")}
            </code>
            <button
              onClick={handleCopy}
              className="absolute top-2 right-2 bg-blue-400 hover:bg-blue-600 text-white font-bold py-1 px-3 rounded flex items-center"
            >
              {copied ? <BsClipboardCheck className="h-4 w-4" /> : <BsClipboard className="h-4 w-4" />}
              <span className="ml-2">{copied ? "Copied!" : "Copy"}</span>
            </button>
          </pre>
        </div>
      </div>
    </div>
  );
}

export function generateContainerDefinition(steamapp: Steamapp | undefined): string {
  if (!steamapp) return "";

  const isBeta = steamapp.branch && steamapp.branch !== "public";
  if (isBeta && (!steamapp.beta_password || steamapp.beta_password.length === 0)) {
    throw new Error("Beta branch requires a beta_password, but none was provided");
  }

  const betaBranch = isBeta ? ` -beta ${steamapp.branch}` : "";
  const betaPassword = isBeta ? ` -betapassword ${steamapp.beta_password}` : "";

  const lines = [
    "FROM steamcmd/steamcmd AS steamcmd",
    "RUN groupadd --system steam \\",
    "  && useradd --system --gid steam --shell /bin/bash --create-home steam \\",
    "  && steamcmd \\",
    "    +force_install_dir /mnt \\",
    "    +login anonymous \\",
    `    @sSteamCmdForcePlatformType ${steamapp.platform_type} \\`,
    `    +app_update ${steamapp.app_id}${betaBranch}${betaPassword} \\`,
    "    +quit",
    "",
    "FROM " + steamapp.base_image,
    steamapp.apt_packages && steamapp.apt_packages.length
      ? `RUN apt-get update -y && apt-get install -y --no-install-recommends ${steamapp.apt_packages.join(" ")} && rm -rf /var/lib/apt/lists/* && apt-get clean`
      : "",
    "RUN groupadd --system steam \\",
    "  && useradd --system --gid steam --shell /bin/bash --create-home steam",
    "USER steam",
    "COPY --from=steamcmd /mnt /home/steam",
    steamapp.execs && steamapp.execs.length
      ? `RUN ${steamapp.execs.join(" && ")}`
      : "",
    "",
    steamapp.entrypoint && steamapp.entrypoint.length
      ? `ENTRYPOINT [${steamapp.entrypoint.map(e => `"${e}"`).join(", ")}]`
      : "",
    steamapp.cmd && steamapp.cmd.length
      ? `CMD [${steamapp.cmd.map(c => `"${c}"`).join(", ")}]`
      : "",
  ];

  return lines.join("\n");
}