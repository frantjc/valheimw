import React from "react";
import { IoMdAdd, IoMdClose } from "react-icons/io";
import { FiSend } from "react-icons/fi";
import { SteamappUpsert } from "~/client";
import { DockerfilePreview } from "./dockerfile-preview";

export type SteamappFormProps = Omit<
  React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement>,
  "onChange" | "onSubmit"
> & {
  steamapp: SteamappUpsert;
  editing?: boolean;
  onChange: (_: SteamappUpsert) => void;
  onSubmit: (_: SteamappUpsert) => void;
};

export function SteamappForm({
  steamapp,
  editing = false,
  onChange,
  onSubmit,
  ...rest
}: SteamappFormProps) {
  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    onSubmit(steamapp);
  }

  const isFormValid = () => {
    const hasValidAppId = steamapp.app_id > 0 && steamapp.app_id % 10 === 0;
    const hasBranch =
      !!steamapp.branch?.trim() && steamapp.branch.trim() !== "public";
    const hasBetaPassword = !!steamapp.beta_password?.trim();
    return hasValidAppId && (!hasBranch || hasBetaPassword);
  };

  const isBetaPasswordRequired = () => {
    const branch = steamapp?.branch?.trim();
    return !!branch && branch !== "public";
  };

  return (
    <div {...rest}>
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <StringInput
          steamapp={steamapp}
          onChange={onChange}
          title="App ID *"
          field="app_id"
          type="number"
          required
          disabled={editing}
        />
        <StringInput
          steamapp={steamapp}
          onChange={onChange}
          title="Base Image"
          field="base_image"
        />
        <StringArrayInput
          steamapp={steamapp}
          onChange={onChange}
          title="APT Packages"
          field="apt_packages"
          placeholder="curl"
        />
        <StringInput
          steamapp={steamapp}
          onChange={onChange}
          title="Base Image"
          field="base_image"
          placeholder="default"
        />
        <div>
          <div className="flex flex-col gap-2">
            <label htmlFor="platform_type" className="text-sm font-medium">
              Platsteamapp Type
            </label>
            <select
              id="platform_type"
              className="grow p-2 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={steamapp.platform_type || ""}
              onChange={(e) =>
                onChange({
                  ...steamapp,
                  platform_type: e.target.value,
                })
              }
            >
              <option value="">Select plat...</option>
              <option value="linux">Linux</option>
              <option value="windows">Windows</option>
              <option value="macos">macOS</option>
            </select>
          </div>
        </div>
        <StringArrayInput
          steamapp={steamapp}
          onChange={onChange}
          title="Executables"
          field="execs"
          placeholder="ln -s /home/steam/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so"
        />
        <StringArrayInput
          steamapp={steamapp}
          onChange={onChange}
          title="Entrypoint"
          field="entrypoint"
          placeholder="/home/steam/start_server.sh"
        />
        <StringArrayInput
          steamapp={steamapp}
          onChange={onChange}
          title="Command"
          field="cmd"
        />
        <StringInput
          steamapp={steamapp}
          onChange={onChange}
          title="Branch"
          field="branch"
          disabled={editing}
        />
        <StringInput
          steamapp={steamapp}
          onChange={onChange}
          title={`Beta Password${isBetaPasswordRequired() ? " *" : ""}`}
          field="beta_password"
          required={isBetaPasswordRequired()}
          disabled={!isBetaPasswordRequired()}
        />
        <button
          type="submit"
          disabled={!isFormValid()}
          className={`${!isFormValid() ? "hover:cursor-not-allowed" : "hover:text-gray-500"} flex justify-center items-center gap-2 p-2`}
        >
          <FiSend />
          Submit
        </button>
      </form>
    </div>
  );
}

function StringInput({
  title,
  placeholder,
  required = false,
  disabled = false,
  type = "text",
  field,
  steamapp,
  onChange,
  ...rest
}: {
  title: string;
  placeholder?: string;
  required?: boolean;
  disabled?: boolean;
  type?: React.HTMLInputTypeAttribute;
  field: "app_id" | "base_image" | "branch" | "beta_password";
  steamapp: SteamappUpsert;
  onChange: (_: SteamappUpsert) => void;
} & Omit<
  React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement>,
  "onChange"
>) {
  return (
    <div {...rest}>
      <div className="flex flex-col gap-2">
        <label htmlFor={field} className="text-sm font-medium">
          {title}
        </label>
        <input
          id={field}
          type={type}
          required={required}
          disabled={disabled}
          placeholder={placeholder}
          className={`${disabled ? "bg-gray-400 cursor-not-allowed " : ""}grow p-2 rounded focus:outline-none focus:ring-2 focus:ring-blue-500`}
          value={steamapp[field] || ""}
          onChange={(e) =>
            onChange({
              ...steamapp,
              [field]:
                field === "app_id" ? parseInt(e.target.value) : e.target.value,
            })
          }
        />
      </div>
    </div>
  );
}

function StringArrayInput({
  title,
  placeholder,
  field,
  steamapp,
  onChange,
  ...rest
}: {
  title: string;
  placeholder?: string;
  field: "apt_packages" | "execs" | "entrypoint" | "cmd";
  steamapp: SteamappUpsert;
  onChange: (_: SteamappUpsert) => void;
} & Omit<
  React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement>,
  "onChange"
>) {
  const [input, setInput] = React.useState("");

  function handleAdd() {
    if (input.trim() && !(steamapp[field] ?? []).includes(input.trim())) {
      onChange({
        ...steamapp,
        [field]: [...(steamapp[field] ?? []), input.trim()],
      });
      setInput("");
    }
  }

  function handleRemove(index: number) {
    onChange({
      ...steamapp,
      [field]: (steamapp[field] ?? []).filter((_, i) => i !== index),
    });
  }

  return (
    <div {...rest}>
      <div className="flex flex-col gap-2">
        <label htmlFor={field} className="text-sm font-medium">
          {title}
        </label>
        <div className="flex flex-row gap-2">
          <input
            type="text"
            placeholder={placeholder}
            className="grow p-2 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                e.stopPropagation();
                handleAdd();
              }
            }}
          />
          <button onClick={handleAdd} className="p-2 hover:text-gray-500">
            <IoMdAdd />
          </button>
        </div>
        {(steamapp[field] ?? []).length > 0 && (
          <div className="flex flex-wrap gap-2">
            {(steamapp[field] ?? []).map((val, index) => (
              <span
                key={index}
                className="bg-gray-700 px-3 p-1 rounded-full text-sm flex items-center gap-2"
              >
                {val}
                <button
                  type="button"
                  onClick={() => handleRemove(index)}
                  className="hover:text-gray-500"
                >
                  <IoMdClose />
                </button>
              </span>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export function SteamappFormWithDockerfilePreview({
  steamapp,
  editing,
  onSubmit,
  onChange,
  ...rest
}: SteamappFormProps) {
  return (
    <div {...rest}>
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-8">
        <SteamappForm
          steamapp={steamapp}
          editing={editing}
          onSubmit={onSubmit}
          onChange={onChange}
        />
        <DockerfilePreview steamapp={steamapp} />
      </div>
    </div>
  );
}
