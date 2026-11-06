package main

apiVersion: "cuestomize.dev/v1alpha1"
kind:       "Cuestomization"

input: {...}

// null does not unify with these constraints: the model only succeeds if the includes' nulls are pruned
includes: [_]: [_]: [_]: [_]: {
	metadata: {
		labels?: [string]:      string
		annotations?: [string]: string
		...
	}
	...
}

outputs: cm: {
	apiVersion: "v1"
	kind:       "ConfigMap"
	metadata: {
		name:      "out"
		namespace: "default"
	}
}
