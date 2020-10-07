# Observatorio
Software desarrollado con el fin de recolectar y analizar datos DNS de un conjunto de dominios.


## Utilizando Docker:

        docker cp Observatorio:Obs .

## Requisitos
#### Geolite
Para poder geolocalizar las direcciones IP es necesario obtener una llave para usar los servicios de geolite, Maxmind. puedes registrarte aquí: https://www.maxmind.com/en/geolite2/signup
esta llave debe tener el nombre
#### Go lang
- Para descargar e instalar go lang siga las instrucciones en https://golang.org/doc/install.

Asegúrese de agregar las variables de entorno $GOROOT y $GOPATH
#### Postgresql
- Tener una base de datos potgreSQL
(si se va a correr en docker, aseguarse de darle acceso al container, modificando el archivo "pg_hba.conf" de postgres según corresponda y reiniciando el servicio)
- Configurar postgresql en Windows seguir las instrucciones en https://www.postgresql.org/download/windows/. Asegúrese de crear el usuario postgres con su contraseña. Para crear la base de datos a utilizar siga los siguentes pasos:
    
        $ psql -U postgres
    
        postgres=# CREATE ROLE su_usuario LOGIN password 'su_contraseña'; //crear un usuario
    
        postgres=# CREATE DATABASE su_base_de_datos OWNER su_usuario; //crear una base de datos y asignarla al usuario creado
    
        postgres=# \q
    
        $
- Configurar postgresql en ubuntu

        $ sudo apt-get install postgresql
    
        $ sudo -u postgres psql postgres
            
        postgres=# CREATE ROLE su_usuario LOGIN password 'su_contraseña'; //crear un usuario
            
        postgres=# CREATE DATABASE su_base_de_datos OWNER su_usuario; //crear una base de datos y asignarla al usuario creado
            
        postgres=# \q
    
        $

##Instalación

- Clonar el repositorio u obtener librería usando:

           $ go get github.com/niclabs/Observatorio

- Setear GOPATH y GOROOT: seguir instrucciones de la documentación de go. Setear el gopath en la ruta donde está la carpeta Obstervatorio (la carpeta de este repositorio)

#### Instalar Librerías y dependencias

1. Librería DNS

        $ go get github.com/miekg/dns

        $ go build github.com/miekg/dns

2. Librería geoip2

        $ sudo apt install libgeoip1 libgeoip-dev geoip-bin (en caso de usar ubuntu)

        $ go get github.com/oschwald/geoip2-golang

3. Librería postgresql

        $ go get github.com/lib/pq

4. Librería para leer archivo de configuracion yml 

        $ go get gopkg.in/yaml.v2






## Modo de uso

- Llenar el archivo de configuración (config.yml) con los datos correspondientes
con los siguientes argumentos:

        #Geoip data
        #Reminder: use spaces, yaml doesn't allow tabs
        geoip:
            geoippath: Geolite                              //folder where the geolite dabases are saved
            geoipasnfilename: GeoLite2-ASN.mmdb             //name of the asn geolite database
            geoipcountryfilename: GeoLite2-Country.mmdb     //name of the country geolite database
            geoipupdatescript: UpdateGeoliteDatabases.sh    //name of the script used to update the geolite databases
        # Database configurations
        database:
            dbname: observatorio        //name of the database you created
            dbuser: obslac              //user you created
            dbpass: password            //password for the user you created
            dbhost: 172.21.128.1        //postgresql host
            dbport: 5432                //postgresql port
        #runing arguments
        runargs:
            inputfilepath: input-example.txt      //file with the list of domains you want to test
            dontprobefilepath: dontprobefile.txt  //file with the list of IPs you dont want to query
            concurrency: 100                      //desired concurrency
            ccmax: 100                            //max concurrency
            maxretry: 2                           //max attemps to retry a dns request
            debug: false        
            dnsservers: ["8.8.8.8", "1.1.1.1"]    //here put the dns servers you want to resolve the requests
        #End of config data


- Para comenzar la recoleccion de datos ejecutar el siguiente comando:

        $go run $GOPATH/src/github.com/maitegm/Observatorio/main/main.go

Esta operación puede tardar varias horas dependiendo del tamaño de la lista de dominios que se quieren analizar.


