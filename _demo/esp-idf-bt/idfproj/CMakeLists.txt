# Minimal ESP-IDF project whose application logic is an llgo build artifact.
#
# llgo compiles the Go program into a relocatable object (targets/esp32s31-idf,
# ld.lld -r); the main component links that object into the IDF firmware and
# calls the generated C entry from app_main.
cmake_minimum_required(VERSION 3.16)

include($ENV{IDF_PATH}/tools/cmake/project.cmake)
project(llgo_idf_s31)
