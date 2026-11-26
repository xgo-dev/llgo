# Main component: a thin C shell around the llgo build artifact of the
# audiocheck demo (opus self-check + voice loop + wakenet).
#
# The Go program and the llgo runtime symbol providers are merged into
# main/llgo_out.o on the host by host/make-artifact.sh (see its header for
# why the providers must live inside the object rather than in this
# component). This component supplies the C shell, the package glues for
# the codec and the wakenet engine, the demo-local glue (the energy VAD,
# the esp_timer clocks, the heap ledger, the test-PCM injection and the
# MCLK hook), the extra linker script and the link options.
#
# Glue sources: demo_glue.c and diag_glue.c are demo-local (checked in); opus_glue.c and
# wakenet_glue.c are build-time copies of the pkg/esp/opus and
# pkg/esp/wakenet glue templates, synced from the llgo repo by
# host/make-artifact.sh (integration way 1 in pkg/esp/README.md) and
# therefore gitignored - post-sync.sh patches the wakenet copy with this
# board's MCLK pin. The gpio/i2c/i2s glues are synced the same way and
# carry the board pins.
#
# WHOLE_ARCHIVE is required: llgo_out.o is the LAST item on the link line,
# and the ESP-IDF link has no --start-group, so each component archive is
# scanned only against the undefined symbols known at its position. The
# glues are referenced solely from llgo_out.o; a plain archive would never
# pull them, exactly like the llgo_syms.S providers that therefore live
# inside llgo_out.o itself. Whole-archive keeps every main-component
# object in the link unconditionally.
#
# REQUIRES 78__esp-opus: the codec glue calls opus_encoder_create /
# opus_encode / opus_decoder_create / opus_decode and the ctl requests
# against the component's library (idf_component.yml: 78/esp-opus).
# REQUIRES esp-sr: the wakenet glue maps the model partition, creates the
# wn9_hilexin engine and runs its detect loop (idf_component.yml:
# espressif/esp-sr). The esp_driver_* components carry the i2c/i2s/gpio
# peripherals the synced glues drive; esp_timer the demo's clocks. The
# esp32s31 default components cover the rest (freertos for app_main's
# parking loop, esp_hw_support for heap_caps).
idf_component_register(SRCS app_main.c demo_glue.c diag_glue.c wakenet_pcm.c opus_glue.c gpio_glue.c i2c_glue.c i2s_glue.c wakenet_glue.c
                       INCLUDE_DIRS "."
                       REQUIRES 78__esp-opus esp-sr esp_driver_gpio esp_driver_i2c esp_driver_i2s esp_timer
                       WHOLE_ARCHIVE)

# The esp-sr prebuilt speech-features archive ships a stray FFT command-line
# tool (fftutil.c.obj: a main() plus esp_kiss_fftndr* imports nothing
# provides in this link). The esp-sr archives sit before llgo_out.o on the
# link line, so the tool object would be pulled first - its main()
# colliding with llgo's generated entry. Delete the member from this
# target's copy of the archive (the fetch happens before any component
# CMakeLists is processed; the path is the component manager's fixed
# layout, lib/<target>/). NOTE: the member probe runs via the literal path,
# NOT file(GLOB*): under the IDF project configure the glob helpers return
# nothing even for paths that EXISTS confirms (script mode works, project
# mode does not). The delete is idempotent and self-healing: an archive
# already stripped (or a re-fetch on a clean tree) ends with the member
# gone either way.
set(espsr_speech_lib "${CMAKE_CURRENT_LIST_DIR}/../managed_components/espressif__esp-sr/lib/esp32s31/libc_speech_features.a")
if(EXISTS "${espsr_speech_lib}")
    find_program(esp_sr_ar riscv32-esp-elf-ar)
    if(NOT esp_sr_ar)
        find_program(esp_sr_ar ar REQUIRED)
    endif()
    # No *_QUIET here: with OUTPUT_QUIET set, execute_process suppresses the
    # OUTPUT_VARIABLE capture too, and the member check would always miss.
    execute_process(COMMAND "${esp_sr_ar}" "t" "${espsr_speech_lib}"
                    OUTPUT_VARIABLE speech_members
                    ERROR_QUIET)
    if(speech_members MATCHES "fftutil\\.c\\.obj")
        execute_process(COMMAND "${esp_sr_ar}" "d" "${espsr_speech_lib}" "fftutil.c.obj"
                        RESULT_VARIABLE esp_sr_ar_rc
                        OUTPUT_QUIET ERROR_QUIET)
        if(NOT esp_sr_ar_rc EQUAL 0)
            message(FATAL_ERROR "esp-sr: failed to strip fftutil.c.obj from ${espsr_speech_lib} (ar d rc=${esp_sr_ar_rc})")
        endif()
        message(STATUS "esp-sr: stripped fftutil.c.obj from ${espsr_speech_lib}")
    endif()
endif()

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
