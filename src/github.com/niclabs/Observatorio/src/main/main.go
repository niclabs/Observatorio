package main
import(

	"gopkg.in/yaml.v2"
	"os"
	"github.com/niclabs/Observatorio/src/dataCollector"
	"github.com/niclabs/Observatorio/src/geoIPUtils"
	"fmt"
)

type Config struct {
    RunArguments struct {
        Input_filepath string `yaml:"inputfilepath"`
        Dontprobe_filepath string `yaml:"dontprobefilepath"`
        Concurrency int `yaml:"concurrency"`
        Drop_database bool `yaml:"dropdatabase"`
        Debug bool `yaml:"debug"`
        Dns_servers []string `yaml:"dnsservers"`

    } `yaml:"runargs"`
    Database struct {
    	Database_name string `yaml:"dbname"`
        Username string `yaml:"dbuser"`
        Password string `yaml:"dbpass"`
    } `yaml:"database"`
    Geoip struct {
    	Geoip_path string `yaml:"geoippath"`
        Geoip_asn_filename string `yaml:"geoipasnfilename"`
        Geoip_country_filename string `yaml:"geoipcountryfilename"`
        Geoip_update_script string `yaml:"geoipupdatescript"`
    } `yaml:"geoip"`
}

var CONFIG_FILE string = "config.yml"
var err error;
func main(){

	//Read config file
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

	//Init geoip databases
	var geoipDB *geoIPUtils.GeoipDB = geoIPUtils.InitGeoIP(cfg.Geoip.Geoip_path, cfg.Geoip.Geoip_country_filename, cfg.Geoip.Geoip_asn_filename, cfg.Geoip.Geoip_update_script)

	
	//Initialize collect 
	err = dataCollector.InitCollect(cfg.RunArguments.Dontprobe_filepath, cfg.RunArguments.Drop_database, cfg.Database.Username, cfg.Database.Password, cfg.Database.Database_name, geoipDB, cfg.RunArguments.Dns_servers)
	if(err != nil){
		fmt.Println(err)
		return
	}
	
	//start collect
	dataCollector.StartCollect(cfg.RunArguments.Input_filepath, cfg.RunArguments.Concurrency, cfg.Database.Database_name, cfg.Database.Username, cfg.Database.Password, cfg.RunArguments.Debug)
	
	dataCollector.EndCollect();


	geoIPUtils.CloseGeoIP(geoipDB);
	//analyze data

	//generate graphics

		
}

