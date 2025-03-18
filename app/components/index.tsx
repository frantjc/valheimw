import React from "react";
import { BsClipboard, BsClipboardCheck } from 'react-icons/bs';

export type CommandProps = {
  children: string;
}

export function Command({ children }: CommandProps) {
  const [copied, setCopied] = React.useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(children);
    setCopied(true);
    const timeout = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(timeout);
  };

  return (
    <pre className="bg-gray-900 p-2 rounded mt-2 flex items-center justify-between w-full">
      <code>{children}</code>
      <button
        onClick={handleCopy}
        className="ml-4 bg-blue-500 hover:bg-blue-700 text-white font-bold py-1 px-2 rounded flex items-center"
      >
        {copied ? <BsClipboardCheck className="h-5 w-4" /> : <BsClipboard className="h-5 w-4" />}
      </button>
    </pre>
  )
}
