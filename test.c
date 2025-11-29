// gcc -shared -o test_struct.dll test.c
#include <windows.h>
#include <winternl.h>  // для IO_STATUS_BLOCK
#include <stdio.h>

__declspec(dllexport) void Sigma(
    HANDLE FileHandle,
    HANDLE Event,
    PIO_APC_ROUTINE ApcRoutine,
    PVOID ApcContext,
    PIO_STATUS_BLOCK IoStatusBlock,
    PVOID Buffer,
    ULONG Length,
    PLARGE_INTEGER ByteOffset,
    PULONG Key
)
{
    printf("FileHandle: %p\n", FileHandle);

    printf("Event: %p\n", Event);
    printf("ApcRoutine: %p\n", ApcRoutine);
    printf("ApcContext: %p\n", ApcContext);

    printf("IoStatusBlock: %p\n", IoStatusBlock);
    if (IoStatusBlock) {
        printf("\tIoStatusBlock->Status: 0x%X\n", IoStatusBlock->Status);
        printf("\tIoStatusBlock->Information: %llu\n", IoStatusBlock->Information);
    }

    printf("Buffer: %p\n", Buffer);
    unsigned char* buf = (unsigned char*)Buffer;
    for (int i = 0; i < 6; i++) {
      printf("\telement %d: %d\n", i, buf[i]);
      buf[i] = 5;
      printf("\telement %d: %d\n", i, buf[i]);
    }
    printf("\nLength: %u\n", Length);
    printf("ByteOffset: %p\n", ByteOffset);
    if (ByteOffset) {
        printf("ByteOffset->QuadPart: %lld\n", ByteOffset->QuadPart);
    }
    printf("Key: %p\n", Key);
    if (Key) {
        printf("*Key: %lu\n", *Key);
    }

    fflush(stdout); // чтобы вывод сразу появился
}
