import type { Book } from "./ui";
const titles = [
  "Orbits of Silence",
  "Map of Winds",
  "After the Rain",
  "Garden of Observations",
  "Northern Light",
  "Fields of Memory",
  "A Little Geology",
  "Letters to the Sea",
  "Winter Room",
  "Clocks Without Hands",
  "Inside Noon",
  "Thin Line",
];
const authors = [
  "Mira Holm",
  "Anton Ray",
  "Leya Snow",
  "I. Polevoy",
  "Nika Orlova",
  "Mark Lesnoy",
  "Eva Rin",
  "Tim Veresk",
  "Asya Mir",
  "Lev Nord",
  "Olga Ro",
  "Yan Svet",
];
export const demoShelves = ["Эссе и заметки", "Перед сном", "В дорогу"];
export const demoBooks: Book[] = titles.map((title, i) => ({
  id: `demo-${i}`,
  title,
  author: authors[i]!,
  format: ["EPUB", "PDF", "FB2", "EPUB", "EPUB", "PDF"][i % 6]!,
  size: (i + 2) * 1024 * 1024,
  progress: i === 0 ? 42 : i === 3 ? 18 : i === 8 ? 65 : 0,
  shelves: [demoShelves[i % 3]!],
  bookmarks: i === 0 ? ["Глава III"] : [],
  originalName: title + "." + ["epub", "pdf", "fb2"][i % 3],
  updatedAt: new Date(2026, 9, 13 - i).toISOString(),
  tone: i,
}));
export const demoRecovery = "7K4M 9R2T W6NP 3X8A H5DQ V2LC 8F7S J4ZE";
export const chapters = [
  "Пролог",
  "1. Берег",
  "2. Далёкий сигнал",
  "3. Орбиты тишины",
  "4. Комната света",
  "5. Возвращение",
];
export const readingText = [
  "В полдень свет ложился на полки ровными полосами. Казалось, каждая книга хранила свою маленькую частицу времени — не прошлого и не будущего, а точного мгновения, в которое можно вернуться.",
  "Лея провела пальцами по корешкам и выбрала книгу без названия. Бумага пахла сухой травой. На первой странице кто-то оставил короткую фразу: «Всё важное движется медленно».",
  "Она села у окна. Город за стеклом продолжал привычно шуметь, но здесь слова появлялись одно за другим, спокойные и ясные. Чтение не уносило её прочь — напротив, возвращало вещам вес и расстояние.",
  "К вечеру на полях появилась тонкая карандашная линия. Не отметка и не ответ, а знак присутствия: я была здесь, я читала это, теперь эта мысль принадлежит и мне.",
];
