import React from "react";
import { BsClipboard, BsClipboardCheck } from "react-icons/bs";
import { DivIfProps } from "./div-if-props";

export type TerminalCommandProps = {
  command: string;
} & React.DetailedHTMLProps<
  React.HTMLAttributes<HTMLDivElement>,
  HTMLDivElement
>;

export function TerminalCommand({ command, ...rest }: TerminalCommandProps) {
  const [copied, setCopied] = React.useState(false);

  const handleCopy = React.useCallback(
    (text: string) => {
      return navigator.clipboard.writeText(text).then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      });
    },
    [setCopied],
  );

  return (
    <DivIfProps {...rest}>
      <pre className="bg-black flex p-2 px-4 rounded items-center justify-between w-full border border-gray-500">
        <code className="font-mono text-white p-1 overflow-auto pr-4">
          <span className="pr-2 text-gray-500">$</span>
          {command}
        </code>
        {command && (
          <button
            onClick={() =>
              handleCopy(command).catch(rest.onError || rest.onErrorCapture)
            }
            className="text-white hover:text-gray-500 p-2"
          >
            {copied ? <BsClipboardCheck /> : <BsClipboard />}
          </button>
        )}
      </pre>
    </DivIfProps>
  );
}
