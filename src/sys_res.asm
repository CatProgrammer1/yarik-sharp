bits 64 ;nasm -f bin src\sys_res.asm -o src\sys_res.bin

global _start

_start:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    ; RCX = ptr_module_name (LoadLibraryA)
    mov rax, [rel loadlib_addr]
    sub rsp, 40
    call rax
    add rsp, 40

    test rax, rax
    jz .ret_zero

    ; RCX = hModule, RDX = ptr_func_name (GetProcAddress)
    mov rcx, rax
    mov rdx, [rel func_ptr]
    mov rax, [rel getproc_addr]
    sub rsp, 40
    call rax
    add rsp, 40

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

.ret_zero:
    xor rax, rax
    jmp .done

; Плейсхолдеры для патчинга из Go
loadlib_addr: dq 0xAAAAAAAAAAAAAAAA
getproc_addr: dq 0xBBBBBBBBBBBBBBBB
func_ptr:     dq 0xCCCCCCCCCCCCCCCC  ; сюда Go запишет fPtr