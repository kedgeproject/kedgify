vendor-update:
	# Handles packages defined in glide.yaml
	glide update -v
	# Vendors OpenShift and its dependencies
	./scripts/vendor-openshift.sh

