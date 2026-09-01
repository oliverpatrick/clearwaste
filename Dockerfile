FROM golang:1.25-alpine AS build
WORKDIR /src
COPY masterserver/go.mod ./masterserver/
WORKDIR /src/masterserver
RUN go mod download
COPY masterserver/ .
RUN go build -o /out/world ./cmd/server && go build -o /out/account ./cmd/account

FROM alpine:3.22
WORKDIR /app
COPY --from=build /out/world /usr/local/bin/world
COPY --from=build /out/account /usr/local/bin/account
COPY content_data /app/content_data
EXPOSE 7777 8080
