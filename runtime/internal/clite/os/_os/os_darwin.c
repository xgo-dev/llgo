// INODE64 compatibility wrappers for darwin/amd64 syscall trampolines.
#include <stdint.h>
#include <dirent.h>

uintptr_t llgo_fdopendir(uintptr_t fd) {
	DIR *ret = fdopendir((int)fd);
	return (uintptr_t)ret;
}

uintptr_t llgo_closedir(uintptr_t dir) {
	int ret = closedir((DIR *)dir);
	return (uintptr_t)(intptr_t)ret;
}

uintptr_t llgo_readdir_r(uintptr_t dir, uintptr_t entry, uintptr_t result) {
	int ret = readdir_r((DIR *)dir, (struct dirent *)entry, (struct dirent **)result);
	return (uintptr_t)(intptr_t)ret;
}
