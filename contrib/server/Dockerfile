FROM public.ecr.aws/docker/library/golang:1.25 AS builder

# hack for building on QC managed laptops
RUN --mount=target=/ctx \
    if [ -d /ctx/.extra-certs/ ] ; then \
        cp /ctx/.extra-certs/* /usr/local/share/ca-certificates/ && update-ca-certificates; \
    fi

WORKDIR /src
COPY go.* ./
RUN go mod download

COPY . .
# go:embed silently bakes Git-LFS pointer stubs into the binary
RUN if grep -q "git-lfs.github.com/spec" server/ui/web/templates/pico_min_211.css; then \
        echo "ERROR: LFS assets are pointer stubs - the web UI would be unstyled. Run 'git lfs pull' and rebuild." >&2; \
        exit 1; \
    fi
RUN go build \
    --ldflags '-linkmode external -extldflags "-static"' \
    -o fioserver \
    github.com/foundriesio/update-server/cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=builder /src/fioserver /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/fioserver"]
