package main

import (
    "os"
    "os/exec"

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

        return json.Marshal(parseCommandTable(rawOutput))
    })

    m.Get("/sh/df.php", func () ([]byte, error) {
        // Run uptime command
        rawOutput, err := exec.Command("df", "-Ph").Output()

        if err != nil {
            return nil, err
        }

        return json.Marshal(parseCommandTable(rawOutput))
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

        lines := strings.Split(string(raw[:]), "\n")

        // We'll add all the parsed lines here
        var entries [][]string

        // Skip first and last line of output
        for _, str := range lines[2:len(lines)-1] {
            fields := strings.Fields(str)
            entries = append(entries, []string{
                // User
                fields[0],
                // From
                fields[2],
                // Login at
                fields[3],
                // Idle
                fields[4],
            })
        }

        return json.Marshal(entries)
    })

    // Serve static files
    m.Get("/.*", martini.Static(""))

    m.Run()
}

func parseCommandTable(rawOutput []byte) [][]string {
    // Convert output to a string (it's not binary data, so this is ok)
    output := string(rawOutput[:])

    // We'll add all the parsed lines here
    var entries [][]string

    // Lines of output
    lines := strings.Split(output, "\n")

    // Skip first and last line of output
    for _, str := range lines[1:len(lines)-1] {
        entries = append(entries, strings.Fields(str))
    }

    return entries
}