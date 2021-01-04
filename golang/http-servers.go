package main

import (
    "fmt"
    "net/http"
    "log"
)

func hello(w http.ResponseWriter, req *http.Request) {

    fmt.Fprintf(w, "Hello world!\n")
}

func main() {
    log.Print("Running on :8090")
    http.HandleFunc("/", hello)
    http.ListenAndServe(":8090", nil)
}
