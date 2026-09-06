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
  // GHCR links a package to its repo from the manifest ANNOTATION, not the
  // config label -- the label alone leaves the package orphaned. "index,manifest"
  // puts it on both the index and each platform manifest.
  annotations = [
    "index,manifest:org.opencontainers.image.source=https://github.com/greyrock-labs/cert-manager-webhook-cloudns"
  ]
}

target "image-local" {
  inherits = ["image"]
  output = ["type=docker"]
}

target "image-all" {
  inherits = ["image"]
  // amd64 is the only architecture we target, by choice. The Forgejo runner
  // also cannot mount binfmt_misc, so QEMU emulation is unavailable there
  // regardless -- do not add docker/setup-qemu-action back.
  platforms = [
    "linux/amd64"
  ]
}
