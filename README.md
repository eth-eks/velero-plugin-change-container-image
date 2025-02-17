# Velero Plugin for Container Image Updates

A Velero plugin that allows you to specify a new container image for Deployments and StatefulSets when they are restored from backup.

## Overview

When restoring applications using Velero, you might want to update the container images to a different registry or version. This plugin enables you to set the desired container image by adding an annotation to your resources before backup, while preserving the image tag if needed.

## Deploying the plugin

The plugin is available as a container image from GitHub Container Registry:
ghcr.io/eth-eks/velero-plugin-change-container-image:latest

To deploy your plugin image to a Velero server:

### Using Velero CLI
1. Make sure your image is pushed to a registry that is accessible to your cluster's nodes.
2. Run `velero plugin add <registry/image:version>`. Example with a dockerhub image: `velero plugin add velero/velero-plugin-example`.

### Using Helm
1. Make sure your image is pushed to a registry that is accessible to your cluster's nodes.
2. Add the plugin to your Velero Helm chart's `values.yaml`:

    ```yaml
    velero:
      initContainers:
        - name: velero-plugin-change-container-image
          image: ghcr.io/eth-eks/velero-plugin-change-container-image:latest
          imagePullPolicy: Always
          volumeMounts:
            - name: plugins
              mountPath: /target
    ```

## Usage

1. Add the annotation `eth-eks.velero/container-image` to your Deployment or StatefulSet resources with the desired image name (e.g., `new-registry/app`).
2. When restoring, the plugin will automatically update the container image while preserving the original tag if not specified in the annotation:

    ```bash
    velero restore create --from-backup my-backup
    ```

## Supported Resources

- Deployments
- StatefulSets

## How It Works

The plugin:
1. Intercepts resources during restore
2. Checks for the `eth-eks.velero/container-image` annotation
3. If present, updates the container image to the specified value
4. Preserves the existing image tag if the new image doesn't specify one
5. If absent, maintains the original container image

## Development

### Prerequisites

- Go 1.23 or later
- Docker

### Running Tests

```