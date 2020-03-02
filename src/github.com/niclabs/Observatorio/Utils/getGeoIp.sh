#!/bin/bash

wget -N -i geoip_url_list.txt

mkdir usr/share/GeoIP

gunzip GeoIP.dat.gz
mv GeoIP.dat /usr/share/GeoIP/

gunzip GeoIPv6.dat.gz
mv GeoIPv6.dat /usr/share/GeoIP/

gunzip GeoIPASNum.dat.gz
mv GeoIPASNum.dat /usr/share/GeoIP/

gunzip GeoIPASNumv6.dat.gz
mv GeoIPASNumv6.dat /usr/share/GeoIP/

