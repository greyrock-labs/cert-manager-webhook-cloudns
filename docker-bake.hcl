variable "DEFAULT_TAG" {
  default = "cert-manager-webhook-cloudns:local"
}

// Special target: https://github.com/docker/metadata-action#bake-definition
target "docker-metadata-action" {
  tags = ["${DEFAULT_TAG}"]
}

// Default target if none specified
group "default" {
  targets = ["image-local"]
}

target "image" {
  inherits = ["docker-metadata-action"]
  labels = {
    "org.opencontainers.image.source" = "https://github.com/greyrock-labs/cert-manager-webhook-cloudns"
  }
}

target "image-local" {
  inherits = ["image"]
  output = ["type=docker"]
}

target "image-all" {
  inherits = ["image"]
  // amd64 only: the cluster is single-arch, and the Forgejo runner cannot
  // mount binfmt_misc, so QEMU emulation is unavailable. Restoring arm64
  // means cross-compiling in the Dockerfile via TARGETARCH, not emulation.
  platforms = [
    "linux/amd64"
  ]
}
