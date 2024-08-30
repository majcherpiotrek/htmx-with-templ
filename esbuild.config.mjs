import { build } from 'esbuild';
import { sync } from 'glob';

const entryPoints = sync('./**/*.ts', {
  ignore: ["node_modules/**"]
});

console.log("Files to build:", entryPoints);

const isDev = process.env.NODE_ENV === 'development';

build({
  entryPoints: entryPoints,
  bundle: true,
  outdir: 'assets/js',
  format: 'esm',
  target: ['es6'],
  sourcemap: isDev,
  logLevel: 'info',
  minify: true,
  treeShaking: true,
  external: [],
}).catch(() => process.exit(1));

