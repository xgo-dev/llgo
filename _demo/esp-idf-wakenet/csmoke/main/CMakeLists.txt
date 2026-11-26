idf_component_register(SRCS "wakenet_smoke.c"
                       INCLUDE_DIRS "." "samples"
                       REQUIRES esp-sr esp_timer
                       WHOLE_ARCHIVE)

target_compile_options(${COMPONENT_LIB} PRIVATE "-Wno-format")
