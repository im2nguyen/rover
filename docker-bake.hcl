variable "GO_VERSION" {
  default = "1.17"
}

variable "NODE_VERSION" {
  default = "16"
}

variable "TF_VERSION" {
  default = "1.1.0"
}

variable "REPO" {
  default = "im2nguyen/rover"
}

variable "VERSION" {
  default = "edge"
}

variable "GIT_SHA" {
  default = "0000000"
}

variable "TAGS" {
  default = [
    "${REPO}:latest",
    "${REPO}:${VERSION}",
    "${REPO}:${VERSION}-${GIT_SHA}"
  ]
}

variable "TAGS_SLIM" {
  default = [
    "${REPO}:slim",
    "${REPO}:slim-latest",
    "${REPO}:slim-${VERSION}",
    "${REPO}:slim-${VERSION}-${GIT_SHA}"
  ]
}


target "_common" {
  args = {
    GO_VERSION   = GO_VERSION
    NODE_VERSION = NODE_VERSION
    TF_VERSION   = TF_VERSION
  }
}

target "_labels" {
  labels = {
    "org.opencontainers.image.title"         = "actions-runner-controller",
    "org.opencontainers.image.base.name "    = "scratch",
    "org.opencontainers.image.licenses"      = "MIT",
    "org.opencontainers.image.description"   = " Interactive Terraform visualization. State and configuration explorer",
    "org.opencontainers.image.version"       = "${VERSION}",
    "org.opencontainers.image.revision"      = "${GIT_SHA}",
    "org.opencontainers.image.source"        = "https://github.com/im2nguyen/rover",
    "org.opencontainers.image.documentation" = "https://github.com/im2nguyen/rover",
  }
}

target "_slim" {
  target = "slim"
  tags   = TAGS_SLIM
}

target "_fat" {
  target = "fat"
  tags   = TAGS
}

target "artifacs" {
  output = ["./dist"]
}

target "image-platform" {
  platforms = [
    "linux/amd64",
    "linux/386",
    "linux/arm64",
    "linux/arm",
  ]
}

target "bin-platform" {
  platforms = [
    "linux/amd64",
    "linux/386",
    "linux/arm64",
    "linux/arm",
    "freebsd/amd64",
    "freebsd/386",
    "freebsd/arm64",
    "freebsd/arm",
    "windows/amd64",
    "windows/arm64",
    "windows/arm",
    "windows/386",
    "darwin/amd64",
    "darwin/arm64",
  ]
}

group "default" {
  targets = ["image-local"]
}

# Creating fat container image for local docker
target "image-local" {
  inherits = ["_common", "_fat", "_labels"]
  output   = ["type=docker"]
}

# Creating slim container image for local docker
target "image-slim" {
  inherits = ["_common", "_slim", "_labels"]
  output   = ["type=docker"]
}

# Creating fat container image for all platforms
target "image-all" {
  inherits = ["_common", "image-platform", "_fat", "_labels"]
  target   = "fat"
}

# Creating slim container image for all platforms
target "image-slim-all" {
  inherits = ["_common", "image-platform", "_slim", "_labels"]
  target   = "slim"
}

# Creating all fat artifact for all platforms
target "artifact" {
  inherits = ["_common", "artifacs"]
  target   = "artifact"
}

# Creating all slim artifact for all platforms
target "artifact-slim" {
  inherits = ["_common", "artifacs"]
  target   = "artifact-slim"
}

# Creating all full, slim artifact for all platforms
target "artifact-all" {
  inherits = ["artifact-all", "artifacs", "bin-platform", ]
  target   = "artifact-all"
}
