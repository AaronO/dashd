package main

import (
    "os"
    "os/exec"

    "errors"

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
