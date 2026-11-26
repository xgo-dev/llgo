# Main component: a thin C shell around the llgo build artifact, plus the
# ADC glue this demo exercises.
#
# The Go program and the llgo runtime symbol providers are merged into
# main/llgo_out.o on the host by host/make-artifact.sh (see its header for why
# the providers must live inside the object rather than in this component).
# This component supplies the C shell, the demo glues and the link options.
#
# Glue sources: demo_glue.c is demo-local (checked in); adc_glue.c is a
# build-time copy of the pkg/esp glue template, synced from the llgo repo's
# pkg/esp/adc/glue/ by host/make-artifact.sh (integration way 1 in
# pkg/esp/README.md) and therefore gitignored.
#
# The llgo runtime ships a weak llgoLockCore default (honest refusal), so a
# demo that never pins needs no shim; a demo wanting real pins defines a
# strong override (see _demo/esp-idf-lockcore).
#
# WHOLE_ARCHIVE is required: llgo_out.o is the LAST item on the link line, and
# the ESP-IDF link has no --start-group, so each component archive is scanned
# only against the undefined symbols known at its position. The glue is
# referenced solely from llgo_out.o; a plain archive would never pull it,
# exactly like the llgo_syms.S providers that therefore live inside llgo_out.o
# itself. Whole-archive keeps every main-component object in the link
# unconditionally.
#
# REQUIRES esp_adc: the glue drives the oneshot ADC API (adc_oneshot.h) and
# the line-fitting calibration scheme (adc_cali_scheme.h).
# esp_hal_ana_conv: the glue's eFuse version query includes hal/adc_efuse_hal.h
# from that component (esp_adc pulls it transitively; the direct include is
# declared explicitly). esp_timer: app_main's park loop and the probe's tick
# pacing sit on FreeRTOS time.
idf_component_register(SRCS app_main.c demo_glue.c adc_glue.c
                       INCLUDE_DIRS "."
                       REQUIRES esp_adc esp_hal_ana_conv esp_timer
                       WHOLE_ARCHIVE)

# The llgo object must be a PUBLIC (interface) link item so CMake places it as
# early as possible on the final link line.
target_link_libraries(${COMPONENT_LIB} PUBLIC
                      "${CMAKE_CURRENT_LIST_DIR}/llgo_out.o")

# Route Go fatal exits (exit() -> _exit) to the component's parking wrapper
# instead of IDF's chip-restarting _exit (see app_main.c). The extra -T
# script only assigns _globals_start/_globals_end; the assignments evaluate
# at layout time, so its position in the -T list does not matter.
target_link_options(${COMPONENT_LIB} PUBLIC
                    "-Wl,--wrap=_exit"
                    "-T" "${CMAKE_CURRENT_LIST_DIR}/llgo_dram.ld")
