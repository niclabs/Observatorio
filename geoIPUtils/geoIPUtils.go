package geoIPUtils

import (
	"fmt"
	"os/exec"
	"bytes"
	"os"
	"time"
	"github.com/abh/geoip"
	"strings"
	//"net"
)

func InitGeoIP()(gi_country_db *geoip.GeoIP, gi_v6_country_db *geoip.GeoIP, gi_asn_db *geoip.GeoIP, gi_asn_v6_db *geoip.GeoIP) {
	var err error

	checkDatabases()

	gi_country_db,err = getGeoIpCountryDB()
	if(err!=nil) {
		fmt.Println(err.Error())
	}

	gi_v6_country_db,err = getGeoIpv6CountryDB()
	if(err!=nil) {
		fmt.Println(err.Error())
	}

	gi_asn_db, err = getGeoIpAsnDB()
	if(err!=nil) {
		fmt.Println(err.Error())
	}

	gi_asn_v6_db, err = getGeoIpAsnv6DB()
	if(err!=nil) {
		fmt.Println(err.Error())
	}
	return gi_country_db, gi_v6_country_db, gi_asn_db, gi_asn_v6_db

}
func downloadGeoIp()(bool){
	getGeoIp := "scripts/getGeoIp.sh"
	/*
	* wget -N http://geolite.maxmind.com/download/geoip/database/GeoLiteCountry.dat.gz
	* gunzip GeoLiteCountry.dat.gz
	*/
	cmd := exec.Command("/bin/sh", getGeoIp)

	//cmd.Stdin = strings.NewReader("some input")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Stderr
	//out2:=string(out)

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
		// path/to/whatever exists
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
func getGeoIpCountryDB()(*geoip.GeoIP,error) {
	//file := "/usr/local/share/GeoIP/GeoLiteCountry.dat"
	gi, err := geoip.Open()

	if err != nil {
		fmt.Printf("Could not open GeoIP database: %s\n", err)
		return nil, err
	}
	return gi,err
}
func getGeoIpv6CountryDB()(*geoip.GeoIP,error){
	file := "/usr/share/GeoIP/GeoIPv6.dat"
	gi, err := geoip.Open(file)
	if err != nil {
		fmt.Printf("Could not open GeoIPv6 database: %s\n", err)
		return nil, err
	}
	return gi,err
}
func getGeoIpAsnDB()(*geoip.GeoIP,error){
	file := "/usr/share/GeoIP/GeoIPASNum.dat"
	gi, err := geoip.Open(file)
	if err != nil {
		fmt.Printf("Could not open GeoIPASNUM database: %s\n", err)
		return nil, err
	}
	return gi,err
}
func getGeoIpAsnv6DB()(*geoip.GeoIP,error){
	file := "/usr/share/GeoIP/GeoIPASNumv6.dat"
	gi, err := geoip.Open(file)
	if err != nil {
		fmt.Printf("Could not open GeoIPASNUMv6 database: %s\n", err)
		return nil, err
	}
	return gi,err
}




func GetIPCountry(ip string, gi_country_db *geoip.GeoIP )(country string){
	country,_ = gi_country_db.GetCountry(ip)
	return country
}

func GetIPASN(ip string, gi_asn_db *geoip.GeoIP )(asn string){
	asn,_ = gi_asn_db.GetName(ip)
	asn = strings.Split(asn," ")[0]
	return asn
}

func GetIPv6Country(ip string, gi_country_db *geoip.GeoIP )(country string){
	country,_ = gi_country_db.GetCountry_v6(ip)
	return country
}

func GetIPv6ASN(ip string, gi_asn_db *geoip.GeoIP )(asn string){
	asn,_ = gi_asn_db.GetNameV6(ip)
	asn = strings.Split(asn," ")[0]
	return asn
}




