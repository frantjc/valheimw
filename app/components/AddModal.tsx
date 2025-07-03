import React from "react";
import { SteamappDetail } from "../client";

type AddModalProps = {
  open: boolean;
  onClose: () => void;
};

export function AddModal({ open, onClose }: AddModalProps) {
  const [formData, setFormData] = React.useState<SteamappDetail & {app_id: number, branch?: string}>({
    app_id: 0,
    base_image: "",
    apt_packages: [],
    beta_password: "",
    launch_type: "",
    platform_type: "",
    execs: [],
    entrypoint: [],
    cmd: [],
    ports: [],
    volumes: [],
    resources: { cpu: "", memory: "" },
  });

  const [aptPackageInput, setAptPackageInput] = React.useState("");
  const addAptPackage = () => {
    if (
      aptPackageInput.trim() &&
      !(formData.apt_packages ?? []).includes(aptPackageInput.trim())
    ) {
      setFormData(prev => ({
        ...prev,
        apt_packages: [...(prev.apt_packages ?? []), aptPackageInput.trim()]
      }));
      setAptPackageInput("");
    }
  };

  const removeAptPackage = (index: number) => {
    setFormData(prev => ({
      ...prev,
      apt_packages: (prev.apt_packages ?? []).filter((_, i) => i !== index)
    }));
  };

  const [execInput, setExecInput] = React.useState("");
  const addExec = () => {
    if (
      execInput.trim() &&
      !(formData.execs ?? []).includes(execInput.trim())
    ) {
      setFormData(prev => ({
        ...prev,
        execs: [...(prev.execs ?? []), execInput.trim()]
      }));
      setExecInput("");
    }
  };

  const removeExec = (index: number) => {
    setFormData(prev => ({
      ...prev,
      execs: (prev.execs ?? []).filter((_, i) => i !== index)
    }));
  };

  const [entrypointInput, setEntrypointInput] = React.useState("");
  const addEntrypoint = () => {
    if (
      entrypointInput.trim() &&
      !(formData.entrypoint ?? []).includes(entrypointInput.trim())
    ) {
      setFormData(prev => ({
        ...prev,
        entrypoint: [...(prev.entrypoint ?? []), entrypointInput.trim()]
      }));
      setEntrypointInput("");
    }
  };

  const removeEntrypoint = (index: number) => {
    setFormData(prev => ({
      ...prev,
      entrypoint: (prev.entrypoint ?? []).filter((_, i) => i !== index)
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
      ports: [],
      volumes: [],
      resources: { cpu: "", memory: "" },
    });
    
    setAptPackageInput("");
    setExecInput("");
    setEntrypointInput("");
    
    onClose();
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: Submit to API
    console.log("Submitting:", formData);
    onClose();
  };

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50 px-4"
      onClick={onClose}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
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
          <span className="font-bold text-lg text-white">Add Steam App</span>
          <div className="flex gap-2">
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-white font-bold px-2 py-1 rounded"
            >
              Close
            </button>
          </div>
        </div>
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
                    placeholder="e.g. curl, wget, git"
                    className="flex-1 px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                    value={aptPackageInput}
                    onChange={(e) => setAptPackageInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
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
              <div>
                <label 
                  htmlFor="execs"
                  className="block text-sm font-medium mb-1"
                >
                  Executables
                </label>
                <div className="flex gap-2 mb-2">
                  <input
                    type="text"
                    className="flex-1 px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                    value={execInput}
                    onChange={(e) => setExecInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
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
              <div>
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
                      if (e.key === 'Enter') {
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
            </div>
          </div>
          <div className="flex justify-start gap-3 mt-8 pt-4 border-t">
            <button
              type="submit"
              className="px-4 py-2 bg-blue-400 text-white rounded hover:bg-blue-600"
            >
              Submit
            </button>
            <button
              type="button"
              onClick={handleCancel}
              className="px-4 py-2 bg-red-500 text-white border border-red-500 rounded hover:bg-red-600"
            >
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
