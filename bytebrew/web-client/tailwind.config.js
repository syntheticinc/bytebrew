/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          accent: '#D7513E',
          'accent-hover': '#C04635',
          dark: '#111111',
          'dark-alt': '#1F1F1F',
          light: '#F7F8F1',
          shade1: '#DFD8D0',
          shade2: '#CBC9BC',
          shade3: '#87867F',
        },
      },
      fontFamily: {
        mono: ['"IBM Plex Mono"', 'monospace'],
      },
      borderRadius: {
        card: '12px',
        btn: '10px',
      },
      keyframes: {
        'bounce-dots': {
          '0%, 80%, 100%': { transform: 'scale(0)' },
          '40%': { transform: 'scale(1)' },
        },
        'fade-in': {
          from: { opacity: '0', transform: 'translateY(4px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        'slide-in-right': {
          from: { transform: 'translateX(100%)' },
          to: { transform: 'translateX(0)' },
        },
      },
      animation: {
        'bounce-dot-1': 'bounce-dots 1.4s infinite ease-in-out both',
        'bounce-dot-2': 'bounce-dots 1.4s infinite ease-in-out both 0.16s',
        'bounce-dot-3': 'bounce-dots 1.4s infinite ease-in-out both 0.32s',
        'fade-in': 'fade-in 0.2s ease-out',
        'slide-in-right': 'slide-in-right 0.25s ease-out',
      },
    },
  },
  plugins: [],
};
