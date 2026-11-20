import assert from 'node:assert/strict';
import {pathToFileURL} from 'node:url';

const {default: init} = await import(pathToFileURL(process.argv[2]));
const module = await init({noInitialRun: true});
assert.equal(typeof module.cwrap, 'function');
const fs = module.FS;
assert.ok(fs.filesystems.MEMFS);
fs.mkdir('/llgo-host-api');
fs.mount(fs.filesystems.MEMFS, {}, '/llgo-host-api');
fs.writeFile('/llgo-host-api/data', 'host filesystem ABI');
assert.equal(fs.readFile('/llgo-host-api/data', {encoding: 'utf8'}), 'host filesystem ABI');
fs.unmount('/llgo-host-api');
