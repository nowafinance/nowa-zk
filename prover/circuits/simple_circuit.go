package circuits

import (
	"github.com/consensys/gnark/frontend"
)

// SimpleCircuit is a tiny example circuit: A + B == 5
type SimpleCircuit struct {
	A frontend.Variable
	B frontend.Variable
}

// Define implements the gnark circuit definition
func (c *SimpleCircuit) Define(api frontend.API) error {
	sum := api.Add(c.A, c.B)
	api.AssertIsEqual(sum, 5)
	return nil
}
