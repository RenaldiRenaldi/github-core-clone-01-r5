import { spawn } from 'node:child_process';
import { createServer } from 'node:net';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
const probe = createServer();
probe.listen(0, '0.0.0.0', () => {
  const address = probe.address();
  if (!address || typeof address === 'string') process.exit(1);
  const child = spawn(join(here, 'forge'), [], { stdio: 'inherit', env: process.env });
  child.once('exit', (code) => process.exit(code ?? 1));
  probe.close();
});
