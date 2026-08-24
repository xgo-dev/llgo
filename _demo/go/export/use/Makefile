# Makefile for C demo using Go exported library
# Use LINK_TYPE environment variable to choose library type:
# LINK_TYPE=static  - Link with static library (default)
# LINK_TYPE=shared  - Link with shared library

CC = clang
CFLAGS = -Wall -Wextra -std=c99
INCLUDES = -I..
TARGET = main.out
SOURCES = main.c
HEADER = ../libexport.h
RUNTIME_LIBS = -lpthread -lm $(shell pkg-config --libs bdw-gc || echo -lgc) $(shell pkg-config --libs libuv || echo -luv)

# Default to static linking
LINK_TYPE ?= static
LLGOFLAGS ?=

# Platform detection
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    SHARED_EXT = dylib
    PLATFORM_LIBS =
else
    SHARED_EXT = so
    PLATFORM_LIBS = $(shell pkg-config --libs libunwind 2>/dev/null || echo -lunwind)
endif
STATIC_STDLIB_LIBS = $(shell pkg-config --libs libffi 2>/dev/null || echo -lffi) -lresolv

# Library and flags based on link type
ifeq ($(LINK_TYPE),shared)
    BUILDMODE = c-shared
    LIBRARY = ../libexport.$(SHARED_EXT)
    LDFLAGS = -L.. -lexport $(RUNTIME_LIBS) $(PLATFORM_LIBS)
    BUILD_MSG = "Building Go shared library..."
    LINK_MSG = "Linking with shared library..."
else
    BUILDMODE = c-archive
    LIBRARY = ../libexport.a
    LDFLAGS = $(LIBRARY) $(RUNTIME_LIBS) $(PLATFORM_LIBS) $(STATIC_STDLIB_LIBS)
    BUILD_MSG = "Building Go static library..."
    LINK_MSG = "Linking with static library..."
endif

.PHONY: all clean run build-go

all: build-go $(TARGET)

# Build the Go library first
build-go:
	@echo $(BUILD_MSG)
	cd .. && ../../../dev/llgo.sh build -buildmode $(BUILDMODE) $(LLGOFLAGS) -o $(notdir $(LIBRARY)) .

# Build the C executable
$(TARGET): $(SOURCES) $(LIBRARY) $(HEADER)
	@echo $(LINK_MSG)
	$(CC) $(CFLAGS) $(INCLUDES) -o $(TARGET) $(SOURCES) $(LDFLAGS)

# Run the executable
run: $(TARGET)
	@echo "Running C demo..."
ifeq ($(LINK_TYPE),shared)
	@echo "Setting library path for shared library..."
	LD_LIBRARY_PATH=.. DYLD_LIBRARY_PATH=.. ./$(TARGET)
else
	./$(TARGET)
endif

# Clean build artifacts
clean:
	rm -f $(TARGET)
	rm -f ../libexport.a ../libexport.h ../libexport.so ../libexport.dylib

# Help target
help:
	@echo "Available targets:"
	@echo "  all       - Build Go library and C executable"
	@echo "  build-go  - Build only the Go library"
	@echo "  run       - Build and run the C demo"
	@echo "  clean     - Clean all build artifacts"
	@echo "  help      - Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  LINK_TYPE - Library type: 'static' (default) or 'shared'"
	@echo ""
	@echo "Examples:"
	@echo "  make run                    # Use static library"
	@echo "  LINK_TYPE=shared make run   # Use shared library"
