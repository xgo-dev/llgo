extern "C" int llgo_eh_cpp_catch() {
    try {
        throw 7;
    } catch (int value) {
        return value;
    }
}
