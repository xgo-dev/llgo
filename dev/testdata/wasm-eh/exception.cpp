#include <csetjmp>
#include <cstdio>

static std::jmp_buf checkpoint;
static int jump_count;

static void jump_back() {
    ++jump_count;
    std::longjmp(checkpoint, 1);
}

int main() {
    if (setjmp(checkpoint) == 0) {
        jump_back();
    }
    if (jump_count != 1) {
        std::puts("sjlj wrong");
        return 1;
    }
    try {
        throw 7;
    } catch (int value) {
        std::puts(value == 7 ? "cpp catch and sjlj ok" : "cpp catch wrong");
        return value == 7 ? 0 : 1;
    }
}
