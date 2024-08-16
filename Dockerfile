# First stage: Build stage
FROM golang:1.21

# Set the working directory
WORKDIR /app

# Set up SSH keys for private GitHub repository access
ARG _SSH_PRIVATE_KEY

# Private key setup
RUN mkdir -p /root/.ssh
RUN echo "${_SSH_PRIVATE_KEY}" > /root/.ssh/id_rsa
RUN chmod 600 /root/.ssh/id_rsa

# Copy the source code into the container
COPY . .

# Download dependencies
RUN ssh-keyscan -t rsa github.com > /root/.ssh/known_hosts
RUN git config --global url.ssh://git@github.com/.insteadOf https://github.com/
ENV GOPRIVATE github.com/jahagaley
RUN go mod download

# Build the application with optimized flags
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-s -w' -o /go/bin/app

# Set the default command to run the application
CMD ["/go/bin/app"]
