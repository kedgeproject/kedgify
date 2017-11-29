package core

// These operations are run on every input Kubernetes definitions that is passed
// to kedgify
func CoreOperations(data []byte) ([]string, error) {

	// Check that only one controller is passed

	// Step 1 - Get the Kind from the supplied files, and see if we support the
	// provided Kind.

	// Step 2 - Unmarshal the data to the Kubernetes object of that Kind

	// Step 3 - Call object.kedgify() on the objects from Step 2, which will
	// return the Kedge representation of the given Kind

	// Step 4 - Apply Kedge shortcuts for the generated file

	// Step 5 - Prune the generated Kedge artifacts for the non-declarative
	// application definition parts

	return nil, nil
}
