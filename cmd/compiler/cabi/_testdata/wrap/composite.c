struct basearray1 {
    int x[1];
};

struct array10 {
    struct basearray1 base;
};

struct array10 demo_array10(struct array10 a) {
    return a;
}

struct array11 {
    struct basearray1 base;
    int z;
};

struct array11 demo_array11(struct array11 a) {
    return a;
}

struct basepoint1 {
    int x; 
};

struct point10 {
    struct basepoint1 base;
};

struct point10 demo_point10(struct point10 a) {
    return a;
}

struct point11 {
    struct basepoint1 base;
    int z;
};

struct point11 demo_point11(struct point11 a) {
    return a;
}

struct basearray2 {
    int x[2];
};

struct array20 {
    struct basearray2 base;
};

struct array20 demo_array20(struct array20 a) {
    return a;
}

struct array21 {
    struct basearray2 base;
    int z;
};

struct array21 demo_array21(struct array21 a) {
    return a;
}

struct basepoint2 {
    int x;
    int y;  
};

struct point20 {
    struct basepoint2 base;
};

struct point20 demo_point20(struct point20 a) {
    return a;
}

struct point21 {
    struct basepoint2 base;
    int z;
};

struct point21 demo_point21(struct point21 a) {
    return a;
}