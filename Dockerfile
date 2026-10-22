FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/taskflow ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /bin/taskflow /taskflow
EXPOSE 8080
ENTRYPOINT ["/taskflow"]
