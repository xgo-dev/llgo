# Main component: a thin C shell around the llgo build artifact of the NVS
# acceptance probe.
#
# The Go program and the llgo runtime symbol providers are merged into
# main/llgo_out.o on the host by host/make-artifact.sh (see its header for
# why the providers must live inside the object rather than in this
# component). This component supplies the C shell, the NVS glue, the extra
# linker script and the link options.
#
# Glue sources: demo_glue.c is demo-local (checked in, the C-side heap
# ledger); nvs_glue.c is a build-time copy of the pkg/esp/nvs glue template,
# synced from the llgo repo's pkg/esp/nvs/glue/ by host/make-artifact.sh
# (integration way 1 in pkg/esp/README.md) and therefore gitignored. It
# drives the nvs.h / nvs_flash.h API directly - no managed component.
#
# WHOLE_ARCHIVE is required: llgo_out.o is the LAST item on the link line,
# and the ESP-IDF link has no --start-group, so each component archive is
# scanned only against the undefined symbols known at its position. The glue
# is referenced solely from llgo_out.o; a plain archive would never pull it,
# exactly like the llgo_syms.S providers that therefore live inside
# llgo_out.o itself. Whole-archive keeps every main-component object in the
# link unconditionally.
#
# REQUIRES nvs_flash: the glue calls nvs_open / nvs_get_* / nvs_set_* /
# nvs_commit / nvs_erase_key / nvs_entry_find against that component. The
# esp32s31 default components cover the rest (esp_runtime provides
# esp_restart for the probe's restart cycle, freertos for app_main's
# parking loop, esp_hw_support for heap_caps).
idf_component_register(SRCS app_main.c demo_glue.c nvs_glue.c
                       INCLUDE_DIRS "."
                       REQUIRES nvs_flash
                       WHOLE_ARCHIVE)

# The llgo object must be a PUBLIC (interface) link item so CMake places it
# as early as possible on the final link line.
target_link_libraries(${COMPONENT_LIB} PUBLIC
                      "${CMAKE_CURRENT_LIST_DIR}/llgo_out.o")

# Route Go fatal exits (exit() -> _exit) to the component's parking wrapper
# instead of IDF's chip-restarting _exit (see app_main.c). The extra -T
# script only assigns _globals_start/_globals_end; the assignments evaluate
# at layout time, so its position in the -T list does not matter.
target_link_options(${COMPONENT_LIB} PUBLIC
                    "-Wl,--wrap=_exit"
                    "-T" "${CMAKE_CURRENT_LIST_DIR}/llgo_dram.ld")
