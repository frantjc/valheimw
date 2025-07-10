import React from "react";
import { BsClipboard, BsClipboardCheck } from "react-icons/bs";
import { Steamapp } from "../client";
import { generateContainerDefinition } from "../lib/code_modal";

type CodeModalProps = {
  open: boolean;
  onClose: () => void;
  steamapp?: Steamapp;
  lines?: number;
};

export function CodeModal({
  steamapp,
  open,
  onClose,
  lines = 16,
}: CodeModalProps) {
  const [copied, setCopied] = React.useState(false);

  const defn = generateContainerDefinition(steamapp);
  const codeLines: string[] = defn.split("\n");
  while (
    codeLines.length > 0 &&
    codeLines[codeLines.length - 1].trim() === ""
  ) {
    codeLines.pop();
  }
  while (codeLines.length < lines) codeLines.push("");

  const handleCopy = () => {
    navigator.clipboard.writeText(codeLines.join("\n"));
    setCopied(true);
    const timeout = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(timeout);
  };

  const handleDownload = () => {
    const blob = new Blob([codeLines.join("\n")], {
      type: "application/octet-stream",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "Dockerfile";
    document.body.appendChild(a);
    a.click();
    setTimeout(() => {
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }, 0);
  };

  return (
    <div
      className={`fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50 px-4 ${
        open ? "block" : "hidden"
      }`}
      onClick={onClose}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === "Escape" || e.key === "Enter" || e.key === " ") {
          onClose();
        }
      }}
    >
      <div
        className="bg-white rounded shadow-lg w-full max-w-screen-lg overflow-hidden cursor-default"
        onClick={(e) => e.stopPropagation()}
        role="presentation"
      >
        <div className="flex justify-between items-center px-4 py-2 border-b border-gray-700 bg-gray-800">
          <span className="font-bold text-lg text-white">Dockerfile</span>
          <div className="flex gap-2">
            <button
              onClick={handleDownload}
              className="bg-gray-700 hover:bg-gray-600 text-white font-bold px-3 py-1 rounded"
            >
              Download
            </button>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-white font-bold px-2 py-1 rounded"
            >
              Close
            </button>
          </div>
        </div>
        <div className="relative flex overflow-x-auto h-[50vh]">
          <div className="relative grid grid-cols-[2.5rem_1fr] overflow-x-auto w-full bg-black text-white font-mono py-4 px-4">
            <div className="text-right pr-2 select-none border-r border-gray-700">
              {codeLines.map((_, i) => (
                <div key={i} className="h-5 leading-5">
                  {i + 1}
                </div>
              ))}
            </div>
            <div className="whitespace-pre pl-4">
              {codeLines.map((line, i) => (
                <div key={i} className="h-5 leading-5">
                  {line}
                </div>
              ))}
            </div>
          </div>
          <button
            onClick={handleCopy}
            className="absolute top-2 right-2 bg-blue-400 hover:bg-blue-600 text-white font-bold py-2 px-3 rounded flex items-center"
          >
            {copied ? (
              <BsClipboardCheck className="h-4 w-4" />
            ) : (
              <BsClipboard className="h-4 w-4" />
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
