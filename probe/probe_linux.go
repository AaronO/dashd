package probe

import (
    "syscall"
)

func Uptime() (int, error) {
    var info syscall.Sysinfo_t

    err := syscall.Sysinfo(&info)

    if err != nil {
        return 0, err
    }

    return info.Uptime, nil
}
