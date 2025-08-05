FROM public.ecr.aws/docker/library/alpine

# hack for building on QC managed laptops
RUN --mount=target=/ctx \
    if [ -d /ctx/.extra-certs/ ] ; then \
        apk add --no-cache ca-certificates && \
        cp /ctx/.extra-certs/* /usr/local/share/ca-certificates/ && update-ca-certificates; \
    fi

RUN \
    set -eux && \
    arch="amd64" && \
    if [ "$(uname -m)" = "aarch64" ] ; then arch="arm64" ; fi && \
    wget -O /vegeta.tgz https://github.com/tsenart/vegeta/releases/download/v12.12.0/vegeta_12.12.0_linux_${arch}.tar.gz && \
    tar -xzf /vegeta.tgz -C /

WORKDIR /device

ENTRYPOINT [ "/vegeta" ]
CMD [ "attack", "-duration=10s", "-targets=urls", "-output=results.bin", "-key=pkey.pem", "-cert=client.pem", "-root-certs=root.crt" ]
