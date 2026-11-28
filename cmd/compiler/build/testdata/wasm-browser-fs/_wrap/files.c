#include <stdio.h>
#include <string.h>
#include <unistd.h>

int llgo_browser_fs_read_write(const char *path) {
    FILE *file = fopen(path, "r+");
    if (!file) return 1;
    char buffer[8] = {0};
    int result = fread(buffer, 1, 7, file) != 7 || strcmp(buffer, "Go file");
    rewind(file);
    result |= fwrite("C file", 1, 6, file) != 6;
    result |= fflush(file);
    result |= ftruncate(fileno(file), 6);
    result |= fclose(file);
    return result;
}
