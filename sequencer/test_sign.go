package main
import (
	"crypto/rand"
	"fmt"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
)
func main() {
	priv, _ := eddsa.GenerateKey(rand.Reader)
	msg := []byte("hello")
	sig, err := priv.Sign(msg, mimc.NewMiMC())
	fmt.Println("sig len:", len(sig), "err:", err)
}
