package main

import (
    "os"
    "os/exec"

    "./probe"

    "time"
    "net/http"

    "errors"

    "io"
    "io/ioutil"

    "log"

//    "fmt"
    "strings"
    "runtime"
    "strconv" // For Itoa
//    "encoding/csv"
    "encoding/json"

    "github.com/go-martini/martini"
)


func main() {
    m := martini.Classic()

    // CPU count
    m.Get("/sh/numberofcores.php", func () ([]byte, error) {
        return json.Marshal(runtime.NumCPU())
    })

    // Server's hostname
    m.Get("/sh/hostname.php", func () ([]byte, error) {
        host, err := os.Hostname()

        if err != nil {
            return nil, err
        }

        return json.Marshal(host)
    })



    // PS
    m.Get("/sh/ps.php", func () ([]byte, error) {
        // Run uptime command
        rawOutput, err := exec.Command("ps", "aux").Output()

        if err != nil {
            return nil, err
        }

        return json.Marshal(parseCommandTable(rawOutput, 1, 1))
    })

    m.Get("/sh/df.php", func () ([]byte, error) {
        // Run uptime command
        rawOutput, err := exec.Command("df", "-Ph").Output()

        if err != nil {
            return nil, err
        }

        return json.Marshal(parseCommandTable(rawOutput, 1, 1))
    })

    m.Get("/sh/time.php", func () ([]byte, error) {
        raw, err := exec.Command("date").Output()

        if err != nil {
            return nil, err
        }

        return json.Marshal(string(raw[:]))
    })

    m.Get("/sh/issue.php", func () ([]byte, error) {
        raw, err := exec.Command("uname", "-rsm").Output()

        if err != nil {
            return nil, err
        }

        return json.Marshal(string(raw[:]))
    })

    m.Get("/sh/users.php", func () ([]byte, error) {
        data, err := ioutil.ReadFile("/etc/passwd")

        if err != nil {
            return nil, err
        }

        lines := strings.Split(string(data), "\n")

        // Output records
        var records [][]string

        for _, line := range lines {
            parts := strings.Split(line, ":")

            // Skip bad or empty lines
            if len(parts) != 7 {
                log.Println(len(parts))
                continue
            }

            // Parse base 10, 16 bit UID integer
            uid, err := strconv.ParseInt(parts[2], 10, 16)

            // Error parsing UID
            if err != nil {
                continue
            }

            userType := "user"

            // Check if system user
            if uid <= 499 {
                userType = "system"
            }

            user := []string{
                // User type
                userType,
                // Username
                parts[0],
                // Home directory
                parts[6],
            }

            records = append(records, user)
        }

        return json.Marshal(records)
    })

    m.Get("/sh/online.php", func () ([]byte, error) {
        raw, err := exec.Command("w").Output()

        if err != nil {
            return nil, err
        }

        // We'll add all the parsed lines here
        var entries [][]string

        // Skip first and last line of output
        for _, entry := range parseCommandTable(raw, 2, 1) {
            entries = append(entries, []string{
                // User
                entry[0],
                // From
                entry[2],
                // Login at
                entry[3],
                // Idle
                entry[4],
            })
        }

        return json.Marshal(entries)
    })

    m.Get("/sh/loadavg.php", func () ([]byte, error) {
        raw, err := exec.Command("w").Output()

        if err != nil {
            return nil, err
        }

        lines := strings.Split(string(raw[:]), "\n")
        headers := strings.Fields(lines[0])

        var cpuLoads [][2]string

        // Last 3 headers are CPU loads, clean them up
        for _, load := range headers[len(headers)-3:] {
            // Remove trailing coma if there
            cleanLoad := strings.Split(string(load), ",")[0]

            loadFloat, err := strconv.ParseFloat(cleanLoad, 32)

            if err != nil {
                continue
            }

            loadPercentage := int(loadFloat * 100) / runtime.NumCPU()

            cpuLoads = append(cpuLoads, [2]string{
                cleanLoad,
                strconv.Itoa(loadPercentage),
            })
        }

        return json.Marshal(cpuLoads)
    })

    m.Get("/sh/ping.php", func () ([]byte, error) {
        HOSTS := [4]string{
            "wikipedia.org",
            "github.com",
            "google.com",
            "gnu.org",
        }

        var entries [][]string

        for _, host := range HOSTS {
            ping, err := pingHost(host, 2)

            if err != nil {
                continue
            }

            entries = append(entries, []string{
                host,
                ping,
            })
        }

        return json.Marshal(entries)
    })

    m.Get("/sh/netstat.php", func () ([]byte, error) {
        raw, err := exec.Command("netstat", "-nt").Output()

        if err != nil {
            return nil, err
        }

        // Get entries as a table
        entries := parseCommandTable(raw, 2, 1)

        // Connections we care about
        var connections [][]string

        // Filter by protocol
        for _, entry := range entries {
            // Skip bad entries or non tcp
            if len(entry) != 6 || (entry[0] != "tcp" && entry[0] != "tcp4") {
                continue
            }

            connections = append(connections, entry)
        }

        // Count connections from a machine
        connectionCount := make(map[string]int)

        // Extract
        for _, conn := range connections {
            // Foreign Address
            remoteAddr := conn[4]

            // Increment counter
            connectionCount[remoteAddr]++
        }

        // Expected output by client
        var output [][2]string

        for remoteAddr, count := range connectionCount {
            // Compensate for difference between OSX and linux
            parts := strings.Split(strings.Replace(remoteAddr, ",", ".", 1), ".")
            ip := strings.Join(parts[:4], ".")
            // port := parts[4]

            output = append(output, [2]string{
                strconv.Itoa(count),
                ip,
            })
        }

        return json.Marshal(output)
    })

    m.Get("/sh/where.php", func () ([]byte, error) {
        SOFTWARE := []string{
            "php",
            "node",
            "mysql",
            "vim",
            "python",
            "ruby",
            "java",
            "apache2",
            "nginx",
            "openssl",
            "vsftpd",
            "make",
        }

        var entries [][2]string

        for _, bin := range SOFTWARE {
            path, err := exec.LookPath(bin)

            if err != nil {
                path = "Not Installed"
            }

            entries = append(entries, [2]string{
                bin,
                path,
            })
        }

        return json.Marshal(entries)
    })

    m.Get("/sh/mem.php", func () ([]byte, error) {
        var entries []string

        raw, err := exec.Command("free", "-m").Output()

        if err != nil {
            // OS X fallback
            raw, err := exec.Command("vm_stat").Output()

            if err != nil {
                return nil, err
            }

            vmMap := parseCommandMap(string(raw[:]))

            freePages := toMB(vmMap["Pages free"])
            activePages := toMB(vmMap["Pages active"])
            inactivePages := toMB(vmMap["Pages inactive"])
            speculativePages := toMB(vmMap["Pages speculative"])

            free := freePages + speculativePages
            used := activePages + inactivePages

            return json.Marshal([]string{
                "Mem:",

                // Total
                strconv.Itoa(free + used),

                // Used
                strconv.Itoa(used),

                // Free
                strconv.Itoa(free),
            })
        }

        // Linux
        entries = parseCommandTable(raw, 2, 1)[0][:4]

        return json.Marshal(entries)
    })

    m.Get("/sh/lastlog.php", func () ([]byte, error) {
        raw, err := exec.Command("last").Output()

        if err != nil {
            return nil, err
        }

        entries := parseCommandTable(raw, 0, 3)

        var data [][]string

        for _, entry := range entries {
            var row []string

            // Non user action
            if entry[1] == "~" {
                continue
            }

            if runtime.GOOS == "linux" {
                row = []string{
                    // Username
                    entry[0],
                    entry[2],
                    strings.Join(entry[3:6], " "),
                }
            } else if runtime.GOOS == "darwin" {
                row = []string{
                    // Username
                    entry[0],
                    // From
                    "",
                    strings.Join(entry[2:6], " "),
                }
            }

            data = append(data, row)
        }

        return json.Marshal(data)
    })

    m.Get("/sh/swap.php", func () ([]byte, error) {
        // Linux
        data, err := ioutil.ReadFile("/proc/swaps")

        if err != nil {
            // OS X fallback
            stats, err := topStats()

            // Failed everything
            if err != nil {
                return nil, err
            }

            fields := strings.Fields(stats["Swap"])

            used, _ := strconv.ParseInt(fields[0][:len(fields[0])-1], 10, 32)
            free, _ := strconv.ParseInt(fields[2][:len(fields[2])-1], 10, 32)

            usedB := mbToB(int(used))
            freeB := mbToB(int(free))

            return json.Marshal([][]string{
                {
                    "/var/vm/swapfile0",
                    "file",
                    readableSize(usedB + freeB),
                    readableSize(usedB),
                    "-1",
                },
            })
        }

        return json.Marshal(parseCommandTable(data, 1, 0))
    })

    m.Get("/sh/speed.php", func () ([]byte, error) {
        speed, err := downloadSpeed("http://cachefly.cachefly.net/10mb.test")

        if err != nil {
            return nil, err
        }

        data := make(map[string]float64)

        data["downstream"] = speed
        data["upstream"] = 0.0

        return json.Marshal(data)
    })

    m.Get("/sh/uptime.php", func () ([]byte, error) {
        uptime, err := probe.Uptime()

        if err != nil {
            return nil, err
        }

        return json.Marshal(formatUptime(uptime))
    })

    // Serve static files
    m.Get("/.*", martini.Static(""))

    m.Run()
}

func parseCommandTable(rawOutput []byte, headerCount int, footerCount int) [][]string {
    // Convert output to a string (it's not binary data, so this is ok)
    output := string(rawOutput[:])

    // We'll add all the parsed lines here
    var entries [][]string

    // Lines of output
    lines := strings.Split(output, "\n")

    // Skip first and last line of output
    for _, str := range lines[headerCount:len(lines)-footerCount] {
        entries = append(entries, strings.Fields(str))
    }

    return entries
}

func parseCommandMap(output string) map[string]string {
    data := make(map[string]string)

    // Lines of output
    lines := strings.Split(output, "\n")

    for _, line := range lines {
        parts := strings.Split(line, ":")

        // Bad line, skip it
        if len(parts) != 2 {
            continue
        }

        key := strings.TrimSpace(parts[0])
        value := strings.TrimSpace(parts[1])

        // Remove trailing dot if exists
        if value[len(value) - 1] == '.' {
            value = value[:len(value) - 1]
        }

        data[key] = value
    }

    return data
}

func formatUptime(uptime int64) string {
    return time.Duration(uptime * 1000000000).String()
}

// Get average pingTime to a host
func pingHost(hostname string, pingCount int) (string, error) {
    raw, err := exec.Command("ping", "-c", strconv.Itoa(pingCount), hostname).Output()

    if err != nil {
        return "", err
    }

    lines := strings.Split(string(raw[:]), "\n")

    if len(lines) < 2 {
        return "", errors.New("Bad output for ping command")
    }

    // Get 2nd last line
    lastLine := lines[len(lines)-2]

    // Extract average ping time as a string
    pingTime := strings.Split(strings.Split(lastLine, "=")[1], "/")[1]

    return pingTime, nil
}

func toMB(x string) int {
    num, _ := strconv.ParseInt(x, 10, 32)

    // 256 = 4096 / (1024 * 1024)
    return int(num) / 256
}


func mbToB(x int) int {
    return x * 1024 * 1024
}

// output frstring "top" command on OS X
func topStats() (map[string]string, error) {
    raw, err := exec.Command("top", "-S", "-l", "1", "-n", "0").Output()

    if err != nil {
        return nil, err
    }

    return parseCommandMap(string(raw[:])), nil
}

func readableSize(x int) string {
    KB := 1024
    MB := KB * KB
    GB := KB * KB * KB

    // Bytes
    if x < KB {
        return strconv.Itoa(x) + "b"
    } else if x < MB {
        return strconv.Itoa(x/KB) + "kb"
    } else if x < GB {
        return strconv.Itoa(x/MB) + "mb"
    }
    // else
    return strconv.Itoa(x/GB) + "gb"
}

// Download speed in kb/s
func downloadSpeed(url string) (float64, error) {
    resp, err := http.Get(url)

    if err != nil {
        return 0, err
    }

    total := 0

    // Read by 4k chunks
    buffer := make([]byte, 4096)

    // Start
    t1 := time.Now().UnixNano()

    for {
        buffer = buffer[:cap(buffer)]
        n, err := resp.Body.Read(buffer)

        total += n

        if err != nil {
            // EOF, that's good, now exit
            if err == io.EOF {
                break
            }

            return 0.0, err
        }

    }

    t2 := time.Now().UnixNano()

    // Time in seconds
    duration := float64(t2 - t1) / 1000000000.0

    // Actual download speed in kb/s
    speed := float64(total) / duration

    log.Println("Total :", total)
    log.Println("Duration :", duration)

    return speed, nil
}
