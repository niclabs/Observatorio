#!/bin/bash
# Get License Key from config file (must get maxmind licence key to use this software, and save it in 'license_key_geoip.txt' file)

geoip_found=false;
while read line; do
	if $geoip_found; then
		case "$line" in 
  			*geoippath:*)
    	geoip_path=${line#*:%%+([[:space:]])} 
    	echo "$geoip_path"
    	esac
    	case "$line" in 
  			*geoipasnfilename:*)
			geoip_asnfilename=${line#*:}
      geoip_asnfilename=${geoip_asnfilename//[[:space:]]}
    	echo "$geoip_asnfilename" # Do stuff
    	esac
    	case "$line" in 
  			*geoipcountryfilename:*)
			geoip_countryfilename=${line#*:}
      geoip_countryfilename=${geoip_countryfilename//[[:space:]]}
    	echo "$geoip_countryfilename" # Do stuff
    	esac
	fi

	case "$line" in 
  		geoip:*)
    	geoip_found=true;	
    	;;
	esac
done < config.yml




YOUR_LICENSE_KEY=`cat license_key_geoip.txt`

if [ $? -ne 0 ]; then
	echo "Error: create the 'license_key_geoip.txt' file into main 'Observatorio' folder and add your geoip licensce key into it"
	exit
fi
key_length=${#YOUR_LICENSE_KEY}
if [ "$key_length" == "0" ]; then
	echo "Error: Empty license key. Paste your geoip licensce key into 'license_key_geoip.txt' file"
	exit
fi
case "$YOUR_LICENSE_KEY" in 
  	*"Delete this text and paste your geoip license Key here."*)
	echo "Error: Paste your geoip licensce key into 'license_key_geoip.txt' file"
	exit
esac

# Download ASN database
#wget "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-ASN&license_key="$YOUR_LICENSE_KEY"&suffix=tar.gz" -O "geoip_ASN.tar.gz"
# Download Country database
#wget "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key="$YOUR_LICENSE_KEY"&suffix=tar.gz" -O "geoip_Country.tar.gz"

# Creating folder specified in config file
mkdir -p $geoip_path
# Auxiliar folder (To be removed later)
mkdir -p Geolite2 

# Extract databases from tar.gz
ASN_output="$(tar -xzvf geoip_ASN.tar.gz -C Geolite2/ --wildcards GeoLite2-ASN_*/GeoLite2-ASN.mmdb)"
if [ $? -ne 0 ]; then
  echo "Error extracting geoip_ASN database"
  exit
fi
Country_output="$(tar -xzvf geoip_Country.tar.gz -C Geolite2/ --wildcards GeoLite2-Country_*/GeoLite2-Country.mmdb)"
if [ $? -ne 0 ]; then
  echo "Error extracting geoip_Country database"
  exit
fi

# Move databases to folder specified in config file(?)
mv Geolite2/$ASN_output Geolite/"$geoip_asnfilename"
if [ $? -ne 0 ]; then
echo "Error moving geoip_ASN database"
exit
fi
mv Geolite2/$Country_output Geolite/"$geoip_countryfilename"
if [ $? -ne 0 ]; then
echo "Error moving geoip_ASN database"
exit
fi

echo Databases updated!!

#remove unused auxiliar folders
rm -r Geolite2

#remove tar.gz??
#rm geoip_Country.tar.gz geoip_ASN.tar.gz