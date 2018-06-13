package main

import "fmt"

const RTTYS_VERSION_MAJOR = 2
const RTTYS_VERSION_MINOR = 0
const RTTYS_VERSION_PATCH = 0

func rttys_version() string {
    return fmt.Sprintf("%d.%d.%d", RTTYS_VERSION_MAJOR, RTTYS_VERSION_MINOR, RTTYS_VERSION_PATCH)
}
