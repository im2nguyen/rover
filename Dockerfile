ARG TF_VERSION=light

# for UPX binary
FROM pratikimprowise/upx as upx

# for terraform binary
FROM hashicorp/terraform:$TF_VERSION AS terraform

# Build ui
FROM node:16-alpine as ui
WORKDIR /src
# Copy specific package files first
COPY ./ui/package*.json ./
# Set Progress, Config and install
RUN npm set progress=false && npm config set depth 0 && npm install
# Copy source
# Copy Specific Directories
COPY ./ui/public ./public
COPY ./ui/src ./src
# build (to dist folder)
RUN npm run build

# Build rover
FROM golang:1.17-alpine AS rover
# Copy upx
COPY --from=upx / /
WORKDIR /src
# Install certs to copy it in build image
RUN apk add --no-cache ca-certificates && update-ca-certificates
# Copy go.mod and go.sum
COPY go.* .
# Download go mods
RUN go mod download
# Copy source
COPY . .
# Copy ui/dist from ui stage as it needs to embedded
COPY --from=ui ./src/dist ./ui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w"
RUN upx -9 rover || true
# Release stage
FROM scratch as release
WORKDIR /tmp
WORKDIR /src
# Copy terraform binary to default terraform path
COPY --from=terraform /bin/terraform  /usr/local/bin/terraform
# Copy certs
COPY --from=rover     /etc/ssl/certs/ /etc/ssl/certs/
# Copy rover binary
COPY --from=rover     /src/rover      /usr/local/bin/rover
ENTRYPOINT ["/usr/local/bin/rover"]
