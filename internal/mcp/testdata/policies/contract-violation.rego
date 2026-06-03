package main

import rego.v1

deny contains msg if {
	input.kind == "Pod"
	# This field doesn't exist in Kubernetes schema
	not input.metadata.invalid_field
	msg := "Contract violation example"
}
