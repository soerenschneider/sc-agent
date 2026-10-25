# keep in sync with the toolchain in go.mod, otherwise the Makefile downloads the pinned toolchain during the build
FROM golang:1.27.3 AS build

ARG VERSION=dev
ARG COMMIT_HASH=unknown

WORKDIR /src
COPY ./go.mod ./go.sum ./
RUN go mod download

COPY ./ ./
# use the same build flags as release binaries built with the Makefile
RUN make compile VERSION="${VERSION}" COMMIT_HASH="${COMMIT_HASH}" OUTPUT=/sc-agent


FROM gcr.io/distroless/static AS final

LABEL maintainer="soerenschneider"
USER nonroot:nonroot
COPY --from=build --chown=nonroot:nonroot /sc-agent /sc-agent

ENTRYPOINT ["/sc-agent"]
