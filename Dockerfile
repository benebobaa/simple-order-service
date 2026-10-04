# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.27-alpine AS build

WORKDIR /src

# Cache module downloads in their own layer.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary so it can run on distroless/static.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/api /api

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
