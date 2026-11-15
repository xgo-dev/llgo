package bdwgc

const LLGoPackage = "link: $(pkg-config --libs bdw-gc); -lgc"

const LLGoFiles = blockingLLGoFiles + gcPolicyLLGoFiles
