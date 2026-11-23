/* ESP-IDF shell around the llgo build artifact (llgo_out.o).
 *
 * The llgo program compiles with -target esp32s31-idf into a relocatable
 * object. Its generated C entry, main(int32, char**), hands package
 * initialization and main.main to runtime.RunMain: the fiber scheduler runs
 * on this FreeRTOS task's stack, and every goroutine gets a fiber stack
 * allocated from the llgo GC heap. When main returns, control comes back
 * here and the main task deletes itself.
 */
#include <stdio.h>
#include <unistd.h>
#include <freertos/FreeRTOS.h>
#include <freertos/task.h>

extern int main(int argc, char **argv);

void app_main(void)
{
    printf("[c] app_main: starting llgo runtime\n");
    (void)main(0, NULL);
    printf("[c] app_main: llgo main returned, deleting main task\n");
    vTaskDelete(NULL);
}

/* The llgo runtime references the non-reentrant POSIX _write from a cold
 * diagnostic path. IDF's picolibc port only provides the reentrant
 * _write_r, so route the plain name through the VFS layer (fd 1/2 are the
 * console). */
ssize_t _write(int fd, const void *data, size_t size)
{
    return write(fd, data, size);
}

/* Go's fatal path (unrecovered panic, scheduler deadlock) calls exit(2).
 * IDF's _exit restarts the chip, which under esp-emu turns one panic into a
 * boot loop and destroys the diagnostics. The component links with
 * --wrap=_exit, so every _exit reference - newlib's exit() included -
 * lands here instead, and the fatal output stays the last thing in the
 * console log. */
void __wrap__exit(int status)
{
    printf("[c] _exit(%d): parking main task\n", status);
    for (;;) {
        vTaskDelay(portMAX_DELAY);
    }
}
