import React from "react";
import { BsClipboard, BsClipboardCheck } from "react-icons/bs";
import { MdFileDownload, MdFileDownloadDone } from "react-icons/md";
import { SteamappUpsert } from "~/client";
import { dockerfileFromSteamapp } from "~/lib";
import { DivIfProps } from "./div-if-props";

export type DockerfilePreviewProps = {
  steamapp: SteamappUpsert;
} & React.DetailedHTMLProps<
  React.HTMLAttributes<HTMLDivElement>,
  HTMLDivElement
>;

export function DockerfilePreview({
  steamapp,
  ...rest
}: DockerfilePreviewProps) {
  const [copied, setCopied] = React.useState(false);
  const [downloaded, setDownloaded] = React.useState(false);

  const dockerfile = dockerfileFromSteamapp(steamapp).toString();

  function handleCopy() {
    navigator.clipboard.writeText(dockerfile);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  async function handleDownload() {
    const blob = new Blob([dockerfile.concat("\n")], {
      type: "text/plain",
    });

    // Try to download using https://developer.mozilla.org/en-US/docs/Web/API/Window/showSaveFilePicker.
    if ("showSaveFilePicker" in window) {
      try {
        // eslint-disable-next-line
        const handle = await (window as any).showSaveFilePicker({
          suggestedName: "Dockerfile",
        });
        const writable = await handle.createWritable();
        await writable.write(blob);
        await writable.close();
        return;
      } catch (_) {
        /* */
      }
    }

    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "Dockerfile";
    document.body.appendChild(a);
    a.click();
    setDownloaded(true);
    setTimeout(() => setDownloaded(false), 2000);
    setTimeout(() => {
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }, 0);
  }

  return (
    <DivIfProps {...rest}>
      <div className="relative bg-black font-mono flex p-4 overflow-auto rounded border border-gray-500 cursor-text">
        <div className="text-right text-gray-500 pr-4 select-none">
          {dockerfile.split("\n").map((_, i) => (
            <div key={i} className="h-5">
              {i + 1}
            </div>
          ))}
        </div>
        <div className="whitespace-pre text-white">
          {dockerfile.split("\n").map((line, i) =>
            line.startsWith("#") ? (
              <div key={i} className="text-green-800 h-5">
                {line}
              </div>
            ) : (
              <div key={i} className="h-5">
                {line.split(" ").map((word, j, arr) => {
                  return word.match(/^[A-Z]+$/) && j === 0 ? (
                    <span key={j} className="text-pink-600">
                      {word}{" "}
                    </span>
                  ) : (
                    `${word}${j === arr.length - 1 ? "" : " "}`
                  );
                })}
              </div>
            ),
          )}
        </div>
        <button
          onClick={handleDownload}
          className="text-white hover:text-gray-500 p-2 absolute top-2 right-12"
          disabled={downloaded}
        >
          {downloaded ? <MdFileDownloadDone /> : <MdFileDownload />}
        </button>
        <button
          onClick={handleCopy}
          className="absolute top-2 right-2 text-white hover:text-gray-500 p-2"
          disabled={copied}
        >
          {copied ? <BsClipboardCheck /> : <BsClipboard />}
        </button>
      </div>
    </DivIfProps>
  );
}
