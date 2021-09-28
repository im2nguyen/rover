# Prep base stage
ARG TF_VERSION=light

# Build rover
FROM golang:1.17 AS rover
WORKDIR /src
# copy source
COPY . .
# build rover
RUN go get -d -v golang.org/x/net/html  
RUN CGO_ENABLED=0 GOOS=linux go build -o rover .

# Build ui
FROM node:16-alpine as ui
WORKDIR /src
# copy source
COPY . .
# change directory to ui source
WORKDIR /src/ui
# install node packages
RUN npm set progress=false && npm config set depth 0
RUN npm install --only=production 
# install ALL node_modules, including 'devDependencies'
RUN npm install
# build (to dist folder)
RUN npm run build


FROM hashicorp/terraform:$TF_VERSION AS release

# copy terraform binary
RUN cp /bin/terraform /usr/local/bin/terraform

# copy rover binary
COPY --from=rover /src/rover /bin/rover
RUN chmod +x /bin/rover

# copy ui
WORKDIR /src
COPY --from=ui /src/ui/dist ./src/ui/dist

ENTRYPOINT [ "/bin/rover" ]
