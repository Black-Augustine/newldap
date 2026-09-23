# 多阶段构建：前端 → Go 二进制（内嵌前端）→ 极小运行镜像
# 《技术架构设计》§6

FROM node:20-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/embed.go
COPY --from=web /src/web/dist web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/newldap ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/newldap /newldap
ENV NEWLDAP_ADDR=:8080
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/newldap"]
