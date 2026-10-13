import type { catalog } from "../wailsjs/go/models";
export type Book = {
  id: string;
  title: string;
  author: string;
  format: string;
  size: number;
  progress: number;
  shelves: string[];
  bookmarks: string[];
  originalName: string;
  updatedAt: string;
  tone: number;
  locator?: string;
  coverId?: string;
};
export type Screen =
  "library" | "login" | "create" | "recovery" | "reader" | "backup" | "restore";
export type Overlay =
  "settings" | "import" | "book" | "shelf" | "manageshelf" | "security" | null;
export function fromCatalog(book: catalog.Book, index: number): Book {
  const id: unknown = book.id;
  const readingDate = String(book.position?.updateat || "");
  return {
    id: Array.isArray(id)
      ? id.map((n: number) => n.toString(16).padStart(2, "0")).join("")
      : String(id),
    title: book.title || book.originalname || "Без названия",
    author: book.author || "Автор не указан",
    format: book.format?.toUpperCase() || "FILE",
    size: book.filesize || 0,
    progress: Math.max(0, Math.min(100, book.position?.progress || 0)),
    shelves: book.shelves || [],
    bookmarks: book.bookmarks || [],
    originalName: book.originalname,
    updatedAt:
      readingDate && !readingDate.startsWith("0001-")
        ? readingDate
        : String(book.updatedat || book.createdat || ""),
    tone: index % 12,
    locator: book.position?.locator || "",
    coverId: book.coverobjectid || "",
  };
}
export function formatSize(bytes: number) {
  return (
    new Intl.NumberFormat("ru", { maximumFractionDigits: 1 }).format(
      bytes / (1024 * 1024),
    ) + " МБ"
  );
}
export const canRead = (format: string) =>
  ["PDF", "EPUB", "FB2"].includes(format.toUpperCase());
export const tones = [
  "#393f41",
  "#a3aaa8",
  "#60676a",
  "#8b9293",
  "#878f90",
  "#6e7678",
  "#939a98",
  "#525b5d",
  "#747d7c",
  "#7b8583",
  "#9da5a3",
  "#454e4f",
];
