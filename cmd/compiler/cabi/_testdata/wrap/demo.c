extern int printf(const char *format, ...);

int demo32(int v) {
    return v+100;
}

long long demo64(long long v) {
    return v+100;
}

struct struct32 {
     int v;
};

struct point64 {
    long long x;
    long long y;
};

struct point64 pt64(struct point64 pt) {
    printf("point64: %lld %lld\n",pt.x,pt.y);
    return pt;
}

struct struct32 demo32s(struct struct32 v) {
    printf("struct32: %d\n",v.v);
    struct struct32 v2 = {v.v+100};
    return v2;
}

struct point {
    int x;
    int y;
};

struct point pt(struct point pt) {
    printf("point: %d %d\n",pt.x,pt.y);
    return pt;
}

struct point1 {
    int x;
    int y;
    int z;
};

struct point1 pt1(struct point1 pt) {
    printf("point1: %d %d %d\n",pt.x,pt.y,pt.z);
    return pt;
}

struct point2 {
    char x;
    int y;
    int z;
};

struct point2 pt2(struct point2 pt) {
    printf("point2: %d %d %d\n",pt.x,pt.y,pt.z);
    return pt;
}

struct point3 {
    char x;
    char y;
    char z;
};

struct point3 pt3(struct point3 pt) {
    printf("point3: %d %d %d\n",pt.x,pt.y,pt.z);
    return pt;
}

struct point4 {
    char x;
    char y;
    char z;
    int  m;
};

struct point4 pt4(struct point4 pt) {
    printf("point4: %d %d %d %d\n",pt.x,pt.y,pt.z,pt.m);
    return pt;
}

struct point5 {
    char x;
    char y;
    char z;
    char m;
    char n;
};

struct point5 pt5(struct point5 pt) {
    printf("point5: %d %d %d %d %d\n",pt.x,pt.y,pt.z,pt.m,pt.n);
    return pt;
}

struct point6 {
    char x;
    char y;
    char z;
    char m;
    char n;
    int  k;
};

struct point6 pt6(struct point6 pt) {
    printf("point6: %d %d %d %d %d %d\n",pt.x,pt.y,pt.z,pt.m,pt.n,pt.k);
    return pt;
}

struct point7 {
    char x;
    char y;
    char z;
    char m;
    char n;
    int  k;
    char o;
};

struct point7 pt7(struct point7 pt) {
    printf("point7: %d %d %d %d %d %d %d\n",pt.x,pt.y,pt.z,pt.m,pt.n,pt.k,pt.o);
    return pt;
}

struct data1 {
    char x;
    long long y;
};

struct data1 fn1(struct data1 pt) {
    printf("data1: %d %lld\n",pt.x,pt.y);
    return pt;
}

struct data2 {
    int x;
    long long y;
};

struct data2 fn2(struct data2 pt) {
    printf("data2: %d %lld\n",pt.x,pt.y);
    return pt;
}

struct data3 {
    long long x;
    char y;
};

struct data3 fn3(struct data3 pt) {
    printf("data3: %lld %d\n",pt.x,pt.y);
    return pt;
}

struct fdata1 {
    float x;
};

struct fdata1 ff1(struct fdata1 pt) {
    printf("ff1: %f\n",pt.x);
    return pt;
}

struct ddata1 {
    double x;
};

struct ddata1 dd1(struct ddata1 pt) {
    printf("dd1: %f\n",pt.x);
    return pt;
}

struct ddata2 {
    double x;
    double y;
};

struct ddata2 dd2(struct ddata2 pt) {
    printf("dd2: %f %f\n",pt.x,pt.y);
    return pt;
}

struct ddata3 {
    double x;
    double y;
    double z;
};

struct ddata3 dd3(struct ddata3 pt) {
    printf("dd3: %f %f %f\n",pt.x,pt.y,pt.z);
    return pt;
}

struct fdata2i {
    float x;
    int   y;
};

struct fdata2i ff2i(struct fdata2i pt) {
    printf("ff2i: %f %d\n",pt.x,pt.y);
    return pt;
}

struct fdata2 {
    float x;
    float y;
};

struct fdata2 ff2(struct fdata2 pt) {
    printf("ff2: %f %f\n",pt.x,pt.y);
    return pt;
}

struct fdata3 {
    float x;
    float y;
    float z;
};

struct fdata3 ff3(struct fdata3 pt) {
    printf("ff3: %f %f %f\n",pt.x,pt.y,pt.z);
    return pt;
}

struct fdata4 {
    float x;
    float y;
    float z;
    float m;
};

struct fdata4 ff4(struct fdata4 pt) {
    printf("ff4: %f %f %f %f\n",pt.x,pt.y,pt.z,pt.m);
    return pt;
}

struct fdata5 {
    float x;
    float y;
    float z;
    float m;
    float n;
};

struct fdata5 ff5(struct fdata5 pt) {
    printf("ff5: %f %f %f %f %f\n",pt.x,pt.y,pt.z,pt.m,pt.n);
    return pt;
}

struct fdata2id {
    char     x;
    char     y;
    double   z;
};

struct fdata2id ff2id(struct fdata2id pt) {
    printf("ff6: %d %d %f\n",pt.x,pt.y,pt.z);
    return pt;
}

struct fdata7if {
    char    x[7];
    float    z;
};

struct fdata7if ff7if(struct fdata7if pt) {
    printf("ff7if: %d %d %f\n",pt.x[0],pt.x[1],pt.z);
    return pt;
}

struct fdata4if {
    float    x;
    char     y;
    float    z;
    float    m;
};

struct fdata4if ff4if(struct fdata4if pt) {
    printf("ff4if: %f %d %f %f\n",pt.x,pt.y,pt.z,pt.m);
    return pt;
}

struct array {
    int x[8];
};

struct array demo(struct array a) {
    printf("demo: %d %d %d\n",a.x[0],a.x[1],a.x[2]);
    return a;
}

struct array demo2(int a1){
    struct array x;
    for (int i = 0; i < 8; i++) {
        x.x[i] = i+a1;
    }
    return x;
}

void callback(struct array (*fn)(struct array ar, struct point pt, struct point1 pt1), struct array ar) {
    demo(ar);
    struct point pt = {1,2};
    struct point1 pt1 = {1,2,3};
    struct array ret = fn(ar,pt,pt1);
    demo(ret);
}

void callback1(struct point (*fn)(struct array ar, struct point pt, struct point1 pt1), struct array ar) {
    printf("callback1 array: %d %d %d\n",ar.x[0],ar.x[1],ar.x[7]);
    struct point pt = {1,2};
    struct point1 pt1 = {1,2,3};
    struct point ret = fn(ar,pt,pt1);
    printf("callback1 ret: %d,%d\n",ret.x,ret.y);
}

struct point mycallback(struct array ar, struct point pt, struct point1 pt1) {
    printf("mycallback array: %d %d %d\n",ar.x[0],ar.x[1],ar.x[7]);
    printf("mycallback pt: %d %d\n",pt.x,pt.y);
    printf("mycallback pt1: %d %d %d\n",pt1.x,pt1.y,pt1.z);
    struct point ret = {pt.x+pt1.x, pt.y+pt1.y};
    return ret;
}
