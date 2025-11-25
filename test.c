//gcc -shared -o test_struct.dll test.c

#include <windows.h>
#include <winternl.h>
#include <stdio.h>

// Определяем тип функции NtOpenFile
typedef NTSTATUS (NTAPI *NtOpenFile_t)(
    PHANDLE            FileHandle,
    ACCESS_MASK        DesiredAccess,
    POBJECT_ATTRIBUTES ObjectAttributes,
    PIO_STATUS_BLOCK   IoStatusBlock,
    ULONG              ShareAccess,
    ULONG              OpenOptions
);

__declspec(dllexport) NTSTATUS NtOpenFile(
    PHANDLE            FileHandle,
    ACCESS_MASK        DesiredAccess,
    POBJECT_ATTRIBUTES ObjectAttributes,
    PIO_STATUS_BLOCK   IoStatusBlock,
    ULONG              ShareAccess,
    ULONG              OpenOptions
)
{
    printf("=== NtOpenFile Arguments ===\n");

    printf("FileHandle: %p\n", FileHandle);
    if (FileHandle) {
        printf("  *FileHandle: %p\n", *FileHandle);
    }

    printf("DesiredAccess: 0x%X\n", DesiredAccess);

    if (ObjectAttributes) {
        printf("ObjectAttributes: %p\n", ObjectAttributes);
        printf("%p\n", ObjectAttributes->ObjectName);
        if (ObjectAttributes->ObjectName && ObjectAttributes->ObjectName->Buffer) {
            printf("OKAY I GOT HERE");
            wprintf(L"  ObjectName: %.*ls\n",
                    ObjectAttributes->ObjectName->Length / sizeof(WCHAR),
                    ObjectAttributes->ObjectName->Buffer);
        }
        printf("SIGMA");
        printf("  Attributes: 0x%X\n", ObjectAttributes->Attributes);
        if (ObjectAttributes->SecurityQualityOfService) {
            SECURITY_QUALITY_OF_SERVICE* sqos = (SECURITY_QUALITY_OF_SERVICE*)ObjectAttributes->SecurityQualityOfService;

            printf("  SecurityQualityOfService: %p\n", ObjectAttributes->SecurityQualityOfService);
            printf("    Length: %u\n", sqos->Length);
            printf("    ImpersonationLevel: %u\n", sqos->ImpersonationLevel);
            printf("    ContextTrackingMode: %u\n", sqos->ContextTrackingMode);
            printf("    EffectiveOnly: %u\n", sqos->EffectiveOnly);
        } else {
            printf("  SecurityQualityOfService: NULL\n");
        }
    }

    if (IoStatusBlock) {
        printf("IoStatusBlock: %p\n", IoStatusBlock);
    }

    printf("ShareAccess: 0x%X\n", ShareAccess);
    printf("OpenOptions: 0x%X\n", OpenOptions);

    *FileHandle = (HANDLE)1000;

    IoStatusBlock->Information = 10000;
    IoStatusBlock->Status = 10000;

    // Возвращаем STATUS_SUCCESS, не открывая файл
    return 0;
}
