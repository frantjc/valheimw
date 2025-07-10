import React from "react";
import { postSteamapp, SteamappDetail } from "../client";

type AddFormProps = {
  onClose: () => void;
};

export function AddForm({ onClose }: AddFormProps) {
  const [formData, setFormData] = React.useState<
    SteamappDetail & { app_id: number; branch?: string }
  >({
    app_id: 0,
    base_image: "",
    apt_packages: [],
    beta_password: "",
    launch_type: "",
    platform_type: "",
    execs: [],
    entrypoint: [],
    cmd: [],
  });

  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const [aptPackageInput, setAptPackageInput] = React.useState("");
  const addAptPackage = () => {
    if (
      aptPackageInput.trim() &&
      !(formData.apt_packages ?? []).includes(aptPackageInput.trim())
    ) {
      setFormData((prev) => ({
        ...prev,
        apt_packages: [...(prev.apt_packages ?? []), aptPackageInput.trim()],
      }));
      setAptPackageInput("");
    }
  };

  const removeAptPackage = (index: number) => {
    setFormData((prev) => ({
      ...prev,
      apt_packages: (prev.apt_packages ?? []).filter((_, i) => i !== index),
    }));
  };

  const [execInput, setExecInput] = React.useState("");
  const addExec = () => {
    if (
      execInput.trim() &&
      !(formData.execs ?? []).includes(execInput.trim())
    ) {
      setFormData((prev) => ({
        ...prev,
        execs: [...(prev.execs ?? []), execInput.trim()],
      }));
      setExecInput("");
    }
  };

  const removeExec = (index: number) => {
    setFormData((prev) => ({
      ...prev,
      execs: (prev.execs ?? []).filter((_, i) => i !== index),
    }));
  };

  const [entrypointInput, setEntrypointInput] = React.useState("");
  const addEntrypoint = () => {
    if (
      entrypointInput.trim() &&
      !(formData.entrypoint ?? []).includes(entrypointInput.trim())
    ) {
      setFormData((prev) => ({
        ...prev,
        entrypoint: [...(prev.entrypoint ?? []), entrypointInput.trim()],
      }));
      setEntrypointInput("");
    }
  };

  const removeEntrypoint = (index: number) => {
    setFormData((prev) => ({
      ...prev,
      entrypoint: (prev.entrypoint ?? []).filter((_, i) => i !== index),
    }));
  };

  const [cmdInput, setCmdInput] = React.useState("");
  const addCmd = () => {
    if (cmdInput.trim() && !(formData.cmd ?? []).includes(cmdInput.trim())) {
      setFormData((prev) => ({
        ...prev,
        cmd: [...(prev.cmd ?? []), cmdInput.trim()],
      }));
      setCmdInput("");
    }
  };

  const removeCmd = (index: number) => {
    setFormData((prev) => ({
      ...prev,
      cmd: (prev.cmd ?? []).filter((_, i) => i !== index),
    }));
  };

  const handleCancel = () => {
    setFormData({
      app_id: 0,
      base_image: "",
      apt_packages: [],
      beta_password: "",
      launch_type: "",
      platform_type: "",
      execs: [],
      entrypoint: [],
      cmd: [],
    });

    setAptPackageInput("");
    setExecInput("");
    setEntrypointInput("");
    setCmdInput("");

    setLoading(false);
    setError(null);
    onClose();
  };

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setLoading(true);
    setError(null);

    const { branch, beta_password, ...body } = formData;
    
    postSteamapp(body.app_id, body, branch, beta_password)
      .then(() => {
        handleCancel();
      })
      .catch((err) => {
        if (err instanceof Response) {
          setError(err.statusText);
        } else if (err instanceof Error) {
          setError(err.message);
        } else {
          setError(String(err));
        }
      })
      .finally(() => {
        setLoading(false);
      });
  }

  const isFormValid = () => {
    const hasAppId = formData.app_id > 0;
    const hasBranch = formData.branch && formData.branch.trim() !== "";
    const hasBetaPassword =
      formData.beta_password && formData.beta_password.trim() !== "";

    return hasAppId && (!hasBranch || hasBetaPassword);
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="p-6 overflow-y-auto max-h-[calc(90vh-60px)]"
    >
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="space-y-4">
          <div>
            <label
              htmlFor="app_id"
              className="block text-sm font-medium mb-1"
            >
              App ID *
            </label>
            <input
              id="app_id"
              type="number"
              required
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
              value={formData.app_id || ""}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  app_id: parseInt(e.target.value) || 0,
                }))
              }
            />
          </div>
          <div>
            <label
              htmlFor="base_image"
              className="block text-sm font-medium mb-1"
            >
              Base Image
            </label>
            <input
              id="base_image"
              type="text"
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={formData.base_image || ""}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  base_image: e.target.value,
                }))
              }
            />
          </div>
          <div>
            <label
              htmlFor="apt_packages"
              className="block text-sm font-medium mb-1"
            >
              APT Packages
            </label>
            <div className="flex gap-2 mb-2">
              <input
                type="text"
                placeholder="curl"
                className="flex-1 px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={aptPackageInput}
                onChange={(e) => setAptPackageInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    addAptPackage();
                  }
                }}
              />
              <button
                type="button"
                onClick={addAptPackage}
                className="px-4 py-2 bg-blue-400 text-white rounded hover:bg-blue-600"
              >
                Add
              </button>
            </div>
            {(formData.apt_packages ?? []).length > 0 && (
              <div className="flex flex-wrap gap-2">
                {(formData.apt_packages ?? []).map((pkg, index) => (
                  <span
                    key={index}
                    className="bg-gray-100 px-3 py-1 rounded-full text-sm flex items-center gap-2"
                  >
                    {pkg}
                    <button
                      type="button"
                      onClick={() => removeAptPackage(index)}
                      className="text-red-500 hover:text-red-700 font-bold"
                    >
                      ×
                    </button>
                  </span>
                ))}
              </div>
            )}
          </div>
          <div>
            <label
              htmlFor="launch_type"
              className="block text-sm font-medium mb-1"
            >
              Launch Type
            </label>
            <input
              id="launch_type"
              type="text"
              placeholder="default"
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={formData.launch_type || ""}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  launch_type: e.target.value,
                }))
              }
            />
          </div>
          <div>
            <label
              htmlFor="platform_type"
              className="block text-sm font-medium mb-1"
            >
              Platform Type
            </label>
            <select
              id="platform_type"
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={formData.platform_type || ""}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  platform_type: e.target.value,
                }))
              }
            >
              <option value="">Select platform...</option>
              <option value="linux">Linux</option>
              <option value="windows">Windows</option>
              <option value="macos">macOS</option>
            </select>
          </div>
        </div>
        <div className="md:col-span-2">
          <label htmlFor="execs" className="block text-sm font-medium mb-1">
            Executables
          </label>
          <div className="flex gap-2 mb-2">
            <input
              type="text"
              placeholder="ln -s /home/steam/linux64/steamclient.so /usr/lib/x86_64-linux-gnu/steamclient.so"
              className="flex-1 px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={execInput}
              onChange={(e) => setExecInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  addExec();
                }
              }}
            />
            <button
              type="button"
              onClick={addExec}
              className="px-4 py-2 bg-blue-400 text-white rounded hover:bg-blue-600"
            >
              Add
            </button>
          </div>
          {(formData.execs ?? []).length > 0 && (
            <div className="flex flex-wrap gap-2">
              {(formData.execs ?? []).map((pkg, index) => (
                <span
                  key={index}
                  className="bg-gray-100 px-3 py-1 rounded-full text-sm flex items-center gap-2"
                >
                  {pkg}
                  <button
                    type="button"
                    onClick={() => removeExec(index)}
                    className="text-red-500 hover:text-red-700 font-bold"
                  >
                    ×
                  </button>
                </span>
              ))}
            </div>
          )}
        </div>
        <div className="md:col-span-2">
          <label
            htmlFor="entrypoints"
            className="block text-sm font-medium mb-1"
          >
            Entrypoints
          </label>
          <div className="flex gap-2 mb-2">
            <input
              type="text"
              className="flex-1 px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={entrypointInput}
              onChange={(e) => setEntrypointInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  addEntrypoint();
                }
              }}
            />
            <button
              type="button"
              onClick={addEntrypoint}
              className="px-4 py-2 bg-blue-400 text-white rounded hover:bg-blue-600"
            >
              Add
            </button>
          </div>
          {(formData.entrypoint ?? []).length > 0 && (
            <div className="flex flex-wrap gap-2">
              {(formData.entrypoint ?? []).map((pkg, index) => (
                <span
                  key={index}
                  className="bg-gray-100 px-3 py-1 rounded-full text-sm flex items-center gap-2"
                >
                  {pkg}
                  <button
                    type="button"
                    onClick={() => removeEntrypoint(index)}
                    className="text-red-500 hover:text-red-700 font-bold"
                  >
                    ×
                  </button>
                </span>
              ))}
            </div>
          )}
        </div>
        <div className="md:col-span-2">
          <label
            htmlFor="commands"
            className="block text-sm font-medium mb-1"
          >
            Commands
          </label>
          <div className="flex gap-2 mb-2">
            <input
              type="text"
              className="flex-1 px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={cmdInput}
              onChange={(e) => setCmdInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  addCmd();
                }
              }}
            />
            <button
              type="button"
              onClick={addCmd}
              className="px-4 py-2 bg-blue-400 text-white rounded hover:bg-blue-600"
            >
              Add
            </button>
          </div>
          {(formData.cmd ?? []).length > 0 && (
            <div className="flex flex-wrap gap-2">
              {(formData.cmd ?? []).map((pkg, index) => (
                <span
                  key={index}
                  className="bg-gray-100 px-3 py-1 rounded-full text-sm flex items-center gap-2"
                >
                  {pkg}
                  <button
                    type="button"
                    onClick={() => removeCmd(index)}
                    className="text-red-500 hover:text-red-700 font-bold"
                  >
                    ×
                  </button>
                </span>
              ))}
            </div>
          )}
        </div>
        <div className="space-y-4">
          <div>
            <label
              htmlFor="branch"
              className="block text-sm font-medium mb-1"
            >
              Branch
            </label>
            <input
              id="branch"
              type="text"
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={formData.branch || ""}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  branch: e.target.value,
                }))
              }
            />
          </div>
          <div>
            <label
              htmlFor="beta_password"
              className="block text-sm font-medium mb-1"
            >
              Beta Password{" "}
              {formData.branch && formData.branch.trim() !== "" ? "*" : ""}
            </label>
            <input
              id="beta_password"
              type="text"
              required={
                !!(formData.branch && formData.branch.trim() !== "")
              }
              disabled={!formData.branch || formData.branch.trim() === ""}
              className={`w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                !formData.branch || formData.branch.trim() === ""
                  ? "bg-gray-100 cursor-not-allowed"
                  : ""
              }`}
              value={formData.beta_password || ""}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  beta_password: e.target.value,
                }))
              }
            />
            {formData.branch && formData.branch.trim() !== "" && (
              <p className="text-xs text-gray-600 mt-1">
                Beta password is required when branch is specified
              </p>
            )}
          </div>
        </div>
      </div>
      <div className="flex justify-start gap-3 mt-8 pt-4 border-t">
        <button
          type="submit"
          disabled={!isFormValid() || loading}
          className={`px-4 py-2 text-white rounded ${
            isFormValid() && !loading
              ? "bg-blue-400 hover:bg-blue-600 cursor-pointer"
              : "bg-gray-400 cursor-not-allowed"
          }`}
        >
          {loading ? "Submitting..." : "Submit"}
        </button>
        <button
          type="button"
          onClick={handleCancel}
          className="px-4 py-2 bg-red-500 text-white border border-red-500 rounded hover:bg-red-600"
        >
          Cancel
        </button>
      </div>
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded mt-4">
          <div className="flex items-center">
            <span className="font-medium">Error:</span>
            <span className="ml-2">{error}</span>
          </div>
        </div>
      )}
    </form>
  );
}
