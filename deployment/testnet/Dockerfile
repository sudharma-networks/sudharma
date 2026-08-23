FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/sudharma-rpcd ./cmd/sudharma-rpcd
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/sudharma-testnet-manifest ./cmd/sudharma-testnet-manifest

FROM debian:bookworm-slim
RUN useradd --system --uid 10001 --create-home sudharma
WORKDIR /app
COPY --from=build /out/sudharma-rpcd /usr/local/bin/sudharma-rpcd
COPY --from=build /out/sudharma-testnet-manifest /usr/local/bin/sudharma-testnet-manifest
RUN mkdir -p /var/lib/sudharma /etc/sudharma && chown -R sudharma:sudharma /var/lib/sudharma /etc/sudharma
USER sudharma
VOLUME ["/var/lib/sudharma"]
EXPOSE 28444/tcp 28545/tcp
ENTRYPOINT ["/usr/local/bin/sudharma-rpcd"]
CMD ["-config", "/etc/sudharma/node.json"]
