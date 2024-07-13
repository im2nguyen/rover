# Prep base stage
ARG TF_VERSION=light

# Build ui
FROM node:20-alpine as ui
WORKDIR /src
# Copy specific package files
COPY ./ui/package-lock.json ./
COPY ./ui/package.json ./
COPY ./ui/babel.config.js ./
# Set Progress, Config and install
RUN npm set progress=false && npm config set depth 0 && npm install
# Copy source
# Copy Specific Directories
COPY ./ui/public ./public
COPY ./ui/src ./src
# build (to dist folder)
RUN NODE_OPTIONS='--openssl-legacy-provider' npm run build

# Build rover
FROM golang:1.22 AS rover
WORKDIR /src
# Copy full source
COPY . .
# Copy ui/dist from ui stage as it needs to embedded
COPY --from=ui ./src/dist ./ui/dist
# Build rover
RUN go get -d -v golang.org/x/net/html
RUN CGO_ENABLED=0 GOOS=linux go build -o rover .

# Release stage
FROM hashicorp/terraform:$TF_VERSION AS release
# Copy terraform binary to the rover's default terraform path
RUN cp /bin/terraform /usr/local/bin/terraform
# Copy rover binary
COPY --from=rover /src/rover /bin/rover
RUN chmod +x /bin/rover

# Install Google Chrome
RUN apk add chromium

WORKDIR /src

ENTRYPOINT [ "/bin/rover" ]
