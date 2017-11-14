#!/usr/bin/env bash

# install geoip c library: https://github.com/maxmind/geoip-api-c
sudo add-apt-repository ppa:maxmind/ppa
sudo apt-get update
sudo apt-get install libgeoip1 libgeoip-dev geoip-bin

go get github.com/abh/geoip

# install and build dns library
go get github.com/miekg/dns
go build github.com/miekg/dns

# install PostgreSQL library
go get github.com/lib/pq
