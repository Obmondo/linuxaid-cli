# Scripts that can help fix alerts

1. Each script's name contains (or should contain) the name of the alert (with words separated by underscores).

2. Run each script like so (once you are inside `...go-scripts/cmd/alerts`):

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

3. Comments in each script should explain more about when that script should be run and what the script needs to be run.
