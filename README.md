# Docker Build and Push to Nova AI

## Overview

This GitHub Action allows you to build and push Docker images to Nova AI's container registry. By providing the necessary authentication details and Docker image specifications, you can seamlessly integrate this action into your CI/CD pipeline.

## Inputs

- **clientId** (required): Client ID for authentication.
- **secret** (required): Secret for authentication.
- **imageName** (required): Docker image name.
- **imageTag** (required): Docker image tag.
- **artifactId** (required): Nova Artifact ID to associate with the image
- **dockerfilePath** (optional): Path to the Dockerfile.
- **dockerfile** (optional): Inline Dockerfile content.

## Usage

Here is an example of how to use the `Docker Build and Push to Nova AI` action in your workflow:

```yaml
name: 'Docker Build and Push'

on:
  workflow_dispatch:
  pull_request:
    branches:
      - development

jobs:
  docker-push:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v2

      - name: Docker Push
        id: docker_push
        uses: trynova-ai/build-and-push-action@v0.0.6
        with:
          clientId: ${{ secrets.NOVA_CLIENT_ID }}
          secret: ${{ secrets.NOVA_SECRET }}
          imageName: 'foo'
          imageTag: ${{ github.sha }}
          artifactId: artifact-group-123
          dockerfile: |
            FROM alpine
            RUN apk add --no-cache curl
            CMD ["echo", "Hello, world!"]

      - name: Get location
        shell: bash
        run: |
            echo "Location: ${{ steps.docker_push.outputs.location }}"
```

## Implementation Details

This GitHub Action runs using Docker and requires the following inputs:
- `clientId` and `secret` for authentication.
- `imageName` and `imageTag` to specify the Docker image details.
- `artifactId` to associate the image with the Nova artifact.
- Either `dockerfilePath` or `dockerfile` must be provided to build the Docker image.

The `clientId` and `secret` should be set as repository secrets in GitHub. You can add these secrets in your repository settings under `Settings > Secrets and variables > Actions`.

The action will authenticate with Nova AI's container registry, build the Docker image from the specified Dockerfile, and push the image to the registry. The location of the uploaded Docker image will be available as an output of the action.


