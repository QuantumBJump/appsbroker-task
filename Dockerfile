FROM golang:1.16 as build

WORKDIR hello-world

# Install dependencies in go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy rest of app source code
COPY . ./
RUN go build -mod=readonly -v -o /app

# Now create separate deployment image
FROM gcr.io/distroless/base

WORKDIR /hello-world
COPY --from=build /app .
COPY /template ./template
ENTRYPOINT ["./app"]