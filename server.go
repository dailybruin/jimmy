package main

import (
    "flag"
    "html/template"
    "io/ioutil"
    "log"
    "net"
    "net/http"
    "regexp"
)

var (
    addr = flag.Bool("addr", false, "find open address and print to final-port.txt")
)

func main() {
    flag.Parse()
    http.HandleFunc("/", makeHandler(rootHandler))

    if *addr {
        l, err := net.Listen("tcp", "127.0.0.1:0")
        if err != nil {
            log.Fatal(err)
        }
        err = ioutil.WriteFile("final-port.txt", []byte(l.Addr().String()), 0644)
        if err != nil {
            log.Fatal(err)
        }
        s := &http.Server{}
        s.Serve(l)
        return
    }

    http.ListenAndServe(":8080", nil)
}
