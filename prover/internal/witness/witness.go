package witness

// BuildExample returns a simple witness map compatible with the example circuit.
// You can extend this to return gnark witness objects as you implement the pipeline.
func BuildExample() (map[string]interface{}, error) {
	w := map[string]interface{}{
		"A": 2,
		"B": 3,
	}
	return w, nil
}
