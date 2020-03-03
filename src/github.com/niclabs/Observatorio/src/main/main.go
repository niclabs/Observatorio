package main
import(

	"gopkg.in/yaml.v2"
	"os"
	"github.com/niclabs/Observatorio/src/dataCollector"
	"flag"
	"github.com/howeyc/gopass"
	"fmt"
)

type Config struct {
    RunArguments struct {
        Input_filepath string `yaml:"inputfilepath"`
        Dontprobe_filepath string `yaml:"dontprobefilepath"`
        Concurrency int `yaml:"concurrency"`
        Ccmax int `yaml:"ccmax"`
        Max_retry int `yaml:"maxretry"`
        Drop_database bool `yaml:"dropdatabase"`
        Debug bool `yaml:"debug"`
    } `yaml:"runargs"`
    Database struct {
    	Database_name string `yaml:"dbname"`
        Username string `yaml:"user"`
        Password string `yaml:"pass"`
    } `yaml:"database"`
}

var CONFIG_FILE string = "config.yml"

func main(){

	//Read config 
	f, err := os.Open(CONFIG_FILE)
	if err != nil {
	    fmt.Printf(err.Error())
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
	    fmt.Printf(err.Error())
	}
	fmt.Printf(cfg.RunArguments.Input_filepath)
	fmt.Printf(string(cfg.RunArguments.Concurrency))
	//input, dp, con, ccmax, max_retry, dropdatabase, db, u, pass, debug:=readArguments()

	

	dataCollector.InitCollect(cfg.RunArguments.Dontprobe_filepath, cfg.RunArguments.Drop_database, cfg.Database.Username, cfg.Database.Password, cfg.Database.Database_name)
	dataCollector.Collect(cfg.RunArguments.Input_filepath, cfg.RunArguments.Concurrency, cfg.RunArguments.Ccmax, cfg.RunArguments.Max_retry, cfg.Database.Database_name, cfg.Database.Username, cfg.Database.Password, cfg.RunArguments.Debug)
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