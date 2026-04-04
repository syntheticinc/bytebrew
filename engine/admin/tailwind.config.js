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
          'dark-surface': '#181818',
          'dark-alt': '#1F1F1F',
          light: '#F7F8F1',
          shade1: '#DFD8D0',
          shade2: '#CBC9BC',
          shade3: '#87867F',
        },
        status: {
          active: '#4CAF50',
          attention: '#D7513E',
          idle: '#87867F',
        },
      },
      fontFamily: {
        mono: ['"IBM Plex Mono"', 'monospace'],
      },
      zIndex: {
        60: '60',
      },
      borderRadius: {
        card: '2px',
        btn: '2px',
      },
      keyframes: {
        'slide-in-right': {
          from: { transform: 'translateX(100%)' },
          to: { transform: 'translateX(0)' },
        },
        'modal-in': {
          from: { opacity: '0', transform: 'scale(0.95)' },
          to: { opacity: '1', transform: 'scale(1)' },
        },
        'fade-in': {
          from: { opacity: '0' },
          to: { opacity: '1' },
        },
        'pulse-dot': {
          '0%, 100%': { transform: 'scale(1)' },
          '50%': { transform: 'scale(1.3)' },
        },
      },
      animation: {
        'slide-in-right': 'slide-in-right 0.2s ease-out',
        'modal-in': 'modal-in 0.15s ease-out',
        'fade-in': 'fade-in 0.3s ease-out',
        'pulse-dot': 'pulse-dot 2s ease-in-out infinite',
      },
    },
  },
  plugins: [],
};
