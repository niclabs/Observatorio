package main
import(
	"dataCollector"
	"flag"
	"github.com/howeyc/gopass"
	"fmt"
)

func main(){
	input, dp, con, ccmax, max_retry, dropdatabase, db, u, pass, debug:=readArguments()
	dataCollector.InitFilePerformanceResults()
	dataCollector.Collect(*input, *dp, *con, ccmax, *max_retry, *dropdatabase, *db, *u, pass, *debug)
}

/*inizialize arguments*/
func readArguments()(input *string, dp *string, con *int, ccmax int, max_retry *int, dropdatabase *bool, db *string, u *string, pass string, debug *bool){
	input = flag.String("i", "", "Input file with domains to analize")
	dp = flag.String("dp", "", "Dont probe file with network to not ask")
	con = flag.Int("c", 50, "Concurrency: how many routines")
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
		ccmax=*con
	}
	return
}