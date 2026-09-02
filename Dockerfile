# syntax = docker/dockerfile:1.22
########################################

FROM --platform=${BUILDPLATFORM} golang:1.26.6-alpine AS builder
RUN apk update && apk add --no-cache make
ENV GO111MODULE=on
WORKDIR /src

COPY ["go.mod", "go.sum", "/src/"]
RUN go mod download && go mod verify

COPY . .
ARG VERSION
ARG TAG
ARG SHA
RUN make build-all-archs

########################################

FROM --platform=${TARGETARCH} scratch AS proxmox-mcp
LABEL org.opencontainers.image.source="https://github.com/sergelogvinov/proxmox-mcp" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.description="Opinionated MCP server for Proxmox"

COPY --from=gcr.io/distroless/static-debian13:nonroot . .
ARG TARGETARCH
COPY --from=builder /src/bin/proxmox-mcp-${TARGETARCH} /bin/proxmox-mcp

ENTRYPOINT ["/bin/proxmox-mcp"]

########################################

FROM --platform=${TARGETARCH} scratch AS release

COPY --from=gcr.io/distroless/static-debian13:nonroot . .
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/bin/proxmox-mcp /bin/proxmox-mcp

ENTRYPOINT ["/bin/proxmox-mcp"]
