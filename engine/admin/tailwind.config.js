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
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' },
        },
        'pulse-dot': {
          '0%, 100%': { transform: 'scale(1)' },
          '50%': { transform: 'scale(1.3)' },
        },
        'pulse-glow': {
          '0%, 100%': { boxShadow: '0 0 0 0 rgba(99, 102, 241, 0)' },
          '50%': { boxShadow: '0 0 0 4px rgba(99, 102, 241, 0.3)' },
        },
        'slide-down': {
          '0%': { opacity: '0', maxHeight: '0', transform: 'translateY(-4px)' },
          '100%': { opacity: '1', maxHeight: '500px', transform: 'translateY(0)' },
        },
        'bounce-dots': {
          '0%, 80%, 100%': { transform: 'scale(0)' },
          '40%': { transform: 'scale(1)' },
        },
      },
      animation: {
        'slide-in-right': 'slide-in-right 0.2s ease-out',
        'modal-in': 'modal-in 0.15s ease-out',
        'fade-in': 'fade-in 0.3s ease-out',
        'pulse-dot': 'pulse-dot 2s ease-in-out infinite',
        'pulse-glow': 'pulse-glow 1.5s ease-in-out infinite',
        'slide-down': 'slide-down 0.25s ease-out',
        'bounce-dot-1': 'bounce-dots 1.4s infinite ease-in-out both',
        'bounce-dot-2': 'bounce-dots 1.4s infinite ease-in-out both 0.16s',
        'bounce-dot-3': 'bounce-dots 1.4s infinite ease-in-out both 0.32s',
      },
    },
  },
  plugins: [],
};
