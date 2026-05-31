import tailwindcssAnimate from 'tailwindcss-animate'

/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // shadcn 语义 token（全部走 CSS 变量，支持 /alpha 修饰；亮色接口预留）
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))',
        },
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        border: 'hsl(var(--border))',
        'border-soft': 'hsl(var(--border-soft))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        faint: 'hsl(var(--faint))',
        // 数据语义色（全站一致）
        'data-requests': 'hsl(var(--data-requests))',
        'data-tokens': 'hsl(var(--data-tokens))',
        'data-cost': 'hsl(var(--data-cost))',
        'data-failed': 'hsl(var(--data-failed))',
        'data-success': 'hsl(var(--data-success))',
        // 模型分布色阶
        'model-1': 'hsl(var(--m1))',
        'model-2': 'hsl(var(--m2))',
        'model-3': 'hsl(var(--m3))',
        'model-4': 'hsl(var(--m4))',
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      fontFamily: {
        sans: ['"Fira Sans"', 'system-ui', 'sans-serif'],
        mono: ['"Fira Code"', 'ui-monospace', 'monospace'],
      },
      keyframes: {
        'accordion-down': {
          from: { height: '0' },
          to: { height: 'var(--radix-accordion-content-height)' },
        },
        'accordion-up': {
          from: { height: 'var(--radix-accordion-content-height)' },
          to: { height: '0' },
        },
        'caret-blink': {
          '0%,70%,100%': { opacity: '1' },
          '20%,50%': { opacity: '0' },
        },
        pulse: {
          '0%': { boxShadow: '0 0 0 0 hsl(var(--data-success) / 0.45)' },
          '70%': { boxShadow: '0 0 0 7px hsl(var(--data-success) / 0)' },
          '100%': { boxShadow: '0 0 0 0 hsl(var(--data-success) / 0)' },
        },
      },
      animation: {
        'accordion-down': 'accordion-down 0.2s ease-out',
        'accordion-up': 'accordion-up 0.2s ease-out',
        'caret-blink': 'caret-blink 1.25s ease-out infinite',
        'pulse-ring': 'pulse 2.4s ease-out infinite',
      },
    },
  },
  plugins: [tailwindcssAnimate],
}
