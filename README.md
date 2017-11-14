#Observatorio
Software desarrollado con el fin de recolectar y analizar datos de un grupo de dominios.


##Instalación

 - Descargar observatoriolac del repositorio
 - instalar golang:

> $sudo add-apt-repository ppa:ubuntu-lxc/lxd-stable

> $sudo apt-get update

> $sudo apt-get install golang

 - Setear path Go:

Modificar el archivo ~/.profile para modificar los paths, para decirle al sistema dónde está go y cuál es el path donde encontrar los archivos escritos en go

>$sudo gedit ~/.profile

 - Agregar go al path:

>$export PATH=$PATH:/usr/local/go/bin

 - Setear el gopath al path donde se encuentra la carpeta
   ObservatorioLAC, por ejemplo:

>$export GOPATH=$HOME/ObservatorioLAC

- Instalar Librerías y dependencias
	1. Librería DNS

>$ go get github.com/miekg/dns

>$ go build github.com/miekg/dns

2. Librería geoip

>$ sudo apt install libgeoip1 libgeoip-dev geoip-bin

>$ go get github.com/abh/geoip

	3. Librería postgresql

>$ go get github.com/lib/pq

4. Librería password

>$ go get "github.com/howeyc/gopass"


- Configurar postgresql

>$sudo apt-get install postgresql

>$sudo -u postgres psql postgres

>    postgres=# CREATE ROLE obslac LOGIN password 'password';

>    postgres=# CREATE DATABASE observatorio OWNER obslac;

>    postgres=# \q

>$

- Obtener datos geográficos:

>$ wget -N -i geoip_url_list.txt

>$ mkdir usr/share/GeoIP

>$ gunzip GeoIP.dat.gz

>$ mv GeoIP.dat /usr/share/GeoIP/

>$ gunzip GeoIPv6.dat.gz

>$ mv GeoIPv6.dat /usr/share/GeoIP/

>$ gunzip GeoIPASNum.dat.gz

>$ mv GeoIPASNum.dat /usr/share/GeoIP/

>$ gunzip GeoIPASNumv6.dat.gz

>$ mv GeoIPASNumv6.dat /usr/share/GeoIP/


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

 > -i string:         Input file with domains to analyze

 > -p:         Prompt for password?

 > -pw string:         Database password

 > -retry int:         retry:how many times want to excecute (default 1)

 > -u string:         Database User



Cuando la recolección termina indica el número de run_id ejecutado. También se puede consultar la tabla runs en la base de datos para ver todas las ejecuciones disponibles(las que no se hayan borrado de la base de datos con la opción -drop=true)


Para comenzar a analizar los datos y generar los archivos ejecutar el siguiente comando

>\$go run $GOPATH/src/github.com/maitegm/Observatorio/dataAnalyzer/dataAnalyzer.go -pw=dbpassword -u=dbuser -db=dbname -runid=runid

ó

>\$go run \$GOPATH/src/github.com/maitegm/Observatorio/dataAnalyzer/dataAnalyzer.go -p=true -u=dbuser -db=dbname -runid=runid

con los siguientes argumentos:

>-db string:         Database Name

>-p:         Prompt for password?

>-pw string:         Database Password

>-runid int:         Database run id (default 1)

>-u string:         Database User


