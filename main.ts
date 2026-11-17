import { spawn } from 'node:child_process';
import { existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
const binary = join(here, 'forge');

if (!existsSync(binary)) {
  process.stderr.write('[entry] required static forge binary is missing\n');
  process.exit(1);
}
launch();

function launch(): void {
  const child = spawn(binary, [], { stdio: 'inherit', env: process.env });
  child.once('exit', (code) => process.exit(code ?? 1));
}
