package utils

import (
    "fmt"
    "os"
)

func Fail(err error, msg string) {
    if err != nil {
        fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
        os.Exit(1)
    }
}
