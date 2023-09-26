
## Applications need to be installed on your local machine before setup:
- kubectl
- kops
- awscli
- kubepug


## Points to follow before running the script

- If you want to use a different kubeconfig file, please set the KUBECONFIG environment variable to the path of your kubeconfig file rather than using the local kubeconfig file in your home directory. ```export KUBECONFIG=PATH TO YOUR KUBECONFIG FILE```


- Install the respective kops version for your cluster version before running the script. You can find the kops version for your cluster version here: [Link](https://github.com/kubernetes/kops/releases)

- Setup the awscli with your aws credentials before running the script.

- Have the kubepug installed on your local machine before running the script. You can find the installation instructions here: [Link](https://github.com/rikatz/kubepug)
  If there is any deprecated api version found, it will store the output in a file called ```depreciated_apis.txt``` in the resource directory. Take suitable action to handle the depreciated apis.