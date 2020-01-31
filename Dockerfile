# Start from the latest golang base image
FROM golang:latest

# Add Maintainer Info
# LABEL maintainer="Rajeev Singh <rajeevhub@gmail.com>"

# We create an /app directory within our
# image that will hold our application source
# files
#RUN mkdir /app
# We copy everything in the root directory
# into our /app directory
#ADD . /app
# We specify that we now wish to execute 
# any further commands inside our /app
# directory
#WORKDIR /app

WORKDIR /


# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
# RUN go mod download
# Copy the source from the current directory to the Working Directory inside the container
COPY ./src .

# Add this go mod download command to pull in any dependencies
#RUN go mod download

RUN go mod tidy -v



# Build the Go app
RUN go build

# Command to run the executable
CMD ["./main"]