FROM golang:1.22.6-alpine

WORKDIR /app

# Install build dependencies and curl for healthcheck
RUN apk add --no-cache gcc musl-dev curl

# Copy go mod files
# COPY go.mod go.sum ./
COPY go.mod  ./
# Configure git for private repos
RUN apk add --no-cache git
ENV GOPRIVATE="github.com/0xElder/*"

# Configure git to use access token (passed as build arg)
ARG GITHUB_ACCESS_TOKEN
RUN git config --global url."https://${GITHUB_ACCESS_TOKEN}@github.com/".insteadOf "https://github.com/"

# Download dependencies
# RUN go mod download && go mod tidy
RUN  go mod download
RUN go mod download github.com/0xElder/elder

# Copy source code
COPY . .

# Build the application
RUN go build -o elder-wrap

# Run the application
CMD ["./elder-wrap"] 