const RUSSIAN_TO_ENGLISH_MAP: Record<string, string> = {
  й: "q",
  ц: "w",
  у: "e",
  к: "r",
  е: "t",
  н: "y",
  г: "u",
  ш: "i",
  щ: "o",
  з: "p",
  х: "[",
  ъ: "]",
  ф: "a",
  ы: "s",
  в: "d",
  а: "f",
  п: "g",
  р: "h",
  о: "j",
  л: "k",
  д: "l",
  ж: ";",
  э: "'",
  я: "z",
  ч: "x",
  с: "c",
  м: "v",
  и: "b",
  т: "n",
  ь: "m",
  б: ",",
  ю: ".",
  ё: "`",
  Й: "Q",
  Ц: "W",
  У: "E",
  К: "R",
  Е: "T",
  Н: "Y",
  Г: "U",
  Ш: "I",
  Щ: "O",
  З: "P",
  Х: "[",
  Ъ: "]",
  Ф: "A",
  Ы: "S",
  В: "D",
  А: "F",
  П: "G",
  Р: "H",
  О: "J",
  Л: "K",
  Д: "L",
  Ж: ";",
  Э: "'",
  Я: "Z",
  Ч: "X",
  С: "C",
  М: "V",
  И: "B",
  Т: "N",
  Ь: "M",
  Б: ",",
  Ю: ".",
  Ё: "`",
  "№": "#",
  ";": ";",
  ":": ":",
  "?": "?",
  "!": "!",
  '"': '"',
  "'": "'",
  "«": '"',
  "»": '"',
  " ": " ",
  "\n": "\n",
  "\t": "\t",
};

const ENGLISH_TO_RUSSIAN_MAP: Record<string, string> = Object.fromEntries(
  Object.entries(RUSSIAN_TO_ENGLISH_MAP).map(([russian, english]) => [
    english,
    russian,
  ])
);

export function containsRussian(text: string): boolean {
  return /[а-яё]/i.test(text);
}

export function containsEnglish(text: string): boolean {
  return /[a-z]/i.test(text);
}

export function translateRussianToEnglish(text: string): string {
  return text
    .split("")
    .map((char) => RUSSIAN_TO_ENGLISH_MAP[char] || char)
    .join("");
}

export function translateEnglishToRussian(text: string): string {
  return text
    .split("")
    .map((char) => ENGLISH_TO_RUSSIAN_MAP[char] || char)
    .join("");
}

export function autoTranslateToEnglish(text: string): string {
  if (containsRussian(text)) {
    return translateRussianToEnglish(text);
  }
  return text;
}

export function generateSearchVariants(text: string): string[] {
  const variants = new Set<string>();

  variants.add(text);

  // Add normalized version (dashes to spaces)
  const normalized = text.replace(/-/g, " ");
  if (normalized !== text) {
    variants.add(normalized);
  }

  if (containsRussian(text)) {
    const englishTranslation = translateRussianToEnglish(text);
    variants.add(englishTranslation);
  }

  if (containsEnglish(text)) {
    const russianTranslation = translateEnglishToRussian(text);
    variants.add(russianTranslation);
  }

  return Array.from(variants);
}

// Optimized Levenshtein distance with early exit and minimal memory
export function levenshteinDistance(
  str1: string,
  str2: string,
  maxDistance?: number
): number {
  const m = str1.length;
  const n = str2.length;

  // Quick length check for early exit
  if (maxDistance !== undefined && Math.abs(m - n) > maxDistance) {
    return maxDistance + 1;
  }

  // Use two rows instead of full matrix for memory efficiency
  let prevRow = new Array(n + 1);
  let currRow = new Array(n + 1);

  // Initialize first row
  for (let j = 0; j <= n; j++) {
    prevRow[j] = j;
  }

  // Fill the matrix row by row
  for (let i = 1; i <= m; i++) {
    currRow[0] = i;
    let minInRow = currRow[0];

    for (let j = 1; j <= n; j++) {
      if (str1[i - 1] === str2[j - 1]) {
        currRow[j] = prevRow[j - 1];
      } else {
        currRow[j] =
          1 + Math.min(prevRow[j], currRow[j - 1], prevRow[j - 1]);
      }
      minInRow = Math.min(minInRow, currRow[j]);
    }

    // Early exit if minimum in row exceeds max distance
    if (maxDistance !== undefined && minInRow > maxDistance) {
      return maxDistance + 1;
    }

    // Swap rows
    [prevRow, currRow] = [currRow, prevRow];
  }

  return prevRow[n];
}

// Calculate allowed errors based on input length
function getAllowedErrors(inputLength: number, maxErrors: number = 3): number {
  if (inputLength <= 4) return 1;
  if (inputLength <= 7) return 2;
  return maxErrors;
}

// Check if two strings are similar with max allowed errors
export function isFuzzyMatch(
  input: string,
  target: string,
  maxErrors: number = 3
): boolean {
  const inputLower = input.toLowerCase();
  const targetLower = target.toLowerCase();

  // Direct match - fast path
  if (targetLower.includes(inputLower)) {
    return true;
  }

  // For very short inputs, require exact match only
  if (inputLower.length <= 2) {
    return false;
  }

  const allowedErrors = getAllowedErrors(inputLower.length, maxErrors);

  // Split target into words and check each word
  const targetWords = targetLower.split(/[\s\-_]+/);

  for (const word of targetWords) {
    // Skip very short words
    if (word.length < 2) continue;

    // Quick length difference check
    const lengthDiff = Math.abs(inputLower.length - word.length);
    if (lengthDiff <= allowedErrors) {
      // Check full word match
      const distance = levenshteinDistance(inputLower, word, allowedErrors);
      if (distance <= allowedErrors) {
        return true;
      }
    }

    // Check prefix match for longer words
    if (word.length > inputLower.length) {
      const wordPrefix = word.substring(0, inputLower.length);
      const prefixDistance = levenshteinDistance(
        inputLower,
        wordPrefix,
        allowedErrors
      );
      if (prefixDistance <= allowedErrors) {
        return true;
      }
    }
  }

  return false;
}
