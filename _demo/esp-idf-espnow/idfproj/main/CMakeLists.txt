# Main component: a thin C shell around the llgo build artifact of the
# ESP-NOW probe.
#
# The Go program and the llgo runtime symbol providers are merged into
# main/llgo_out.o on the host by host/make-artifact.sh (see its header for why
# the providers must live inside the object rather than in this component).
# This component only supplies the C shell, the extra linker script and the
# link options.
#
# Glue sources: the package glues are build inputs synced from the llgo repo's
# pkg/esp/*/glue templates by host/make-artifact.sh (integration way 1 in
# pkg/esp/README.md) and therefore gitignored. The probe needs two of them:
# wifi_glue.c (the radio's lifecycle: NVS/netif/event-loop/esp_wifi_init,
# mode and start - esp_now rides a started WiFi driver) and espnow_glue.c
# (the ESP-NOW surface itself).
#
# WHOLE_ARCHIVE is required: llgo_out.o is the LAST item on the link line, and
# the ESP-IDF link has no --start-group, so each component archive is scanned
# only against the undefined symbols known at its position. The glues are
# referenced solely from llgo_out.o; a plain archive would never pull them,
# exactly like the llgo_syms.S providers that therefore live inside
# llgo_out.o itself.
idf_component_register(SRCS app_main.c wifi_glue.c espnow_glue.c
                       INCLUDE_DIRS "."
                       REQUIRES esp_wifi esp_netif esp_event nvs_flash
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
