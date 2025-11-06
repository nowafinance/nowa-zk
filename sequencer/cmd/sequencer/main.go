package main

import (
    "fmt"

    "github.com/tannetwork/zk-sequencer/sequencer/internal/sequencer"
)

func main() {
    fmt.Println("zk-sequencer: starting up…")
    s := sequencer.New()
    fmt.Println(s.Info())
}


