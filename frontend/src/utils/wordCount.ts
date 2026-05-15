export function countStoryUnits(text: string): number {
  let count = 0;
  let inLatin = false;
  for (const char of text.trim()) {
    if (/[\p{Script=Han}\p{Script=Hiragana}\p{Script=Katakana}\p{Script=Hangul}]/u.test(char)) {
      inLatin = false;
      count += 1;
    } else if (/[\p{L}\p{N}]/u.test(char)) {
      if (!inLatin) count += 1;
      inLatin = true;
    } else {
      inLatin = false;
    }
  }
  return count;
}
