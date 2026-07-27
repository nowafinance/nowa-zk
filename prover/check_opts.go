package main

import (
	"fmt"
	"reflect"
	"github.com/consensys/gnark/backend"
)

func main() {
	t := reflect.TypeOf(backend.WithProverHashToFieldFunction)
	fmt.Printf("backend.WithProverHashToFieldFunction exists? %v\n", t != nil)
}
