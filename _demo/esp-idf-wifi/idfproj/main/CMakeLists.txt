# Main component: a thin C shell around the llgo build artifact.
#
# The Go program and the llgo runtime symbol providers are merged into
# main/llgo_out.o on the host by host/make-artifact.sh (see its header for
# why the providers must live inside the object rather than in this
# component). This component only supplies the C shell, the extra linker
# script and the link options.
#
# Glue sources: demo_glue.c is demo-local (checked in); the package glues
# are build inputs synced from the llgo repo's pkg/esp/*/glue templates by
# host/make-artifact.sh (integration way 1 in pkg/esp/README.md) and
# therefore gitignored.
#
# WHOLE_ARCHIVE is required: llgo_out.o is the LAST item on the link line,
# and the ESP-IDF link has no --start-group, so each component archive is
# scanned only against the undefined symbols known at its position. The
# glues are referenced solely from llgo_out.o; a plain archive would never
# pull them, exactly like the llgo_syms.S providers that therefore live
# inside llgo_out.o itself. Whole-archive keeps every glue object in the
# link unconditionally.
#
# REQUIRES lwip: pkg/esp/net's glue calls the lwip socket/netdb API
# directly; esp_timer because the same glue reads esp_timer_get_time for
# the Go-side deadline clock. mbedtls: pkg/esp/tls's glue drives the
# mbedtls_ssl_* / mbedtls_x509_crt_* API directly (the PSA crypto runtime
# comes with the same component and is initialized by IDF at startup).
# esp_hw_support is kept defensively: the glue compiles against mbedtls/
# psa headers only (the TRNG wiring behind psa_generate_random lives
# inside IDF's mbedtls port), but the explicit REQUIRES documents that the
# handshake's entropy path depends on that port's init chain.
idf_component_register(SRCS app_main.c demo_glue.c wifi_glue.c gpio_glue.c
                             i2c_glue.c i2s_glue.c net_glue.c tls_glue.c
                       INCLUDE_DIRS "."
                       REQUIRES esp_wifi esp_netif esp_event nvs_flash
                                esp_driver_gpio esp_driver_i2c esp_driver_i2s
                                lwip esp_timer mbedtls esp_hw_support
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
