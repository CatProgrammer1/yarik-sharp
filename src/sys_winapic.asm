bits 64
global _start
_start:
    sub rsp, 40
    mov rax, rcx  ; адрес функции
    call rax
    add rsp, 40
    ret