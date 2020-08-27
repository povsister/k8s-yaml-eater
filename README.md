# k8s-yamleater
Read and de-serialize multiple YAML definition from a singe source for Kubernetes

# What's this ?
Do you have a headache on programmatically "kubectl apply" pre-defined
K8S resources from YAML files in your go code ?

This simple tool could help you develop your own "kubectl apply" in your app.

It helps you

* Read a single&completed YAML document from multiple definitions in the same source
* Unmarshall YAML definition to known Kubernetes struct `runtime.Object`
* Support typed resources and CRDs


# TODO
WIP
