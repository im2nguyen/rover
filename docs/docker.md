# Container image

Everything is dockerized and handled by [buildx bake](docker-bake.hcl) for an agnostic usage of this repo:

There are two image variants:

1. Simple: the standard image

- e.g., `docker run --rm -it -p 9000:9000 -v $(pwd):/src im2nguyen/rover`

2. Slim: rover and terraform compressed by [UPX](https://github.com/upx/upx)

  > Slim images will take little more time to build as terraform and rover both pass through upx compression

- e.g., `docker run --rm -it -p 9000:9000 -v $(pwd):/src im2nguyen/rover:slim`

> Create docker buildx builder when first time using
> ```docker buildx create --use```

```shell
git clone --depth 1 https://github.com/im2nguyen/rover.git rover
cd rover

## Create local image
docker buildx bake

## Create local slim image
docker buildx bake slim

## build multi-platform image
docker buildx bake image-all

## build multi-platform slim image
docker buildx bake image-slim-all

```

Multi-platform create container image for these platforms
   - linux
     - amd64
     - 386
     - arm64
     - arm

You can override args and tags with envs

- Args:

  - `GO_VERSION`:    Golang version

  - `NODE_VERSION`:  Node version

  - `TF_VERSION`:    Terraform version
- Tags:

   It accepts comma seprated values e.g. `export TAGS='im2nguyen/rover:latest,im2nguyen/rover:test'`.

   - `TAGS` for image

   - `TAGS_SLIM` for slim image

    Default image tags are

     - `im2nguyen/rover:edge`
     - `im2nguyen/rover:latest`
     - `im2nguyen/rover:edge-0000000`

    For slim image
     - `im2nguyen/rover:slim`
     - `im2nguyen/rover:slim-edge`
     - `im2nguyen/rover:slim-latest`
     - `im2nguyen/rover:slim-edge-0000000`

- OR you can override individual env also
    - `REPO`:     for repository
    - `VERSION`:  for version of project
    - `GIT_SHA`:  for git ref

  That will form tags like this
   - `${REPO}:latest`
   - `${REPO}:${VERSION}`
   - `${REPO}:${VERSION}-${GIT_SHA}`

   For slim
   - `${REPO}:slim`
   - `${REPO}:slim-latest`
   - `${REPO}:slim-${VERSION}`
   - `${REPO}:slim-${VERSION}-${GIT_SHA}`

## Build binary (through docker)

Binaries will be exported to `.dist` directory

```shell
## Create binary for local platform
docker buildx bake artifact

## Create binaries for all platform
docker buildx bake artifact-slim

## Create slim binaries for all platform
docker buildx bake artifact-all
```

All plateforms covers in both binary and archive format
  - **linux**:  `amd64`, `386`, `arm64`, `arm`
  - **freebsd**: `amd64, `386`, `arm64`, `arm`
  - **windows**: `amd64, `386`, `arm64`, `arm`
  - **darwin**: `amd64, `arm64`
