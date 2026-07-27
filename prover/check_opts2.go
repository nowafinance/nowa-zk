package main

import (
	"fmt"
	"reflect"
	"github.com/consensys/gnark/backend"
)

func main() {
	fmt.Printf("%v\n", reflect.TypeOf(backend.WithProverHashToFieldFunction))
}
