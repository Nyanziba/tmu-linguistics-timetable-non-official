// @ts-check
import { defineConfig } from 'astro/config';

import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
  site: 'https://nyanziba.github.io',
  base: '/tmu-linguistics-timetable-non-official',
  vite: {
    plugins: [tailwindcss()]
  }
});