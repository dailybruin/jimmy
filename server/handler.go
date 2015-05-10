package main

import (
    "github.com/gorilla/mux"
)

func getRouter() (*mux.Router, error) {
    r := mux.NewRouter()
    return r, nil
}
