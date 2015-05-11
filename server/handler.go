package main

import (
    "net/http"
    "html/template"
    "github.com/gorilla/mux"
)

func getRouter() (*mux.Router, error) {
    fs := http.FileServer(http.Dir("static"))
    r := mux.NewRouter()
    r.HandleFunc("/", indexHandler)
    r.Handle("/static/{.+}", http.StripPrefix("/static/",fs))
    // Alternatively, use PathPrefix. However, PathPrefix will show directory of files
    // r.PathPrefix("/static/").Handler(http.StripPrefix("/static/",fs))
    return r, nil
}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        /**
         ** Check for valid path here
        m := validPath.Find
        if m == nil {
            http.NotFound(w, r)
            return
        }
        fn(w, r, m[2])
        **/
        fn(w, r)
    }
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
    t, _ := template.ParseFiles("templates/index.html")
    t.Execute(w, nil)
}
