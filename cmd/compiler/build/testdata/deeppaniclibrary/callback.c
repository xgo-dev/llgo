#if defined(_WIN32)
__declspec(dllexport)
#endif
void callback_bridge(void (*callback)(void)) {
    callback();
    /* Retain the native return frame even under optimization. */
    static volatile unsigned returned;
    returned++;
}
