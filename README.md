# k8s-yamleater
Read and de-serialize multiple Kubernetes YAML definitions from a single source

# What's this ?
Do you have a headache on programmatically "kubectl apply" multiple pre-defined
K8S resources from a single YAML file in your go code ?

This simple tool could help you develop your own "kubectl apply" in your app.

* Extract a single&full K8S YAML document from multiple definitions
* Unmarshall YAML definition to known Kubernetes struct `runtime.Object`
* Support typed resources and CRDs
* Maybe Server-side apply with discovery client ?

# TODO
WIP
