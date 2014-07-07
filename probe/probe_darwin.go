package probe

import (
    "fmt"
    "time"
    "syscall"
)

func boottime() (int64, error) {
    value, err := syscall.Sysctl("kern.boottime")

    if err != nil {
        return -1, err
    }

    bytes := []byte(value[:])

    fmt.Println("V is", bytes)
    var boottime int64
    boottime = int64(bytes[0]) + int64(bytes[1])*256 + int64(bytes[2])*256*256 + int64(bytes[3])*256*256*256

    return boottime, nil
}

func Uptime() (int64, error) {
    boot, err := boottime()

    if err != nil {
        return 0, err
    }

    // uptime = current - boot
    uptime := (time.Now().Unix() - int64(boot))

    return uptime, err
}

