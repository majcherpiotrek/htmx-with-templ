import { build } from 'esbuild';
import { sync } from 'glob';

// Get all TypeScript files in the pages directory
const entryPoints = sync('./ts/**/*.ts');

const isDev = process.env.NODE_ENV === 'development';

build({
  entryPoints: entryPoints,
  bundle: true,
  outdir: '../assets/js',
  format: 'esm',
  target: ['es6'],
  sourcemap: isDev,
  logLevel: 'info',
  minify: true,
  treeShaking: true,
  external: [],
}).catch(() => process.exit(1));

