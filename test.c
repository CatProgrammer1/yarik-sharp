//gcc -shared -o test_struct.dll test.c

#include <stdio.h>

__declspec(dllexport) int Sigma(int* a)
{   
   a[0] = 10;
   return 0;
}
