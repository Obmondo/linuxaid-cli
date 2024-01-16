# Use the official Go image as the base image
FROM golang

# Set the working directory inside the container
WORKDIR /app/tls-common-name-verifier

# Copy the Go application source code into the container
COPY . .

RUN go work use .

# Build the Go application
RUN go build -o tls-common-name-verifier

# Install FPM
RUN apt-get update && apt-get install -y ruby ruby-dev build-essential rpm && gem install fpm

# Create a DEB package
RUN fpm -s dir -t deb -n tls-common-name-verifier -v 1.0.0 -p tls-common-name-verifier-1.0.0.deb ./tls-common-name-verifier=/usr/local/bin/tls-common-name-verifier

# Create an RPM package
RUN fpm -s dir -t rpm -n tls-common-name-verifier -v 1.0.0 -p tls-common-name-verifier-1.0.0.rpm ./tls-common-name-verifier=/usr/local/bin/tls-common-name-verifier

# Clean up unnecessary packages and files
RUN apt-get remove -y ruby ruby-dev build-essential && apt-get autoremove -y && apt-get clean

# Set the entry point for the container
ENTRYPOINT ["/usr/local/bin/tls-common-name-verifier"]
