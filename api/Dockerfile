# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine AS builder
WORKDIR /src

# ดาวน์โหลด dependency ก่อน เพื่อให้ layer นี้ถูก cache ตอนแก้แค่ source
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY migrations/ ./migrations/

# build ทุกไบนารีแบบ static: api, migrate (และ seed ในเฟสถัดไป)
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

# distroless: ไม่มี shell ไม่มี package manager — ผิวโจมตีเล็ก; tzdata ฝังในไบนารีแล้ว (time/tzdata)
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/ /app/
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/app/api"]
