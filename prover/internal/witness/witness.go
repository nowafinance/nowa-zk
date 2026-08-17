package witness

// This package is deprecated — witness construction now lives inline in
// cmd/prover/start.go → generateProof().
//
// The function below is retained only to avoid breaking any existing imports.
// It returns an empty map and should not be used.
func BuildExample() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
