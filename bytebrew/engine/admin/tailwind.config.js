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
        status: {
          active: '#4CAF50',
          attention: '#D7513E',
          idle: '#87867F',
        },
      },
      fontFamily: {
        mono: ['"IBM Plex Mono"', 'monospace'],
      },
      borderRadius: {
        card: '12px',
        btn: '10px',
      },
    },
  },
  plugins: [],
};
