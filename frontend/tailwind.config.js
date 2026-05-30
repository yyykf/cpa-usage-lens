/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        background: '#0A0E1A',
        foreground: '#E6E9F0',
        card: '#111627',
        popover: '#111627',
        muted: '#1A2036',
        'muted-foreground': '#8B93A7',
        border: '#232A40',
        input: '#232A40',
        primary: '#3B82F6',
        'primary-foreground': '#FFFFFF',
        accent: '#1E2540',
        ring: '#3B82F6',
        destructive: '#EF4444',
        // 数据语义色（全站一致）
        'data-requests': '#3B82F6',
        'data-tokens': '#22D3EE',
        'data-cost': '#F97316',
        'data-failed': '#EF4444',
        'data-success': '#10B981',
      },
      fontFamily: {
        sans: ['"Fira Sans"', 'system-ui', 'sans-serif'],
        mono: ['"Fira Code"', 'ui-monospace', 'monospace'],
      },
      borderRadius: {
        '2xl': '0.875rem',
      },
    },
  },
  plugins: [],
}
