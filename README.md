# Pipelines as Code Interceptor

## Description

`pac-interceptor` is a service that uses `tkn-autogenerate` to inspect a repository and try to guess which tasks to
add and generate a pipelinerun suitable for
[Pipelines-as-Code](https://pipelinesascode.com).
It returns default PipelineRun.

The service uses the GIT API to clone the code so that it can pass required information to
`tkn-autogenerate` library.

This service facilitates a fully automated CI system by eliminating the
dependency on the .tekton folder for Pipelines as Code.

Additionally, this solution is helpful for beginners looking to learn Tekton without the need for extensive configuration.

## Installation

```shell
go install github.com/savitaashture/pac-interceptor@latest
```

## Supported SCM

GitHub SCM is the Only supported provider today.

## Usage

```shell
ko apply -f config/
```

The above command will create `pac-interceptor` deployment, service and OpenShift route.

In order to use `pac-interceptor` service in Pipelines as Code, configure the `pac-interceptor`
route URL in the `pipelines-as-code` ConfigMap as shown below:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: pipelines-as-code
data:
  pac-interceptor-url: http://pac.interceptor.url/pac-interceptor
```

Please make sure to replace http://pac.interceptor.url/pac-interceptor with the
actual URL of your pac-interceptor service.

## Copyright

[Apache-2.0](./LICENSE)

## Authors

### Savita Ashture

- Twitter - <[@savitaashture](https://twitter.com/savitaashture)>
