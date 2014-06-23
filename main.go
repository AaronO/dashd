package main

import (
    "os"
    "os/exec"

//    "log"

//    "fmt"
    "strings"
    "runtime"
//    "strconv" // For Itoa
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

        // Convert output to a string (it's not binary data, so this is ok)
        output := string(rawOutput[:])

        if err != nil {
            return nil, err
        }

        // We'll add all the parsed lines here
        var entries [][]string

        // Lines of output
        lines := strings.Split(output, "\n")

        // Skip first and last line of output
        for _, str := range lines[1:len(lines)-1] {

            entries = append(entries, strings.Fields(str))
        }

        return json.Marshal(entries)
    })

    // Serve static files
    m.Get("/.*", martini.Static(""))

    m.Run()
}
