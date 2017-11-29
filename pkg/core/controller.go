package core

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
)

// These operations are run on every input Kubernetes definitions that is passed
// to kedgify
func CoreOperations(data []byte) (KedgifyInterface, error) {

	// Get Kubernetes and Kedge objects for given Kind
	kubernetesObject, kedgeObject, err := getKubernetesObject(data)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get resource Kind")
	}

	// Populate Kubernetes entries in Kedge object
	err = kedgeObject.Kedgify(kubernetesObject)
	if err != nil {
		return nil, errors.Wrap(err, "unable to kedgify")
	}

	// Optimize the Kedge object, like adding shortcuts
	if err := kedgeObject.Optimize(); err != nil {
		return nil, errors.Wrap(err, "failed to optimize")
	}

	// Prune the generated Kedge object of the non-declarative definition parts
	if err := kedgeObject.Prune(); err != nil {
		return nil, errors.Wrap(err, "failed to prune")
	}

	return kedgeObject, nil
}

type objectKind struct {
	Kind string `json:"kind"`
}

func getKubernetesObject(data []byte) (runtime.Object, KedgifyInterface, error) {
	var oKind objectKind
	yaml.Unmarshal(data, &oKind)

	switch strings.ToLower(oKind.Kind) {
	case "deployment":
		kubernetesDeployment := v1beta1.Deployment{}
		err := yaml.Unmarshal(data, &kubernetesDeployment)
		if err != nil {
			return nil, nil, errors.Wrap(err, "unable to unmarshal Kubernetes Deployment")
		}
		kedgeDeploymentSpec := kedgeDeploymentSpec{}
		return &kubernetesDeployment, &kedgeDeploymentSpec, nil
	default:
		return nil, nil, fmt.Errorf("unknown/invalid type: %v", oKind)
	}
}

type KedgifyInterface interface {
	Kedgify(object runtime.Object) error

	Optimize() error

	Prune() error
}
