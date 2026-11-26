# The P0 pure-C smoke project: does esp-sr build, link and run on S31 at all,
# before any llgo involvement. No llgo artifact, no glue - one C app_main
# against the esp-sr component (see main/wakenet_smoke.c for what it proves).
cmake_minimum_required(VERSION 3.16)
include($ENV{IDF_PATH}/tools/cmake/project.cmake)
project(wakenet_csmoke)
