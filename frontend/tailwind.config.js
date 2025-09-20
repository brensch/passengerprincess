/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'princess-lavender': '#E6E0FF',
        'princess-rose': '#FFE0E6',
        'princess-mint': '#E0FFF0',
        'princess-peach': '#FFE6D9',
        'princess-lilac': '#F0E6FF',
        'princess-blush': '#FFE6F0',
        'princess-accent-lavender': '#D4C5FF',
        'princess-accent-rose': '#FFD4C5',
        'princess-accent-mint': '#C5FFD4',
        'princess-accent-peach': '#FFD1B3',
        'princess-text-primary': '#6B4D7C',
        'princess-text-secondary': '#8B5A7A',
        'princess-text-accent': '#9B6B8F',
        'princess-surface': '#FDFBFF',
        'princess-surface-soft': '#F8F4FF',
        'princess-border': '#E8DCF0',
        // Additional pink shades for table styling to match original
        'pink': {
          200: '#FBBDCB',
          500: '#EC4899',
          600: '#DB2777',
          700: '#BE185D',
          800: '#9D174D',
        },
        // Purple shades for table borders
        'purple': {
          200: '#E2BBE9',
          500: '#A855F7',
          600: '#9333EA',
        }
      },
      fontFamily: {
        'dancing': ['Dancing Script', 'cursive'],
      },
    },
  },
  plugins: [],
}
