package geoIPUtils

import (
	"fmt"
	"os/exec"
	"bytes"
	"os"
	"time"
	"net"
	"github.com/oschwald/geoip2-golang"
)


var GEOIP_path string = "./Geolite/"

func InitGeoIP()(gi_country_db *geoip2.Reader, gi_asn_db *geoip2.Reader) {
	var err error

	checkDatabases()

	gi_country_db,err = getGeoIpCountryDB()
	if(err!=nil) {
		fmt.Println(err.Error())
	}


	gi_asn_db, err = getGeoIpAsnDB()
	if(err!=nil) {
		fmt.Println(err.Error())
	}


	return gi_country_db, gi_asn_db

}
func downloadGeoIp()(bool){
	getGeoIp := "scripts/getGeoIp.sh"
	cmd := exec.Command("/bin/sh", getGeoIp)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Stderr

	if err != nil {
		fmt.Println(err)
		return false
	}
	fmt.Println("Get GeoIP databases:",out.String())

	return true
}
func checkDatabases(){
	file:="/usr/share/GeoIP/GeoIP.dat"
	if file_info, err := os.Stat(file); err == nil {
		if(time.Now().After(file_info.ModTime().AddDate(0,1,0))){
			fmt.Println("Bases de Dato de GeoIP NO Actualizadas")
			/*
			got := downloadGeoIp()
			if (!got) {
				return

			}
			*/
		}
	}else{
		fmt.Println("no GeoIP database found")
		got := downloadGeoIp()
		if (!got) {
			return
		}
	}
}


func getGeoIpCountryDB()(*geoip2.Reader,error) {
	file := GEOIP_path + "GeoLite2-Country.mmdb"
	gi, err := geoip2.Open(file)

	if err != nil {
		fmt.Printf("Could not open GeoLite2-Country database: %s\n", err)
		return nil, err
	}
	fmt.Printf("GEOLITE2 country db opened\n")
	return gi,err
}

func getGeoIpAsnDB()(*geoip2.Reader,error){
	file := GEOIP_path + "GeoLite2-ASN.mmdb"
	gi, err := geoip2.Open(file)
	if err != nil {
		fmt.Printf("Could not open GeoLite2-ASN database: %s\n", err)
		return nil, err
	}
	fmt.Printf("GEOLITE2 asn db opened\n")
	return gi,err
}



func GetIPCountry(ip string, gi_country_db *geoip2.Reader )(country string){
	ip_addr := net.ParseIP(ip)
	var ctry , err = gi_country_db.Country(ip_addr)
	if(err!= nil){
		fmt.Printf("Could not get country: %s\n", err)
		return ""
	}
	country = ctry.Country.IsoCode
	return country
}

func GetIPASN(ip string, gi_asn_db *geoip2.Reader )(asn string){
	ip_addr := net.ParseIP(ip)
	var asnum,_ = gi_asn_db.ASN(ip_addr)
	asn = fmt.Sprint(asnum.AutonomousSystemNumber)
	return asn
}


