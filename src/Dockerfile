# stage 1 builds the program using the official Go image
FROM golang:1.27 AS build

# all following commands run inside the /app folder in the container
WORKDIR /app

# copy only the dependency files first so Docker can cache this step
COPY go.mod go.sum ./

# download Gin and any other libraries listed in go.mod
RUN go mod download

# copy the rest of the code into the container
COPY . .

# compile for Linux on x86 chips since that is what AWS Fargate runs
# CGO_ENABLED=0 makes it run on its own without extra system libraries
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server product-service.go

# stage 2 starts from a tiny x86 Linux image so the final image matches AWS
FROM --platform=linux/amd64 alpine:3.20

# work inside /app again
WORKDIR /app

# copy just the compiled program from stage 1, leaving the Go tools behind
COPY --from=build /app/server .

# note that the server listens on port 8080
EXPOSE 8080

# the command that runs when the container starts
CMD ["./server"]