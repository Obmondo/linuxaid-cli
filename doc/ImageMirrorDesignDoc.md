# ImageMirrorScript

## Workflow

- CI will export the following environment variables
    1. KUBEAID_IMAGE_REGISTERY 
    2. KUBEAID_REGISTERY_TOKEN
    3. MIRROR_IMAGE_REGISTERY 
    4. MIRROR_REGISTERY_TOKEN
    5. OLD_KUBEAID_IMAGE_REGISTERY
    6. OLD_KUBEAID_REGISTERY_TOKEN
    7. OLD_MIRROR_IMAGE_REGISTERY
    8. OLD_MIRROR_REGISTERY_TOKEN

- CI Will run ImageMirrorScript
- Script will upload images to KUBEAID_IMAGE_REGISTERY and MIRROR_IMAGE_REGISTERY and replace the old image urls from helm chart. 
- CI ends

## Architecture 

```mermaid
flowchart TD
   A[READ Environment variables] -->  B[if all environment variables are not found crash]
   B --> C[Change Directory to KUBEIAD helm charts directory Run helm template and parse the values]
   B --> D[Change Directory to Customer helm helm charts directory Run helm template and parse the values]
   C --> E[check the image registery URL used in helm chart]
   E --> F[If registery url is not same as KUBEAID_IMAGE_REGISTERY]-->G[Add imageurl to listOfDockerToPull]
   G --> H[kaniko pull imageToReplace,tag imageToReplace newRegisteryImagePath and push newRegisteryImagePath]
   D --> I[check the image registery URL used in helm chart]
   I --> J[If registery url is not same as MIRROR_IMAGE_REGISTERY]
   J --> K[Add imageurl to listOfDockerToPull]
   K --> L[kaniko pull imageToReplace,tag imageToReplace newRegisteryImagePath and push newRegisteryImagePath]
```
