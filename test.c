#include <stdio.h>

//gcc -shared -o test_struct.dll test.c

__declspec(dllexport) void Sigma(long* a) {
    *a = 1488;
}
