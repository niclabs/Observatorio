#Observatorio
Software desarrollado con el fin de recolectar y analizar datos de un grupo de dominios.


##Requisitos
Correr en sistema Linux (probado en Ubuntu 19.10)
Tener una instalación de Go (probado con version go1.12.10 linux/amd64)

##Instalación

- Clonar el repositorio
- Setear GOPATH y GOROOT: seguir instrucciones de la documentación de go. Setear el gopath en la ruta donde está la carpeta Obstervatorio (la carpeta de este repositorio)

- Instalar Librerías y dependencias

1. Librería DNS

>$ go get github.com/miekg/dns

>$ go build github.com/miekg/dns

2. Librería geoip2

>$ sudo apt install libgeoip1 libgeoip-dev geoip-bin

>$ go get github.com/oschwald/geoip2-golang

3. Librería postgresql

>$ go get github.com/lib/pq

4. Librería para leer archivo de configuracion yml 

>$ go get gopkg.in/yaml.v2



- Configurar postgresql

>$sudo apt-get install postgresql

>$sudo -u postgres psql postgres

>    postgres=# CREATE ROLE obslac LOGIN password 'password';

>    postgres=# CREATE DATABASE observatorio OWNER obslac;

>    postgres=# \q

>$

- Obtener datos geográficos:

>$ wget -N -i ~/Observatorio/Utils/geoip_url_list.txt  #agregar ruta a Observatorio
>$ mkdir /usr/share/GeoIP
>$ gunzip GeoIP.dat.gz
>$ mv GeoIP.dat /usr/share/GeoIP/
>$ gunzip GeoIPv6.dat.gz
>$ mv GeoIPv6.dat /usr/share/GeoIP/
>$ gunzip GeoIPASNum.dat.gz
>$ mv GeoIPASNum.dat /usr/share/GeoIP/
>$ gunzip GeoIPASNumv6.dat.gz
>$ mv GeoIPASNumv6.dat /usr/share/GeoIP/
https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-ASN&license_key=YOUR_LICENSE_KEY&suffix=tar.gz



##Modo de uso

Para comenzar la recoleccion de datos ejecutar el siguiente comando:

>\$go run $GOPATH/src/github.com/maitegm/Observatorio/dataCollector/dataCollector.go

> -i=inputfile -dp=dontprobefile -c=concurrency -pw=password -u=user -db=dbname


con los siguientes argumentos:

 > -c int: Concurrency: how many routines (default 50)

 > -cmax int: max Concurrency: how many routines (default equal concurrency)

 > -db string: Database Name

 > -dp string:         Dont probe file with network to not ask

 > -drop:         true if want to drop database

 > -i string:         Input file with domains to analize

 > -p:         Prompt for password?

 > -pw string:         Database password

 > -retry int:         retry:how many times want to excecute (default 1)

 > -u string:         Database User

Cuando la recolección termina indica el número de run_id ejecutado. También se puede consultar la tabla runs en la base de datos para ver todas las ejecuciones disponibles(las que no se hayan borrado de la base de datos con la opción -drop=true)


Para comenzar a analizar los datos y generar los archivos ejecutar el siguiente comando

>\$go run \$GOPATH/src/github.com/maitegm/Observatorio/dataAnalyzer/dataAnalyzer.go -pw=dbpassword -u=dbuser -db=dbname -runid=runid

ó

>\$go run \$GOPATH/src/github.com/maitegm/Observatorio/dataAnalyzer/dataAnalyzer.go -p=true -u=dbuser -db=dbname -runid=runid

con los siguientes argumentos:

>-db string:         Database Name

>-p:         Prompt for password?

>-pw string:         Database Password

>-runid int:         Database run id (default 1)

>-u string:         Database User







go get github.com/miekg/dns
go get github.com/lib/pq

sudo apt-get install libgeoip-dev    # or 'brew install pkg-config'
go get github.com/abh/geoip
