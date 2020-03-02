# Start from the latest golang base image
FROM golang:latest

ENV GOOS=linux\
	GOARCH=amd64

WORKDIR /github.com/niclabs/Observatorio

COPY . .

# Get dependencies
RUN apt update

RUN apt install -y libgeoip-dev 
#libgeoip1  geoip-bin

# Create database	
RUN apt-get install -y postgresql
USER postgres
RUN /etc/init.d/postgresql start &&\
	psql --command "CREATE USER obslac WITH SUPERUSER PASSWORD 'password';" &&\
    createdb -O obslac observatorio

USER root

RUN go build -o main ./src/main

CMD ["./main"]


