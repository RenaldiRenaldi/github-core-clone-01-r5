import { spawn } from 'node:child_process';
import { existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
const binary = join(here, 'forge');
const go = process.env.GO ?? `${process.env.HOME}/bin/go/bin/go`;

if (!existsSync(binary)) {
  const build = spawn(go, ['build', '-trimpath', '-ldflags=-s -w', '-o', binary, '.'], {
    cwd: here,
    stdio: 'inherit',
    env: { ...process.env, CGO_ENABLED: '0' },
  });
  build.once('exit', (code) => {
    if (code !== 0) process.exit(code ?? 1);
    launch();
  });
} else {
  launch();
}

function launch(): void {
  const child = spawn(binary, [], { stdio: 'inherit', env: process.env });
  child.once('exit', (code) => process.exit(code ?? 1));
}
