#include <stdarg.h>

typedef struct PyObject PyObject;

PyObject *PyObject_GetAttrString(PyObject *object, const char *name);

void llgoLoadPyModSyms(PyObject *module, ...) {
    va_list args;
    va_start(args, module);
    for (;;) {
        const char *name = va_arg(args, const char *);
        if (name == 0) {
            break;
        }
        PyObject **slot = va_arg(args, PyObject **);
        if (*slot == 0) {
            *slot = PyObject_GetAttrString(module, name);
        }
    }
    va_end(args);
}
