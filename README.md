# kedgify

Brain dump:

- I think that the conversion code for, Kedge to Kubernetes (kedge) and Kubernetes to Kedge (kedgify) can exist independently.

- Kedgify should be made to used as a library mostly, instead of a CLI tool like Kedge.

- The structure should be a lot similar to Kedge. The following operations are at core of Kedge, and for Kedgify, maybe we can lost `Validate()` for now.


```golang 
// Every controller that Kedge supports is required to implement this interface
type ControllerInterface interface {
	// Unmarshals input YAML data to the corresponding Kedge controller spec
	Unmarshal(data []byte) error

	// Validates the unmarshalled data
	Validate() error

	// Fixes the unmarshalled data, e.g. auto population/generation of fields
	Fix() error

	// Transforms the data in Kedge spec to Kubernetes' resource objects
	Transform() ([]runtime.Object, []string, error)
}

```

- First, the Kubernetes resources should be unmarshalled into Kubernetes structs (need to make sure that we use the same vendoring mechanism that Kedge uses)

- Then we need to extract out the part that Kedge supports and marshal it to Kedge structs. We need to make sure that the remaining part, if any, is imperative in nature or conveys cluster's state that is not required for declarative definition of the application, and can be safely discarded from the output. We need to log this as and when anything is being discarded.

- To unmarshal to Kedge, the first thing that we need to detect is the controller. This needs to be imported from Kedge directly. So, the `Kind` field from Kubernetes needs to be matched with the supported controllers in Kedge. If, the controller is not supported, then a separate file should be created containing the supplied definition, and `extraResources` field should be generated appropriately.

- If multiple files are being supplied, then they should be treated as belonging to the same application (not sure about that)

- The following could make up for the core operations for Kedgify -

```golang 
type ControllerInterface interface {
  // Unmarshals input YAML data to the corresponding Kedge controller spec
  Unmarshal(data []byte) error

  // Transforms the data in Kubernetes spec to Kedge spec
  Transform() ([]runtime.Object, []string, error)

  // Optimizes the Kedge spec to use shortcuts
  Optimize() error
}

```
