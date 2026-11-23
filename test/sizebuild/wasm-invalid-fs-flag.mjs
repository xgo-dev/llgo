import fs from 'node:fs';

// Exercise an invalid flag during Go initialization, before main can run.
const host = Object.create(fs);
Object.defineProperty(host, 'constants', {
  value: {...fs.constants, O_WRONLY: undefined},
});
globalThis.fs = host;
