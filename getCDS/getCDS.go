package main

import (
	"net"
	"sync"
	"database/sql"
	_ "github.com/lib/pq"
	"fmt"
	"flag"
	"github.com/howeyc/gopass"
	"time"
	"strings"
	"github.com/niclabs/Observatorio/dbController"
	"strconv"
	"github.com/miekg/dns"
	"runtime"
	"os"
	"bufio"
	"encoding/csv"
	"github.com/niclabs/Observatorio/utils"
)

//var Concurrency = 100;
var dontProbeListFile string;
var dontProbeList []*net.IPNet;

var ErrorsCount int
var TimeoutsCount int
var TimeoutsRetryCount int
var TotalTime int
var mutexT *sync.Mutex
var mutexTT *sync.Mutex
var mutexE *sync.Mutex
var mutexR *sync.Mutex
var debug = false
var err error;

var resultsFolder = "results"
var fo *os.File

var Drop=false;


func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
func CollectData(db *sql.DB, input_file string, run_id int, debug_var bool, concurrency int ){


	debug=debug_var
	t:=time.Now()
	fmt.Println("input file: ",input_file)
	writeToResultsFile("input file: "+input_file)

	lines, err := readLines(input_file)
	if(err!=nil){
		fmt.Println(err.Error())
	}
	config, _ := dns.ClientConfigFromFile("/etc/resolv.conf")
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println("num CPU:",runtime.NumCPU())
	writeToResultsFile("num CPU: "+strconv.Itoa(runtime.NumCPU()))
	mutexR = &sync.Mutex{}
	mutexT = &sync.Mutex{}
	mutexE = &sync.Mutex{}
	mutexTT = &sync.Mutex{}
	ErrorsCount = 0
	TimeoutsCount = 0
	TimeoutsRetryCount = 0;
	TotalTime = 0
	getDataQueue := make(chan string, concurrency)
	wg := sync.WaitGroup{}
	wg.Add(concurrency)
	/*Init n routines to read the queue*/
	for i := 0; i < concurrency; i++ {
		go func(run_id int) {
			j:=0
			totalTime:=0
			for line := range getDataQueue {
				t2:=time.Now()
				//fmt.Println(line)
				getCDSInfo(line, run_id, config, db)
				duration := time.Since(t2)
				mutexTT.Lock()
				totalTime += int(duration)
				mutexTT.Unlock()
				j++
			}
			wg.Done()
		}(run_id)
	}
	/*fill the queue with data to obtain*/
	for _, line := range lines {
		line := dns.Fqdn(line)
		getDataQueue <- line
	}
	/*Close the queue*/
	close(getDataQueue)
	/*wait for routines to finish*/
	wg.Wait()
	TotalTime = (int)(time.Since(t).Nanoseconds())
	dbController.SaveCorrectRun(run_id, TotalTime, true, db)
	fmt.Println("Successful Run. run_id:", run_id)
	writeToResultsFile("Successful Run. run_id: "+strconv.Itoa(run_id))
	db.Close()
}
func initResultsFile(){
	var err error;
	f:= "2006-01-02T15:04:05"
	ts := time.Now().Format(f)

	if _, err := os.Stat(resultsFolder); os.IsNotExist(err) {
		os.Mkdir(resultsFolder, os.ModePerm)
	}


	fo, err = os.Create(resultsFolder+"/CDS-"+ts+".txt")
	if err != nil {
		fmt.Println(err.Error())
	}
	// close fo on exit and check for its returned error
}
func writeToResultsFile(s string){
	if _, err := fo.WriteString(s+"\n"); err != nil {
		fmt.Println("error escribiendo en output",err.Error())
	}
}

var csvWriter *csv.Writer

func closeResultsFile(){
	fo.Close()
}
var csvFile os.File
func initCSV(){

	f:= "2006-01-02T15:04:05"
	ts := time.Now().Format(f)

	filename := "results-"+ts+".csv"
	folder:="CDS_csvs"

	csvFile, err := utils.CreateFile(folder,filename)
	if err != nil {
		fmt.Println(err.Error())
	}
	csvWriter = csv.NewWriter(csvFile)
}


func closeCSV(){
	csvWriter.Flush()
	csvFile.Close()
}



func getCDSInfo(domain_name string, run_id int, config *dns.ClientConfig, db *sql.DB) {
	c:=new(dns.Client)


	msg := new(dns.Msg)
	msg.SetQuestion(domain_name, dns.TypeCDS)
	records , _ , error := c.Exchange(msg,config.Servers[0]+":53")
	if(error!=nil){
		fmt.Println(error)
	} else {
		for _, record := range records.Answer {
			if _, ok := record.(*dns.CDS); ok {
				dt := record.(*dns.CDS).DigestType
				dg := record.(*dns.CDS).Digest
				kt := record.(*dns.CDS).KeyTag
				al := record.(*dns.CDS).Algorithm
				r := []string{domain_name, strconv.Itoa(int(kt)), strconv.Itoa(int(al)), strconv.Itoa(int(dt)), dg}
				err = csvWriter.Write(r)
				if(err!=nil){
					fmt.Println(err)
				}
				fmt.Println(r)
				//(dns.CDS)(record).DigestType
				//(dns.CDS)(record).KeyTag
				//(dns.CDS)(record).Algorithm
				//writeToResultsFile(record.String())
			}
		}
	}
}

func collectCDS(inputfile string, concurrency int, ccmax int, max_retry int, dropdatabase bool, database string, user string, password string, debug bool){

	Drop=dropdatabase
	var retry int = 0 //initial retry
	db, err := sql.Open("postgres", "user="+user+" password="+password+" dbname="+database+" sslmode=disable")
	if err != nil {
		fmt.Println(err)
		return
	}
	CreateTables(db);
	db.Close()

	for concurrency <= ccmax{
		for retry < max_retry {

			db, err := sql.Open("postgres", "user="+user+" password="+password+" dbname="+database+" sslmode=disable")
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("EXECUTING WITH ",concurrency, " GOROUTINES; retry: ",retry)
			writeToResultsFile("EXECUTING WITH "+strconv.Itoa(concurrency)+ " GOROUTINES; retry: "+strconv.Itoa(retry))

			/*Initialize*/
			//InitGeoIP()
			run_id := NewRun(db)
			/*Collect data*/
			CollectData(db, inputfile, run_id, debug, concurrency)

			ec:=ErrorsCount
			tc:=TimeoutsCount
			trc:=TimeoutsRetryCount
			tt:=TotalTime
			fmt.Println("TotalTime(nsec):", tt ," (sec) ", tt/1000000000," (min:sec) ", tt/60000000000,":",tt%60000000000/1000000000)
			writeToResultsFile("TotalTime(nsec):"+strconv.Itoa(tt) +" (sec) "+strconv.Itoa( tt/1000000000)+" (min:sec) "+strconv.Itoa( tt/60000000000)+":"+strconv.Itoa(tt%60000000000/1000000000))
			var line string;
			line = string(strconv.Itoa(run_id) + ", "+ strconv.Itoa(concurrency) + ", " + strconv.Itoa(retry)+ ", " + strconv.Itoa(ec) + ", " + strconv.Itoa(tc) + ", " +strconv.Itoa(trc) + ", " + strconv.Itoa(tt))
			fmt.Println(line)
			db.Close()
			retry ++
		}
		concurrency++
		retry=0
	}

}

func main(){

	inputFile, Concurrency, ccmax, maxRetry, dropDatabase, database, user, password, debug := readArguments()

	initResultsFile()
	initCSV()

	collectCDS(*inputFile, *Concurrency, ccmax, *maxRetry, *dropDatabase, *database, *user, password, *debug)

	closeResultsFile()
	closeCSV()

}

func readArguments()(inputfile *string, concurrency *int, ccmax int, max_retry *int, dropdatabase *bool, db *string, u *string, pass string, debug *bool){
	inputfile = flag.String("i", "", "Input file with domains to analize")
	concurrency = flag.Int("c", 50, "Concurrency: how many routines")
	cmax := flag.Int("cmax", -1, "max Concurrency: how many routines")
	max_retry = flag.Int("retry", 1, "retry:how many times")
	dropdatabase = flag.Bool("drop", false, "true if want to drop database")
	p := flag.Bool("p", false, "Prompt for password?")
	u = flag.String("u","","Database User")
	db = flag.String("db","","Database Name")
	pw := flag.String("pw","", "Database Password")
	debug = flag.Bool("d",false,"Debug flag")
	flag.Parse()
	pass=""
	if (*p) {
		fmt.Printf("Password: ")
		// Silent. For printing *'s use gopass.GetPasswdMasked()
		pwd, err := gopass.GetPasswdMasked()
		if err != nil {
			fmt.Println(err.Error())
		}
		pass=string(pwd)

	}else{
		pass=*pw
	}
	ccmax=*cmax
	if(ccmax==-1){
		ccmax=*concurrency
	}
	return
}

func CreateTables(db *sql.DB) {

	DropTable("runs_cds", db)
	_, err := db.Exec("CREATE TABLE  IF NOT EXISTS runs ( id SERIAL PRIMARY KEY, tstmp timestamp, correct_run bool, duration int)")
	if err != nil {
		fmt.Println("OpenConnections", db.Stats())
		panic(err)
	}

	DropTable("cds", db)
	// id | run_id | domain_name | int field_1 | int field_2 |  int field_3 | varchar() field_4 |  field_5 |
	_,err = db.Exec("CREATE TABLE  IF NOT EXISTS domain (id SERIAL PRIMARY KEY, run_id integer REFERENCES runs(id), domain_name varchar(253), field_1 int, field_2 int, field_3 int, field_4 varchar(253), field_5 varchar(253))")
	if err != nil {
		fmt.Println("OpenConnections",db.Stats())
		panic(err)
	}
}

func SaveCDS(line string, field_1 int, field_2 int, field_3 int, field_4 string, field_5 string, run_id int, db *sql.DB)(int){
	var cds_id int;
	err := db.QueryRow("INSERT INTO cds(domain_name, field_1, field_2, field_3, field_4, field_5, run_id) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id", line,run_id).Scan(&cds_id)
	if err != nil {
		fmt.Println("OpenConnections",db.Stats(),"domain name",line)
		if(strings.Contains(err.Error(),"too many open files")) {
			return SaveCDS(line, field_1, field_2, field_3, field_4, field_5, run_id, db)
		}
		panic(err)
	}
	return cds_id
}



func DropTable(table string, db *sql.DB) {
	if (Drop) {
		_, err := db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE")
		if err != nil {
			fmt.Println("OpenConnections", db.Stats())
			panic(err)
		}
	}
}

func NewRun(db *sql.DB)(int){
	var run_id int;
	err := db.QueryRow("INSERT INTO runs(tstmp) VALUES($1) RETURNING id", time.Now()).Scan(&run_id)
	if err != nil {
		fmt.Println("OpenConnections",db.Stats())
		panic(err)
	}
	return run_id
}

func SaveCorrectRun(run_id int, duration int, correct bool, db *sql.DB){
	_,err := db.Exec("UPDATE runs SET duration = $1, correct_run = $2 WHERE id = $3", int(duration/1000000000), correct, run_id)
	if err != nil {
		fmt.Println("OpenConnections",db.Stats()," run_id",run_id)
		panic(err)
	}
}