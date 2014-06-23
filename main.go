package main

import (
    "github.com/go-martini/martini"
)


func main() {
    m := martini.Classic()

    m.Get("/.*", martini.Static(""))

    m.Run()
}
