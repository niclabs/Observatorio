package main
import(

	"gopkg.in/yaml.v2"
	"os"
	"github.com/niclabs/Observatorio/src/dataCollector"
	"fmt"
)

type Config struct {
    RunArguments struct {
        Input_filepath string `yaml:"inputfilepath"`
        Dontprobe_filepath string `yaml:"dontprobefilepath"`
        Concurrency int `yaml:"concurrency"`
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
	dataCollector.Collect(cfg.RunArguments.Input_filepath, cfg.RunArguments.Concurrency, cfg.RunArguments.Max_retry, cfg.Database.Database_name, cfg.Database.Username, cfg.Database.Password, cfg.RunArguments.Debug)
}

