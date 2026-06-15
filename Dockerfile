# ---- build ----
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /aewc .

# ---- run ----
FROM alpine:3
RUN apk add --no-cache ca-certificates
COPY --from=build /aewc /aewc
EXPOSE 8090
VOLUME /data
ENTRYPOINT ["/aewc"]
