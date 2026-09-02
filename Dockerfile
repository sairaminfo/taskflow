# ---- build stage ----
# Pin the toolchain for reproducible builds.
FROM golang:1.22-alpine AS build
WORKDIR /src

# Cache module downloads separately from source for faster rebuilds.
COPY go.mod ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 produces a static binary that runs on scratch/distroless.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/taskflow ./cmd/taskflow

# ---- run stage ----
# Minimal, non-root runtime image.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/taskflow /app/taskflow
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/taskflow"]
