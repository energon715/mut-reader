export type DirectoryPaths = {
  library: string;
  backup: string;
  restore: string;
  imports: string;
};

// A build on this computer has useful defaults before the optional native
// GetDefaultPaths method is implemented. Native defaults take precedence.
const home = import.meta.env.VITE_MUT_HOME.replace(/\/$/, "");
export const initialDefaults: DirectoryPaths = {
  library: `${home}/.local/share/mut/vault`,
  backup: `${home}/Documents/mut-backups`,
  restore: `${home}/Documents/mut-restored`,
  imports: `${home}/Downloads`,
};

export function loadSavedPaths(
  defaults: DirectoryPaths,
  storageKey = "mut.paths",
): DirectoryPaths {
  const result = { ...defaults };
  try {
    const saved: unknown = JSON.parse(localStorage.getItem(storageKey) || "{}");
    if (saved && typeof saved === "object") {
      for (const key of Object.keys(result) as (keyof DirectoryPaths)[]) {
        const value = (saved as Record<string, unknown>)[key];
        if (typeof value === "string" && value.startsWith("/"))
          result[key] = value;
      }
    }
  } catch {}
  return result;
}

type DirectoryBridge = {
  GetDefaultPaths?: () => Promise<DirectoryPaths>;
  ChooseDirectory?: (title: string, currentPath: string) => Promise<string>;
  ChooseFiles?: (currentDirectory: string) => Promise<string[]>;
};

function bridge(): DirectoryBridge {
  return (
    (window as unknown as { go?: { app?: { App?: DirectoryBridge } } }).go?.app
      ?.App || {}
  );
}

// Optional until the application author implements these Go bridge methods.
export function hasDirectoryPicker() {
  return typeof bridge().ChooseDirectory === "function";
}
export function hasFilePicker() {
  return typeof bridge().ChooseFiles === "function";
}
export async function getNativeDefaults(): Promise<DirectoryPaths | null> {
  const method = bridge().GetDefaultPaths;
  return method ? method() : null;
}
export async function chooseDirectory(
  title: string,
  currentPath: string,
): Promise<string> {
  const method = bridge().ChooseDirectory;
  if (!method) throw new Error("Directory picker is not available");
  return method(title, currentPath);
}
export async function chooseFiles(currentDirectory: string): Promise<string[]> {
  const method = bridge().ChooseFiles;
  if (!method) throw new Error("File picker is not available");
  return (await method(currentDirectory)) || [];
}
