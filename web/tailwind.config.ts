import type { Config } from 'tailwindcss';

// Цвета берутся из CSS-переменных темы, поэтому тёмная и светлая переключаются атрибутом data-theme.
const v = (name: string) => `rgb(var(--${name}) / <alpha-value>)`;

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        bg: v('bg'),
        elevated: v('bg-elevated'),
        input: v('bg-input'),
        border: v('border'),
        text: v('text'),
        muted: v('text-muted'),
        accent: v('accent'),
        'accent-text': v('accent-text'),
        up: v('up'),
        down: v('down'),
        warn: v('warn'),
        info: v('info'),
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'monospace'],
      },
      fontSize: { base: ['14px', '20px'] },
    },
  },
  plugins: [],
} satisfies Config;
