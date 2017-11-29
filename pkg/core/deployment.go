package core

import (
	"github.com/kedgeproject/kedge/pkg/spec"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
)

type kedgeDeploymentSpec spec.DeploymentSpecMod

func (kedgeDS *kedgeDeploymentSpec) Kedgify(object runtime.Object) error {

	kd, ok := object.(*v1beta1.Deployment)
	if !ok {
		return errors.New("failed to assert type")
	}

	// setting ObjectMeta and Controller
	kedgeDS.ControllerFields = spec.ControllerFields{
		Controller: "Deployment",
		ObjectMeta: kd.ObjectMeta,
	}

	kedgeDS.DeploymentSpec = kd.Spec

	// removing PodSpec from Kedge Deployment
	kedgeDS.DeploymentSpec.Template.Spec = v1.PodSpec{}
	// populating PodSpec to Kedge PodSpecMod
	kedgeDS.ControllerFields.PodSpecMod.PodSpec = kd.Spec.Template.Spec

	// removing Containers from Kedge PodSpecMod
	kedgeDS.ControllerFields.PodSpecMod.PodSpec.Containers = nil
	// populating Containers to Kedge Containers
	for _, container := range kd.Spec.Template.Spec.Containers {
		kedgeDS.ControllerFields.PodSpecMod.Containers = append(
			kedgeDS.ControllerFields.PodSpecMod.Containers,
			spec.Container{
				Container: container,
			})
	}

	// removing Init Containers from Kedge PodSpecMod
	kedgeDS.ControllerFields.PodSpecMod.PodSpec.InitContainers = nil
	// populating Containers to Kedge Init Containers
	for _, initContainer := range kd.Spec.Template.Spec.InitContainers {
		kedgeDS.ControllerFields.PodSpecMod.InitContainers = append(
			kedgeDS.ControllerFields.PodSpecMod.InitContainers,
			spec.Container{
				Container: initContainer,
			})
	}

	return nil
}

func (kedgeDS *kedgeDeploymentSpec) Optimize() error {
	return nil
}

func (kedgeDS *kedgeDeploymentSpec) Prune() error {
	return nil
}
