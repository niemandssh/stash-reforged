const RUSSIAN_TO_ENGLISH_MAP: Record<string, string> = {
  'й': 'q', 'ц': 'w', 'у': 'e', 'к': 'r', 'е': 't', 'н': 'y', 'г': 'u', 'ш': 'i', 'щ': 'o', 'з': 'p',
  'х': '[', 'ъ': ']', 'ф': 'a', 'ы': 's', 'в': 'd', 'а': 'f', 'п': 'g', 'р': 'h', 'о': 'j', 'л': 'k',
  'д': 'l', 'ж': ';', 'э': "'", 'я': 'z', 'ч': 'x', 'с': 'c', 'м': 'v', 'и': 'b', 'т': 'n', 'ь': 'm',
  'б': ',', 'ю': '.', 'ё': '`',
  'Й': 'Q', 'Ц': 'W', 'У': 'E', 'К': 'R', 'Е': 'T', 'Н': 'Y', 'Г': 'U', 'Ш': 'I', 'Щ': 'O', 'З': 'P',
  'Х': '[', 'Ъ': ']', 'Ф': 'A', 'Ы': 'S', 'В': 'D', 'А': 'F', 'П': 'G', 'Р': 'H', 'О': 'J', 'Л': 'K',
  'Д': 'L', 'Ж': ';', 'Э': "'", 'Я': 'Z', 'Ч': 'X', 'С': 'C', 'М': 'V', 'И': 'B', 'Т': 'N', 'Ь': 'M',
  'Б': ',', 'Ю': '.', 'Ё': '`',
  '№': '#', ';': ';', ':': ':', '?': '?', '!': '!', '"': '"', "'": "'", '«': '"', '»': '"',
  ' ': ' ', '\n': '\n', '\t': '\t'
};

const ENGLISH_TO_RUSSIAN_MAP: Record<string, string> = Object.fromEntries(
  Object.entries(RUSSIAN_TO_ENGLISH_MAP).map(([russian, english]) => [english, russian])
);

export function containsRussian(text: string): boolean {
  return /[а-яё]/i.test(text);
}

export function containsEnglish(text: string): boolean {
  return /[a-z]/i.test(text);
}

export function translateRussianToEnglish(text: string): string {
  return text
    .split('')
    .map(char => RUSSIAN_TO_ENGLISH_MAP[char] || char)
    .join('');
}

export function translateEnglishToRussian(text: string): string {
  return text
    .split('')
    .map(char => ENGLISH_TO_RUSSIAN_MAP[char] || char)
    .join('');
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