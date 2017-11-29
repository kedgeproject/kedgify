package core

import (
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

func GenerateKedge(paths []string) error {

	files, err := getAllYAMLFiles(paths)
	if err != nil {
		return errors.Wrap(err, "unable to get YAML files")
	}

	inputs, err := getResourcesFromFiles(files)
	if err != nil {
		return errors.Wrap(err, "unable to get kedge definitions from input files")
	}

	// TODO: Check that only one controller is passed

	var kedgeObjects []KedgifyInterface

	for _, input := range inputs {
		kedgeObject, err := CoreOperations(input.data)
		if err != nil {
			return errors.Wrap(err, "unable to perform core operations")
		}

		kedgeObjects = append(kedgeObjects, kedgeObject)
	}

	// Collate all the objects to form a Kedge definition
	collatedKedgeObject := kedgeObjects[0]

	// Marshal
	marshalled, err := yaml.Marshal(collatedKedgeObject)
	if err != nil {
		return errors.New("failed to unmarshal the final Kedge object")
	}

	if err = writeObject(marshalled); err != nil {
		return errors.Wrap(err, "unable to write the final Kedge object")
	}

	return nil
}

func writeObject(data []byte) error {
	_, err := fmt.Fprintln(os.Stdout, "---")
	if err != nil {
		return errors.Wrap(err, "could not print to STDOUT")
	}

	_, err = os.Stdout.Write(data)
	return errors.Wrap(err, "could not write to STDOUT")
}
