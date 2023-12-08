# ImageMirrorScript

## Workflow

- CI will export the following environment variables
    1. MIRROR_IMAGE_REGISTERY 
    2. MIRROR_REGISTERY_TOKEN

- CI Will run ImageMirrorScript
- Script will upload images to KUBEAID_IMAGE_REGISTERY and MIRROR_IMAGE_REGISTERY and replace the old image urls from helm chart. 
- CI ends

## Architecture 

```mermaid
flowchart TD
    A[READ Environment variables] -->  B[check if all environment variables are present]
    B --No--> C[Crash script]
    B --Yes--> D[Change Directory to Customer helm helm charts directory Run helm template and parse the values]
    D --> I[check the image registery URL used in helm chart]
    I --> J[If registery url is not same as MIRROR_IMAGE_REGISTERY]
    J --> K[Add imageurl to listOfDockerToPull]
    K --> L[kaniko pull imageToReplace,tag imageToReplace newRegisteryImagePath and push newRegisteryImagePath]
```
