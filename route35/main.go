package main

import (
    "os"
)

func main() {
    configFile := os.Args[1]

    config := MustParseConfig(configFile)

    MustStartDns(config)


    router := BuildWeb(config)

    router.Run(":8081")
}
