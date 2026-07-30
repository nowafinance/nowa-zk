package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os/exec"
)

func main() {
	cmd := exec.Command("go", "run", "cmd/cli/test_client.go")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Run()
	fmt.Println(stdout.String())
	fmt.Println(stderr.String())
}
