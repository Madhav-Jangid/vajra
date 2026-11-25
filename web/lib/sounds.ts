const cache: Record<string, HTMLAudioElement> = {};

export function playSound(name: string) {
  try {
    const audio = cache[name] ?? (cache[name] = new Audio(`/sounds/${name}.mp3`));

    audio.currentTime = 0;

    void audio.play().catch(() => {});
  } catch {
    return;
  }
}
