# liveness-wrapper

![Go](https://github.com/gandalfmagic/liveness-wrapper/workflows/Go/badge.svg)
[![Coverage Status](https://coveralls.io/repos/github/gandalfmagic/liveness-wrapper/badge.svg?branch=master)](https://coveralls.io/github/gandalfmagic/liveness-wrapper?branch=master)

A better way to integrate some applications inside of a Kubernetes deployment.

## How it works

`liveness-wrapper` executes a child process, and keeps it monitored, providing a set of REST endpoints to use in a [Deployment on Kubernetes](#deployment-on-kubernetes).

- `[GET] /ready`: this endpoint expose the `readiness` for the internal http server.

- `[GET] /alive`: this endpoint expose the `liveness` of the child process. The http status code provided by this endpoint will change as the state of the wrapped process changes.

- `[GET] /ping`: this endpoint can be used by the child process to actively report that it's still functioning.

## Command line usage

You can use the `-h` or `--help` flags to list the available command line options:

```bash
$ ./liveness-wrapper -h            
a tool to wrap another executable and generate the readiness and 
the liveness http endpoints needed by kubernetes.

Usage:
  liveness-wrapper [flags]

Flags:
  -c, --config string                  Path to config file (with extension)
  -h, --help                           help for liveness-wrapper
      --log-level string               Output level of logs (TRACE, DEBUG, INFO, WARN, ERROR, FATAL) (default "WARN")
      --process-args strings           Comma separated list of arguments for the child process
      --process-fail-on-stderr         Mark the child process as failed if it writes logs on stderr
      --process-hide-stderr            Hide the stderr of the child process from the logs
      --process-hide-stdout            Hide the stdout of the child process from the logs
  -p, --process-path string            Path of the child process executable
  -r, --process-restart-always         Always restart the child process when it ends
  -e, --process-restart-on-error       Restart the child process only when it fails
  -a, --server-address string          Bind address for the http server (default ":6060")
  -t, --server-ping-timeout duration   Ping endpoint timeout, use 0 to disable (default 10m0s)
  -v, --version                        Display the current version of this CLI
```

## Configuration file

`liveness-wrapper` will load the configuration from a file named `$HOME/.liveness-wrapper.yaml`. With the `-c` flag you can load a custom configuration file from any path.

Example configuration file:

```yaml
log:
  level: INFO
process:
  path: /path/to/command
  args:
  - -flag1
  - value1
  - -flag2
  - value2
  fail-on-stderr: true 
  hide-stderr: false
  hide-stdout: true 
  restart-always: false
  restart-on-error: true 
server:
  address: :6060
  ping-timeout: 10m0s
```

## Deployment on Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: liveness-wrapper
spec:
  replicas: 1
  selector:
    matchLabels:
      app: liveness-wrapper
  template:
    metadata:
      labels:
        app: liveness-wrapper
    spec:
      containers:
      - name: factmod
        image: liveness-wrapper:latest
        ports:
        - containerPort: 6060
          name: internal
          protocol: TCP
        livenessProbe:
          httpGet:
            path: /alive
            port: internal
            scheme: HTTP
        readinessProbe:
          httpGet:
            path: /ready
            port: internal
            scheme: HTTP
```
