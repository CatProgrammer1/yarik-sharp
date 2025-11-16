#include <windows.h>
#include <winternl.h>
#include <stdio.h>

//gcc -shared -o test_struct.dll test.c

__declspec(dllexport) void check_OBJECT_ATTRIBUTES(POBJECT_ATTRIBUTES obj) {
    printf("Length: %lu\n", obj->Length);
    printf("RootDirectory: %p\n", obj->RootDirectory);
    printf("Buffer: %p\n", obj->ObjectName->Buffer);

    wprintf(L"String (ObjectName): %.*ls\n",
            obj->ObjectName->Length / 2,
            obj->ObjectName->Buffer);

    printf("Length: %u\n", obj->ObjectName->Length);
    printf("MaximumLength: %u\n", obj->ObjectName->MaximumLength);
    printf("Attributes: %lu\n", obj->Attributes);

    obj->Length = -1;
}
