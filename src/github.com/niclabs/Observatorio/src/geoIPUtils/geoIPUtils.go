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


var GEOIP_path string = "./Geolite/" // "/usr/share/GeoIP"
var GEOIP_country_name string = "GeoLite2-Country.mmdb"
var GEOIP_ASN_name string = "GeoLite2-ASN.mmdb"
var Get_GeoIP_script_path string = "scripts/getGeoIp.sh"

// Initialize GEO IP databases
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
	
	cmd := exec.Command("/bin/sh", Get_GeoIP_script_path)
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

//Checks if databases exists, if exists, check if they are updated, return (bool)databases_found and (bool)databases_updated
func checkDatabases() ( bool , bool){
	file := GEOIP_path + GEOIP_country_name
	databases_found := false
	databases_updated := false
	if file_info, err := os.Stat(file); err == nil {
		databases_found = true
		if(time.Now().After(file_info.ModTime().AddDate(0,1,0))){
			fmt.Println("Bases de Dato de GeoIP NO Actualizadas")
			//TODO request to update
			/*
			got := downloadGeoIp()
			if (!got) {
				return
			}
			*/
		}else{
			databases_updated = true
		}
	}else{
		fmt.Println("no GeoIP database found")
		got := downloadGeoIp()
		fmt.Println("Attempting to Download databases")
		if (!got) {
			fmt.Println("Attempting to Download failed!!")
		}else{
			fmt.Println("Attempting to Download Succeded!!")
			databases_found = true
			databases_updated = true
		}
	}
	return databases_found, databases_updated
}

// Finds and return the Country database
func getGeoIpCountryDB()(*geoip2.Reader,error) {
	file := GEOIP_path + GEOIP_country_name
	gi, err := geoip2.Open(file)
	if err != nil {
		fmt.Printf("Could not open GeoLite2-Country database: %s\n", err)
		return nil, err
	}
	fmt.Printf("GEOLITE2 country db opened\n")
	return gi,err
}

// Finds and return the ASN database
func getGeoIpAsnDB()(*geoip2.Reader,error){
	file := GEOIP_path + GEOIP_ASN_name
	gi, err := geoip2.Open(file)
	if err != nil {
		fmt.Printf("Could not open GeoLite2-ASN database: %s\n", err)
		return nil, err
	}
	fmt.Printf("GEOLITE2 asn db opened\n")
	return gi,err
}


// Finds and returns the conuntry of the given ip
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

// Finds and returns the ASN of the given ip
func GetIPASN(ip string, gi_asn_db *geoip2.Reader )(asn string){
	ip_addr := net.ParseIP(ip)
	var asnum,_ = gi_asn_db.ASN(ip_addr)
	asn = fmt.Sprint(asnum.AutonomousSystemNumber)
	return asn
}


