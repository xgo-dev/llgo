# Main component: a thin C shell around the llgo build artifact.
#
# The Go program and the llgo runtime symbol providers are merged into
# main/llgo_out.o on the host by host/make-artifact.sh (see its header for why
# the providers must live inside the object rather than in this component).
# This component supplies the C shell and the core-affinity shim the Go
# runtime's LockCore links against (runtime/internal/runtime/lockcore_idf.go
# documents why the shim lives here: the IDF kernel offers no runtime setter,
# so the pin goes through this component's llgoLockCore).
#
# WHOLE_ARCHIVE is required: llgo_out.o is the LAST item on the link line, and
# the ESP-IDF link has no --start-group, so each component archive is scanned
# only against the undefined symbols known at its position. The glue is
# referenced solely from llgo_out.o; a plain archive would never pull it,
# exactly like the llgo_syms.S providers that therefore live inside llgo_out.o
# itself. Whole-archive keeps every main-component object in the link
# unconditionally.
idf_component_register(SRCS app_main.c lockcore_glue.c
                       INCLUDE_DIRS "."
                       WHOLE_ARCHIVE)

# The llgo object must be a PUBLIC (interface) link item so CMake places it as
# early as possible on the final link line.
target_link_libraries(${COMPONENT_LIB} PUBLIC
                      "${CMAKE_CURRENT_LIST_DIR}/llgo_out.o")

# Route Go fatal exits (exit() -> _exit) to the component's parking wrapper
# instead of IDF's chip-restarting _exit (see app_main.c). The extra -T script
# only assigns _globals_start/_globals_end; the assignments evaluate at layout
# time, so its position in the -T list does not matter.
target_link_options(${COMPONENT_LIB} PUBLIC
                    "-Wl,--wrap=_exit"
                    "-T" "${CMAKE_CURRENT_LIST_DIR}/llgo_dram.ld")
