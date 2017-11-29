package core

import "github.com/pkg/errors"

func GenerateKedge(paths []string) error {

	files, err := getAllYAMLFiles(paths)
	if err != nil {
		return errors.Wrap(err, "unable to get YAML files")
	}

	inputs, err := getResourcesFromFiles(files)
	if err != nil {
		return errors.Wrap(err, "unable to get kedge definitions from input files")
	}

	for _, input := range inputs {
		_, err := CoreOperations(input.data)
		if err != nil {
			return errors.Wrap(err, "unable to perform core operations")
		}
	}

	// Collate all the objects to form a Kedge definition

	// Marshal

	return nil
}
