#include <windows.h>
#include <winternl.h>
#include <stdio.h>

//gcc -shared -o test_struct.dll test.c

// Проверка валидности любого указателя
int is_valid_ptr(void* p, SIZE_T size) {
    MEMORY_BASIC_INFORMATION mbi;
    if (!VirtualQuery(p, &mbi, sizeof(mbi)))
        return 0;

    return (mbi.State == MEM_COMMIT &&
            !(mbi.Protect & PAGE_NOACCESS) &&
            !(mbi.Protect & PAGE_GUARD));
}

// Печать текста из UNICODE_STRING
void print_unicode_string(UNICODE_STRING* ustr) {
    if (!ustr) {
        printf("(UNICODE_STRING = NULL)\n");
        return;
    }
    if (!ustr->Buffer) {
        printf("(Buffer = NULL)\n");
        return;
    }
    
    // Length в байтах, поэтому делим на sizeof(WCHAR)
    wprintf(L"%.*ls\n", ustr->Length / sizeof(WCHAR), ustr->Buffer);
}

__declspec(dllexport)
void DumpObjectAttributes(HANDLE* h, POBJECT_ATTRIBUTES oa, PIO_STATUS_BLOCK iosb)
{
    printf("\n==================== DUMP ====================\n");

    // HANDLE*
    printf("Handle ptr: %p  valid=%d\n", h, is_valid_ptr(h, sizeof(HANDLE)));
    if (is_valid_ptr(h, sizeof(HANDLE)))
        printf("Handle value: 0x%p\n", *h);

    // OBJECT_ATTRIBUTES
    printf("\nOBJECT_ATTRIBUTES ptr: %p  valid=%d\n",
           oa, is_valid_ptr(oa, sizeof(OBJECT_ATTRIBUTES)));

    if (!oa || !is_valid_ptr(oa, sizeof(OBJECT_ATTRIBUTES))) {
        printf("oa invalid or NULL\n");
        goto END;
    }

    printf("Length: %lu\n", oa->Length);
    printf("RootDirectory: %p\n", oa->RootDirectory);
    printf("ObjectName ptr: %p  valid=%d\n",
           oa->ObjectName,
           is_valid_ptr(oa->ObjectName, sizeof(UNICODE_STRING)));
    printf("Attributes: 0x%lx\n", oa->Attributes);

    // UNICODE_STRING
    if (oa->ObjectName && is_valid_ptr(oa->ObjectName, sizeof(UNICODE_STRING))) {
        printf("\nUNICODE_STRING:\n");
        printf("  Length: %u\n", oa->ObjectName->Length);
        printf("  MaximumLength: %u\n", oa->ObjectName->MaximumLength);
        printf("Buffer hex %#x\n", oa->ObjectName->Buffer);
        printf("  Buffer ptr: %p  valid=%d\n",
               oa->ObjectName->Buffer,
               is_valid_ptr(oa->ObjectName->Buffer, oa->ObjectName->Length));

        printf("  Text: ");
        print_unicode_string(oa->ObjectName);
    } else {
        printf("\nObjectName is NULL or invalid\n");
    }

    // IO_STATUS_BLOCK
    printf("\nIO_STATUS_BLOCK ptr: %p  valid=%d\n",
           iosb,
           is_valid_ptr(iosb, sizeof(IO_STATUS_BLOCK)));

    if (iosb && is_valid_ptr(iosb, sizeof(IO_STATUS_BLOCK))) {
        printf("  Status: 0x%lx\n", iosb->Status);
        printf("  Information: 0x%p\n", (void*)iosb->Information);
    }

    *h = *h = (HANDLE)(uintptr_t)1000;

END:
    printf("=============== END OF DUMP =================\n\n");
}
