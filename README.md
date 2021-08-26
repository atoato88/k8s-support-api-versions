# k8s-support-api-versions
Get support API versions for kubernetes cluster.

## Usage
```
go run main.go
# For getting indented JSON
# go run main.go | jq .
```
By running this, display support k8s API versions in JSON format to k8s cluster in current kubeconfig setting.


## Sample output
You can see [sample output here](sample/v1.22.0.json).

## What is the diference of `kubectl api-resources` ?
This program shows not only current preferd API versions but also deprecated API versions.
For example about `CronJob`, [see here](https://github.com/atoato88/k8s-support-api-versions/blob/main/sample/v1.22.0.json#L71-L80)
