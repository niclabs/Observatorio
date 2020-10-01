# Start from the latest golang base image
FROM golang:latest

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

#WORKDIR /github.com/niclabs/Observatorio/

#ADD . /github.com/niclabs/Observatorio/

# Postgres
#EXPOSE 5432
# Download all the dependencies
#RUN go get github.com/miekg/dns
#RUN go get github.com/oschwald/geoip2-golang
#RUN go get github.com/lib/pq
#RUN go get gopkg.in/yaml.v2

# Build the dns package
#RUN go build github.com/miekg/dns

#CMD ["go", "run", "main/main.go"]


WORKDIR $GOPATH/src/github.com/niclabs/Observatorio

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Build the application
#RUN go build -o main .

# Move to /dist directory as the place for resulting binary folder
#WORKDIR /dist

# Copy binary from build to main folder
#RUN cp $GOPATH/src/github.com/niclabs/Observatorio/main .

# Export necessary port
EXPOSE 5432

# Command to run when starting the container
##CMD ["/dist/main"]
CMD ["go", "run", "main/main.go"]