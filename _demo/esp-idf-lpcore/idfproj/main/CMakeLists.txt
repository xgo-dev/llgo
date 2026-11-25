# Main component: a thin C shell around the llgo build artifact, plus the LP
# core program the IDF ULP build system compiles and embeds.
#
# The Go program and the llgo runtime symbol providers are merged into
# main/llgo_out.o on the host by host/make-artifact.sh (see its header for why
# the providers must live inside the object rather than in this component).
# This component supplies the C shell, the demo glue, the LP core build, the
# extra linker script and the link options.
#
# Glue sources: demo_glue.c is demo-local (checked in); lpcore_glue.c is a
# build-time copy of the pkg/esp/lpcore glue template, synced from the llgo
# repo's pkg/esp/lpcore/glue/ by host/make-artifact.sh (integration way 1 in
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
# REQUIRES ulp: the lpcore glue drives the ULP driver API (ulp_lp_core_*,
# lp_core_mailbox_*) and the ULP build system compiles lp_core/main.c into the
# embedded lp_core_main.bin. esp_timer: app_main's park loop and the probe's
# tick pacing sit on FreeRTOS time. The lp_core subdirectory is NOT a
# component: ulp_embed_binary compiles it as a separate subproject with the
# LP toolchain.
idf_component_register(SRCS app_main.c demo_glue.c lpcore_glue.c
                       INCLUDE_DIRS "."
                       REQUIRES ulp esp_timer
                       WHOLE_ARCHIVE)

# ---- LP support additions to the component CMakeLists.txt (the pattern of
# examples/system/ulp/lp_core/lp_mailbox/main/CMakeLists.txt). ----
#
# 1. The LP Core app name must be unique (it names the embedded binary and
#    the _binary_ symbols demo_glue.c declares).
set(ulp_app_name lp_core_main)
#
# 2. Specify the LP core's C files. They live in a separate directory
#    (lp_core/) which is NOT part of COMPONENT_SRCS: they compile with the
#    LP RISC-V toolchain, not the HP one.
set(ulp_lp_core_sources "lp_core/main.c")
#
# 3. List the component sources that must rebuild when the LP binary changes.
#    demo_glue.c references the embedded binary's symbols; it does not include
#    the generated lp_core_main.h (no shared LP variables cross here), so the
#    dependency is only about rebuild tracking.
set(ulp_exp_dep_srcs "demo_glue.c")
#
# 4. Build the LP binary and embed it in the HP firmware.
ulp_embed_binary(${ulp_app_name} "${ulp_lp_core_sources}" "${ulp_exp_dep_srcs}")

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
