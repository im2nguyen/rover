ARG TF_VERSION=light

FROM hashicorp/terraform:$TF_VERSION

RUN cp /bin/terraform /usr/local/bin/terraform

COPY ./rover /bin/rover
RUN chmod +x /bin/rover

WORKDIR /src

ENTRYPOINT [ "/bin/rover" ]
