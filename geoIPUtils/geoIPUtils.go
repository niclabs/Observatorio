package geoIPUtils

import (
	"bytes"
	"fmt"
	"github.com/oschwald/geoip2-golang"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"
)

//var GEOIP_path string = "Geolite/" // "/usr/share/GeoIP"
//var GEOIP_country_name string = "GeoLite2-Country.mmdb"
//var GEOIP_ASN_name string = "GeoLite2-ASN.mmdb"
//var Get_GeoIP_script_path string = "UpdateGeoliteDatabases.sh"

type GeoipDB struct {
	Country_db *geoip2.Reader
	Asn_db     *geoip2.Reader
}

// Initialize GEO IP databases
func InitGeoIP(geoip_path string, geoip_country_db_name string, geoip_asn_db_name string, geoip_update_script string) *GeoipDB {
	var err error
	checkDatabases(geoip_path, geoip_country_db_name, geoip_asn_db_name, geoip_update_script)
	gi_country_db, err := getGeoIpCountryDB(geoip_path + "/" + geoip_country_db_name)
	if err != nil {
		fmt.Println(err.Error())
	}
	gi_asn_db, err := getGeoIpAsnDB(geoip_path + "/" + geoip_asn_db_name)
	if err != nil {
		fmt.Println(err.Error())
	}
	geoip_db := &GeoipDB{gi_country_db, gi_asn_db}
	return geoip_db
}

func CloseGeoIP(geoipDB *GeoipDB) {
	geoipDB.Country_db.Close()
	geoipDB.Asn_db.Close()
}

func downloadGeoIp(geoip_update_script string) bool {

	cmd := exec.Command("/bin/sh", geoip_update_script)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Stderr

	if err != nil {
		fmt.Println(err)
		return false
	}
	fmt.Println("Get GeoIP databases:", out.String())

	return true
}

//Checks if databases exists, if exists, check if they are updated, return (bool)databases_found and (bool)databases_updated
func checkDatabases(geoip_path string, geoip_country_db_name string, geoip_asn_db_name string, geoip_update_script string) (bool, bool) {
	go_again := true
	file := geoip_path + geoip_country_db_name
	databases_found := false
	databases_updated := false
checkdb:
	if file_info, err := os.Stat(file); err == nil {
		databases_found = true
		if time.Now().After(file_info.ModTime().AddDate(0, 1, 0)) {
			fmt.Println("not updated geoip databases")
		} else {
			fmt.Println("geoipDBs ok!!")
			databases_updated = true
			if go_again {
				go_again = false
				file = geoip_path + geoip_asn_db_name
				goto checkdb //now check asn db
			}
			return databases_found, databases_updated
		}
	}
	fmt.Println("Updating geoip databases")
	got := downloadGeoIp(geoip_update_script)
	fmt.Println("Attempting to Download databases")
	if !got {
		fmt.Println("Attempting to Download failed!! :( ")
	} else {
		fmt.Println("Attempting to Download Succeded!!")
		databases_found = true
		databases_updated = true
	}
	return databases_found, databases_updated
}

// Finds and return the Country database
func getGeoIpCountryDB(file string) (*geoip2.Reader, error) {
	gi, err := geoip2.Open(file)
	if err != nil {
		fmt.Printf("Could not open GeoLite2-Country database: %s\n", err)
		return nil, err
	}
	fmt.Printf("GEOLITE2 country db opened\n")
	return gi, err
}

// Finds and return the ASN database
func getGeoIpAsnDB(file string) (*geoip2.Reader, error) {
	gi, err := geoip2.Open(file)
	if err != nil {
		fmt.Printf("Could not open GeoLite2-ASN database: %s\n", err)
		return nil, err
	}
	fmt.Printf("GEOLITE2 asn db opened\n")
	return gi, err
}

// Finds and returns the conuntry of the given ip
func GetIPCountry(ip string, gi_country_db *geoip2.Reader) (country string) {
	ip_addr := net.ParseIP(ip)
	var ctry, err = gi_country_db.Country(ip_addr)
	if err != nil {
		fmt.Printf("Could not get country: %s\n", err)
		return ""
	}
	country = ctry.Country.IsoCode
	return country
}

// Finds and returns the ASN of the given ip
func GetIPASN(ip string, gi_asn_db *geoip2.Reader) (asn string) {
	ip_addr := net.ParseIP(ip)
	var asnum, _ = gi_asn_db.ASN(ip_addr)
	asn = strconv.FormatUint(uint64(asnum.AutonomousSystemNumber), 10)
	return asn
}
