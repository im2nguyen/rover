ARG TF_VERSION=light

FROM hashicorp/terraform:$TF_VERSION

RUN cp /bin/terraform /usr/local/bin/terraform

# Add azure-cli
RUN apk update  && \
  apk add --no-cache bash py-pip make && \
  apk add --no-cache --virtual=build gcc libffi-dev musl-dev openssl-dev python3-dev && \
  pip install azure-cli && \
  apk del --purge build

COPY ./rover /bin/rover
RUN chmod +x /bin/rover

WORKDIR /src

ENTRYPOINT [ "/bin/rover" ]
