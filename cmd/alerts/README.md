# Scripts that can help fix alerts

## Running alert scripts

Run each script like so (once you are inside `...go-scripts/cmd/alerts`):

```sh
go clean -cache
go build -o main -tags <script name> .
./main
```

So, to run `kube_statefulset_replicas_mismatch`, you would type this:

```sh
go clean -cache
go build -o main -tags kube_statefulset_replicas_mismatch .
./main
```

Note that each script with a main function has a build tag at the top. This allows all scripts to be in a single folder and have multiple main functions.

## Adding a new alert script

1. Comments in each script should explain more about when that script should be run and what the script needs to be run.

2. To add a new script, create a file with the name same as the alert it seeks to help with (with words separated by underscores).

3. The file `common.go` has functions and data that's common to all scripts. This is where logic/data common to all scripts must be added.

4. Add only data/logic specific to each script i the script's own file. Try to add only data (and as less logic as possible) to each script's own file.

5. Each script needs to have a build tag at the top of the form `//go:build kube_statefulset_replicas_mismatch
`. This allows each script to have its own main function and run independent of other scripts.
