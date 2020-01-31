package main
import(
	"github.com/niclabs/Observatorio/dbController"
	"database/sql"
	_ "github.com/lib/pq"
	//"github.com/miekg/dns"
	"fmt"
	"sync"
	"time"
	"log"
//	"github.com/maitegm/dnsUtils"
	"os"
	"encoding/csv"
	"strconv"
	"flag"
	"github.com/howeyc/gopass"
)
var mutexTT *sync.Mutex
var csvsFolder string = "csvs"
func main(){
	p := flag.Bool("p", false, "Prompt for password?")
	u := flag.String("u","","Database User")
	db := flag.String("db","","Database Name")
	pw := flag.String("pw","", "Database Password")
	runid :=flag.Int("runid",1, "Database run id")
	flag.Parse()

	pass:=""
	//
	if (*p) {
		fmt.Printf("Password: ")
		// Silent. For printing *'s use gopass.GetPasswdMasked()
		pwd, err := gopass.GetPasswdMasked()
		if err != nil {
			// Handle gopass.ErrInterrupted or getch() read error
		}
		pass=string(pwd)

	}else{
		pass=*pw
	}
	fmt.Printf("Analyzing Data...")
	AnalyzeData(*runid,*db,*u,pass)
}

func AnalyzeData(run_id int, dbname string, user string, password string){
	mutexTT = &sync.Mutex{}
	t:=time.Now()
	c:=30
	db, err := sql.Open("postgres", "user="+user+" password="+password+" dbname="+dbname+" sslmode=disable")
	if err != nil {
		fmt.Println(err)
		return
	}
	ts:=dbController.GetRunTimestamp(run_id,db)
	concurrency := int(c)
	domain_ids := make(chan int, concurrency)

	wg := sync.WaitGroup{}
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {//Lanzo n rutinas para que lean de la cola
		go func(t int) {
			j:=0
			totalTime:=0
			for domain_id := range domain_ids {
				t2:=time.Now()
				CheckDomainInfo(domain_id, db)
				duration := time.Since(t2)
				mutexTT.Lock()
				totalTime += int(duration)
				mutexTT.Unlock()
				j++
			}
			wg.Done()
		}(i)
	}

	//Ahora hay que llenar la cola!
	rows, err := dbController.GetDomains(run_id, db)
	defer rows.Close()
	for rows.Next() {//para cada dominio hacer lo siguiente:
		var domain_id int
		if err := rows.Scan(&domain_id); err != nil {
			log.Fatal(err)
		}
		domain_ids <- domain_id
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	close(domain_ids) //Cierro la cola
	//espero a que todos terminen
	wg.Wait()
	getGlobalStatistics(run_id, ts, db)

	TotalTime := (int)(time.Since(t).Nanoseconds())
	fmt.Println("Total Time (nsec):", TotalTime )
	fmt.Println("Total Time (min:sec):", TotalTime/60000000000,":",TotalTime%60000000000/1000000000 )

	fmt.Println("openconnections",db.Stats())
}
func CheckDomainInfo(domain_id int, db *sql.DB) {
	//CheckDispersion(domain_id,db)
	dnssec_ok := false
	ds_found, ds_ok, dnskey_found, dnskey_ok,nsec_found, nsec_ok, nsec3_found, nsec3_ok, _ := CheckDNSSEC(domain_id, db)

	if(ds_found && ds_ok && dnskey_found && dnskey_ok &&((nsec_found && nsec_ok)|| (nsec3_found && nsec3_ok))){
		dnssec_ok = true
	}
	dbController.UpdateDomainDNSSEC(domain_id, dnssec_ok, db)

}
/*global statistics*/
func getGlobalStatistics(run_id int, ts string, db *sql.DB){
	initcsvsFolder()
	saveDispersion(run_id, ts, db)
	saveDNSSEC(run_id, ts,db)
	saveCountNameserverCharacteristics(run_id, ts,db)
}
func initcsvsFolder(){
	if _, err := os.Stat(csvsFolder); os.IsNotExist(err) {
		os.Mkdir(csvsFolder, os.ModePerm)
	}
}
/*Nameserver characteristics*/
/*Dispersion*/
func saveDispersion(run_id int, ts string, db *sql.DB){
	saveCountNSPerDomain(run_id,ts,db)
	saveCountASNPerDomain(run_id ,ts,db)
	saveCountCountryPerDomain(run_id ,ts,db)
	saveCountNSCountryASNPerDomain(run_id ,ts,db)
	saveCountNSIPv4IPv6(run_id ,ts,db)
	saveCountDomainsWithCountNSIPs(run_id ,ts,db)
	saveCountDomainsWithCountNSIPExclusive(run_id ,ts,db)
}
func saveCountDomainsWithCountNSIPExclusive(run_id int, ts string,db *sql.DB){
	rows, err:=dbController.CountDomainsWithCountNSIPExclusive(run_id, db)
	if(err!=nil){
		panic(err)
	}
	defer rows.Close()
	filename:="csvs/"+strconv.Itoa(run_id)+"CountDomainsWithCountNSIPExclusive"+ts+".csv"
	file, err := os.Create(filename)
	if(err!=nil){
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	columns,err :=rows.Columns()
	if(err!=nil){
		panic(err)
	}
	writer.Write([]string{columns[0],columns[1]})
	for rows.Next() {
		var family int
		var num int
		if err := rows.Scan(&num,&family); err != nil {
			log.Fatal(err)
		}
		line:=[]string{strconv.Itoa(num),strconv.Itoa(family)}
		err := writer.Write(line)
		if(err!=nil){
			panic(err)
		}
	}
	defer writer.Flush()
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
func saveCountCountryPerDomain(run_id int, ts string, db *sql.DB){
	rows, err:=dbController.CountCountryPerDomain(run_id, db)
	if(err!=nil){
		panic(err)
	}
	defer rows.Close()
	filename:="csvs/"+strconv.Itoa(run_id)+"CountCountryPerDomain"+ts+".csv"
	file, err := os.Create(filename)
	if(err!=nil){
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	columns,err :=rows.Columns()
	if(err!=nil){
		panic(err)
	}
	writer.Write([]string{columns[0],columns[1]})
	for rows.Next() {
		var numCountries int
		var num int
		if err := rows.Scan(&numCountries,&num); err != nil {
			log.Fatal(err)
		}
		line:=[]string{strconv.Itoa(numCountries),strconv.Itoa(num)}
		err := writer.Write(line)
		if(err!=nil){
			panic(err)
		}
	}
	defer writer.Flush()
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

}
func saveCountASNPerDomain(run_id int, ts string, db *sql.DB){
	rows, err:=dbController.CountASNPerDomain(run_id, db)
	if(err!=nil){
		panic(err)
	}
	defer rows.Close()
	filename:="csvs/"+strconv.Itoa(run_id)+"CountASNPerDomain"+ts+".csv"
	file, err := os.Create(filename)
	if(err!=nil){
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	columns,err :=rows.Columns()
	if(err!=nil){
		panic(err)
	}
	writer.Write(columns)
	for rows.Next() {
		var numASN int
		var num int
		if err := rows.Scan(&numASN,&num); err != nil {
			log.Fatal(err)
		}
		line:=[]string{strconv.Itoa(numASN),strconv.Itoa(num)}
		err := writer.Write(line)
		if(err!=nil){
			panic(err)
		}
	}
	defer writer.Flush()
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

}
func saveCountNSPerDomain(run_id int, ts string, db *sql.DB){
	rows, err:=dbController.CountNSPerDomain(run_id ,db)
	if(err!=nil){
		panic(err)
	}
	defer rows.Close()
	filename:="csvs/"+strconv.Itoa(run_id)+"CountNSPerDomain"+ts+".csv"
	file, err := os.Create(filename)
	if(err!=nil){
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	columns,err :=rows.Columns()
	if(err!=nil){
		panic(err)
	}
	writer.Write(columns)
	for rows.Next() {
		var numNS int
		var num int
		if err := rows.Scan(&numNS,&num); err != nil {
			log.Fatal(err)
		}
		line:=[]string{strconv.Itoa(numNS),strconv.Itoa(num)}
		err := writer.Write(line)
		if(err!=nil){
			panic(err)
		}
	}
	defer writer.Flush()
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
func saveCountNSCountryASNPerDomain(run_id int, ts string, db *sql.DB){
	rows, err:=dbController.CountNSCountryASNPerDomain(run_id, db)
	if(err!=nil){
		panic(err)
	}
	defer rows.Close()
	filename:="csvs/"+strconv.Itoa(run_id)+"CountNSCountryASNPerDomain"+ts+".csv"
	file, err := os.Create(filename)
	if(err!=nil){
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	columns,err :=rows.Columns()
	if(err!=nil){
		panic(err)
	}
	writer.Write([]string{columns[0],columns[1],columns[2],columns[3]})
	for rows.Next() {
		var numCountries int
		var numNS int
		var numASN int
		var num int
		if err := rows.Scan(&num,&numNS, &numASN,&numCountries); err != nil {
			log.Fatal(err)
		}
		line:=[]string{strconv.Itoa(num),strconv.Itoa(numNS),strconv.Itoa(numASN),strconv.Itoa(numCountries)}
		err := writer.Write(line)
		if(err!=nil){
			panic(err)
		}
	}
	defer writer.Flush()
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
func saveCountNSIPv4IPv6(run_id int, ts string, db *sql.DB){
	rows, err:=dbController.CountDistinctNSWithIPv4(run_id, db)
	if(err!=nil){
		panic(err)
	}
	var countIPv4 int
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&countIPv4); err != nil {
			log.Fatal(err)
		}
	}
	rows, err=dbController.CountDistinctNSWithIPv6(run_id, db)
	if(err!=nil){
		panic(err)
	}
	var countIPv6 int
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&countIPv6); err != nil {
			log.Fatal(err)
		}
	}

	filename:="csvs/"+strconv.Itoa(run_id)+"CountNSIPv4IPv6"+ts+".csv"
	file, err := os.Create(filename)
	if(err!=nil){
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	err = writer.Write([]string{"countIPv4","countIPv6"})
	if(err!=nil){
		panic(err)
	}
	line:=[]string{strconv.Itoa(countIPv4),strconv.Itoa(countIPv6)}
	err = writer.Write(line)
	if(err!=nil){
		panic(err)
	}
	defer writer.Flush()
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
func saveCountDomainsWithCountNSIPs(run_id int, ts string,db *sql.DB){
	rows, err:=dbController.CountDomainsWithCountNSIp(run_id, db)
	if(err!=nil){
		panic(err)
	}
	defer rows.Close()
	filename:="csvs/"+strconv.Itoa(run_id)+"CountDomainsWithCountNSIps"+ts+".csv"
	file, err := os.Create(filename)
	if(err!=nil){
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	columns,err :=rows.Columns()
	if(err!=nil){
		panic(err)
	}
	writer.Write([]string{columns[0],columns[1],columns[2],columns[3]})
	for rows.Next() {
		var numIP int
		var numIPv6 int
		var numIPv4 int
		var num int
		if err := rows.Scan(&num,&numIP, &numIPv4,&numIPv6); err != nil {
			log.Fatal(err)
		}
		line:=[]string{strconv.Itoa(num),strconv.Itoa(numIP),strconv.Itoa(numIPv4),strconv.Itoa(numIPv6)}
		err := writer.Write(line)
		if(err!=nil){
			panic(err)
		}
	}
	defer writer.Flush()
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
func saveCountDNSSEC(run_id int, ts string,db *sql.DB){
	dnssec_fail, dnssec_ok, no_dnssec:=dbController.CountDomainsWithDNSSEC(run_id, db)
	filename:="csvs/"+strconv.Itoa(run_id)+"CountDomainsWithDNSSEC"+ts+".csv"
	file, err := os.Create(filename)
	if(err!=nil){
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	if(err!=nil){
		panic(err)
	}
	/*categoría, cantidad de dominios
	no_dnssec, 800
	dnssec_fail, 110
	dnssec_ok, 90
	*/
	writer.Write([]string{"category","domains"})
	line:=[]string{"no_dnssec",strconv.Itoa(no_dnssec)}
	err = writer.Write(line)
	if(err!=nil){
		panic(err)
	}
	line=[]string{"dnssec_fail",strconv.Itoa(dnssec_fail)}
	err = writer.Write(line)
	if(err!=nil){
		panic(err)
	}
	line=[]string{"dnssec_ok",strconv.Itoa(dnssec_ok)}
	err = writer.Write(line)
	if(err!=nil){
		panic(err)
	}
	defer writer.Flush()
}
func saveCountDNSSECerrors(run_id int, ts string,db *sql.DB){
	denial_proof, dnskey_validation, ds_validation:=dbController.CountDomainsWithDNSSECErrors(run_id, db)
	filename:="csvs/"+strconv.Itoa(run_id)+"CountDomainsWithDNSSECErrors"+ts+".csv"
	file, err := os.Create(filename)
	if(err!=nil){
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	writer.Write([]string{"failiure","domains"})
	writer.Write([]string{"Negación de Existencia",strconv.Itoa(denial_proof)})
	writer.Write([]string{"Validación de llaves", strconv.Itoa(dnskey_validation)})
	writer.Write([]string{"Validación de DS", strconv.Itoa(ds_validation)})
	defer writer.Flush()
}
func saveCountNameserverCharacteristics(run_id int, ts string,db *sql.DB){
	recursivity, no_recursivity, edns, no_edns, tcp, no_tcp, zone_transfer, no_zone_transfer, loc_query, no_loc_query :=dbController.CountNameserverCharacteristics(run_id, db)
	filename:="csvs/"+strconv.Itoa(run_id)+"CountNameserverCharacteristics"+ts+".csv"
	file, err := os.Create(filename)
	if(err!=nil){
		panic(err)
	}
	defer file.Close()
	/*categoria, cantidad de dominios "si", cantidad dominios "no"
	permite recursividad, 300, 700
	no posee EDNS, 900, 100
	no permite comunicación tcp, 500, 500
	permite transferir la zona, 100, 900
	*/
	writer := csv.NewWriter(file)
	writer.Write([]string{"category","fail", "fulfill"})
	writer.Write([]string{"Permite Recursividad",strconv.Itoa(recursivity),strconv.Itoa(no_recursivity)})
	writer.Write([]string{"EDNS activado", strconv.Itoa(no_edns),strconv.Itoa(edns)})
	writer.Write([]string{"comunicacion TCP", strconv.Itoa(no_tcp),strconv.Itoa(tcp)})
	writer.Write([]string{"Transferencia de zona TCP", strconv.Itoa(zone_transfer),strconv.Itoa(no_zone_transfer)})
	writer.Write([]string{"Respuesta a consultas LOC", strconv.Itoa(loc_query),strconv.Itoa(no_loc_query)})
	defer writer.Flush()
}
/*DNSSEC zone*/
func saveDNSSEC(run_id int, ts string,db *sql.DB){
	saveCountDNSSEC(run_id, ts,db)
	saveCountDNSSECerrors(run_id, ts,db)
}
func CheckDNSSEC(domain_id int, db *sql.DB)(bool, bool, bool, bool, bool, bool, bool, bool, bool){

	nsec_found, nsec_ok, wildcard1:= CheckNSECs(domain_id, db)
	if (nsec_found){
		dbController.UpdateDomainNSECInfo(domain_id, nsec_ok, nsec_found, wildcard1, db)
	}
	nsec3_found, nsec3_ok, wildcard2:= CheckNSEC3s(domain_id, db)

	if (nsec3_found){
		dbController.UpdateDomainNSEC3Info(domain_id, nsec3_ok, nsec3_found, wildcard2 , db)
	}
	ds_found, ds_ok := CheckDS(domain_id, db)
	dnskey_found, dnskey_ok := CheckDNSKEY(domain_id, db)
	return ds_found, ds_ok, dnskey_found, dnskey_ok,nsec_found, nsec_ok, nsec3_found, nsec3_ok, wildcard1||wildcard2
}
func CheckDNSKEY(domain_id int, db *sql.DB)(dnskey_found bool, dnskey_ok bool){
	dnskey_found, dnskey_ok = dbController.GetDNSKEYInfo(domain_id, db)
	return
}
func CheckDS(domain_id int, db *sql.DB)(ds_found bool, ds_ok bool) {
	ds_found, ds_ok = dbController.GetDSInfo(domain_id, db)
	return
}
func CheckNSECs(domain_id int, db *sql.DB)(nsec_found bool,nsec_ok bool,wildcard bool){
	_,non_existence_status, err :=dbController.GetNonExistenceStatus(domain_id, db)
	if(err!=nil){
		return false,false, false
	}
	rows, err := dbController.GetNSECsInfo(domain_id, db)
	if(err != nil){
		panic(err)
	}
	defer rows.Close()
	nsec_found=false
	nsec_ok=false
	wildcard=false
	nnsec:=0
	var nrrsig_ok,ncover, ncoverwc, niswc bool
	nrrsig_ok = true
	ncover =false
	ncoverwc = false
	niswc = false
	for rows.Next() {
		nnsec++
		nsec_found=true
		var rrsig_ok,cover, coverwc, iswc bool
		if err := rows.Scan(&rrsig_ok,&cover,&coverwc,&iswc); err != nil {
			log.Fatal(err)
		}
		nrrsig_ok=nrrsig_ok&&rrsig_ok
		ncover=ncover||cover
		ncoverwc = ncoverwc || coverwc
		niswc=niswc|| iswc
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	if(nnsec==0){
		return
	}
	nsec_found=true
	wildcard=niswc
	if(!nrrsig_ok){
		return
	}
	if(niswc &&nnsec==1 && non_existence_status==0){
		nsec_ok=true
		return
	}
	if(ncover && ncoverwc &&!niswc && nnsec==2 && non_existence_status==3){
		nsec_ok=true
		return
	}
	return
}
func CheckNSEC3s(domain_id int, db *sql.DB)(nsec3_found bool, nsec3_ok bool, wildcard bool){
	_,non_existence_status, err :=dbController.GetNonExistenceStatus(domain_id, db)
	if(err!=nil){
		return false,false, false
	}
	rows,err := dbController.GetNSEC3s(domain_id, db)
	if(err != nil){
		panic(err)
	}
	defer rows.Close()

	nnsec:=0

	nrrsigok:=true
	nmatch:=false
	ncover:=false
	ncoverwc :=false
	nwc:=false
	for rows.Next() {
		nsec3_found = true
		nnsec++
		var rrsig_ok bool
		var match bool
		var cover bool
		var coverwc bool
		var wc bool

		if err := rows.Scan(&rrsig_ok,&match,&cover,&coverwc,&wc); err != nil {
			log.Fatal(err)
		}

		nrrsigok=nrrsigok&&rrsig_ok
		nmatch=nmatch||match
		ncover=ncover||cover
		ncoverwc = ncoverwc ||coverwc
		nwc=nwc||wc
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	nsec3_ok = false
	wildcard = nwc
	if(nnsec==0){
		nsec3_found = false
		return
	}
	if(nnsec==1 && nwc && nrrsigok && non_existence_status==0){
		nsec3_ok=true
		return
	}
	if(nmatch && ncover && ncoverwc && !nwc && nrrsigok && non_existence_status==3){
		nsec3_ok=true
		return
	}
	return
}