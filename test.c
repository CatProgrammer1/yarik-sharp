//gcc -shared -o test_struct.dll test.c

#include <stdio.h>

__declspec(dllexport) void Sigma(short* a)
{   
   a[1] = 32000;
}
