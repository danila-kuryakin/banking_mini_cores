import { useCallback, useState } from 'react';
import { writeStorage } from './storage';

export type Theme = 'dark' | 'light';

const KEY = 'bmc.theme';

// Начальную тему ставит скрипт в index.html до отрисовки, здесь только переключение.
function current(): Theme {
  return document.documentElement.dataset.theme === 'light' ? 'light' : 'dark';
}

export function useTheme() {
  const [theme, setTheme] = useState<Theme>(current);
  const toggle = useCallback(() => {
    const next: Theme = current() === 'dark' ? 'light' : 'dark';
    document.documentElement.dataset.theme = next;
    writeStorage(KEY, next);
    setTheme(next);
  }, []);
  return { theme, toggle };
}
