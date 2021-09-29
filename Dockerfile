# Prep base stage
ARG TF_VERSION=light

# Build ui
FROM node:16-alpine as ui
WORKDIR /src
# Copy all package*.json files
# COPY ./ui/package*.json ./
# Copy specific package files
COPY ./ui/package-lock.json ./
COPY ./ui/package.json ./
# Set Progress, Config and install
RUN npm set progress=false && npm config set depth 0 && npm install
# Copy source
# Copy Full Directory
# COPY ./ui .
# Copy Specific Directories
COPY ./ui/public ./public
COPY ./ui/src ./src
# build (to dist folder)
RUN npm run build

# Build rover
FROM golang:1.17 AS rover
WORKDIR /src
# copy full source
# COPY . .
# copy go sources
COPY ./go.mod .
COPY ./go.sum .
COPY ./graph.go .
COPY ./main.go .
COPY ./map.go .
COPY ./rso.go .
COPY ./server.go .
COPY ./zip.go .
# copy ui/dist from ui stage as it needs to embedded
COPY --from=ui ./src/dist ./ui/dist
# build rover
RUN go get -d -v golang.org/x/net/html  
RUN CGO_ENABLED=0 GOOS=linux go build -o rover .

# Release stage
FROM hashicorp/terraform:$TF_VERSION AS release
# copy terraform binary to the rover's default terraform path
RUN cp /bin/terraform /usr/local/bin/terraform
# copy rover binary
COPY --from=rover /src/rover /bin/rover
RUN chmod +x /bin/rover
ENTRYPOINT [ "/bin/rover" ]