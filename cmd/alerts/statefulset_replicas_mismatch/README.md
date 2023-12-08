# Scripts that can help fix alerts

## Video of sample script (click to play)

[![This is a video of running 'kube_statefulset_replicas_mismatch'](sample_alert_script_screenshot.png)](kube-statefulset-replicas-mismatch_fast.webm)

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

1. Each alert related script can be added in the `alerts` package. A single file should suffice.

2. To add a new script, create a file with the name same as the alert it seeks to help with (with words separated by underscores).

3. The file `common.go` has functions and data that's common to all scripts. This is where logic/data common to all scripts is added.

4. Add the steps specific to each script in the script's own file. Try to add only data/steps (and as less logic as possible) to each script's own file. Details about the fundamental structure of a script step are mentioned in `common.go`.

5. Each script needs to have a `build tag` at the top of the form `//go:build kube_statefulset_replicas_mismatch
`. This allows each script to have its own main function and run independent of other scripts.
