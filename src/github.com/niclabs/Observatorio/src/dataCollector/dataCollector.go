package dataCollector
import ("fmt"
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/miekg/dns"
	"time"
	"net"
	"runtime"
	"sync"
	"strings"
	"github.com/niclabs/Observatorio/src/dbController"
	"github.com/oschwald/geoip2-golang"
	"github.com/niclabs/Observatorio/src/dnsUtils"
	"github.com/niclabs/Observatorio/src/geoIPUtils"
	"github.com/niclabs/Observatorio/src/utils"
)


var concurrency int = 100;
var dontProbeListFile string;
var dontProbeList []*net.IPNet;

var total_time int = 0;


var debug = false;
var err error;
var geoip_country_db *geoip2.Reader;
var geoip_asn_db *geoip2.Reader;


var config_servers []string ;

var weird_string_domain_name = "zskldhoisdh123dnakjdshaksdjasmdnaksjdh";//potentially unexistent subdomain To use with NSEC


var dns_client *dns.Client;


func InitCollect(dont_probe_file_name string , drop bool, user string, password string, dbname string){
	//check Dont probelist file
	dontProbeList = InitializeDontProbeList(dont_probe_file_name)

	//Init geoip
	geoip_country_db, geoip_asn_db = geoIPUtils.InitGeoIP()	

	//initialize database (create tables if not created already and drop database if indicated)
	database, err := sql.Open("postgres", "user="+user+" password="+password+" dbname="+dbname+" sslmode=disable")
	if err != nil {
		fmt.Println(err)
		return
	}
	dbController.CreateTables(database, drop)
	database.Close()

	//set maximum number of cpus
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println("num CPU:",runtime.NumCPU())
	
	//obtain config default dns servers
	config, _ := dns.ClientConfigFromFile("/etc/resolv.conf")
	config_servers = config.Servers 

	dns_client = new(dns.Client)


}

func InitializeDontProbeList(dpf string)(dontProbeList  []*net.IPNet ){
	dontProbeListFile := dpf
	if(len(dontProbeListFile)==0) {
		fmt.Println("no dont Pobe list file found")
		return
	}
	domain_names, err := utils.ReadLines(dontProbeListFile)
	if(err!=nil){
		fmt.Println(err.Error())

	}
	fmt.Println("don probe file ok")
	for _,domain_name:=range domain_names{

		if(strings.Contains(domain_name,"#") || len(domain_name)==0){
			continue
		}
		_,ipNet, err :=net.ParseCIDR(domain_name)
		if(err!=nil){
			fmt.Println("no CIDR in DontProbeList:",domain_name)
		}
		dontProbeList=append(dontProbeList,ipNet)
	}
	return dontProbeList
}


func StartCollect(input string, c int, dbname string, user string, password string, debug_bool bool) {
	
	database, err := sql.Open("postgres", "user="+user+" password="+password+" dbname="+dbname+" sslmode=disable")
	if err != nil {
		fmt.Println(err)
		return
	}
	/*Initialize*/
	concurrency = c
	run_id := dbController.NewRun(database)
	debug = debug_bool
	
	/*Collect data*/
	collect(database, input, run_id)

	fmt.Println("TotalTime(nsec):", total_time ," (sec) ", total_time/1000000000," (min:sec) ", total_time/60000000000,":",total_time%60000000000/1000000000)

	database.Close()
}

func collect(db *sql.DB, inputFile string, run_id int){
	start_time:=time.Now()

	fmt.Println("EXECUTING WITH ", concurrency , " GOROUTINES;")

	domains_list, err := utils.ReadLines(inputFile)
	if(err!=nil){
		fmt.Println("Error reading domains list" + err.Error())
		return
	}

	domains_queue := make(chan string, concurrency)
	wg := sync.WaitGroup{}
	wg.Add(concurrency)
	/*Init n routines to read the queue*/
	for i := 0; i < concurrency; i++ {
		go func(run_id int) {
			j:=0
			for domain_name := range domains_queue {
				//t2:=time.Now()
				collectDomainInfo(domain_name, run_id, db)
				//duration := time.Since(t2)
				j++
			}
			wg.Done()
		}(run_id)
	}


	/*fill the queue with data to obtain*/
	for _, domain_name := range domains_list {
		domain_name := dns.Fqdn(domain_name)
		domains_queue <- domain_name
	}

	/*Close the queue*/
	close(domains_queue)
	
	/*wait for routines to finish*/
	wg.Wait()
	

	total_time = (int)(time.Since(start_time).Nanoseconds())
	dbController.SaveCorrectRun(run_id, total_time, true, db)
	fmt.Println("Successful Run. run_id:", run_id)
	db.Close()
}

func manageError(err string){
	if(debug){
		fmt.Println(err)
	}
}

func getDomainsNameservers(domain_name string)(nameservers []dns.RR){
	
	nss, _, err := dnsUtils.GetRecordSet(domain_name, dns.TypeNS, config_servers, dns_client)
	if (err != nil) {
		manageError(strings.Join([]string{"get NS", domain_name, err.Error()}, ""))
		fmt.Println("Error asking for NS", domain_name, err.Error())
		return nil
	}else {
		if(len(nss.Answer) == 0 || nss.Answer == nil){
			return nil
		}
		return nss.Answer
	}
}

func obtain_NS_IPv4_info(ip net.IP, domain_id int, domain_name string, nameserver_id int, run_id int, db *sql.DB) (nameserver_ip_string string){
	nameserver_ip_string = net.IP.String(ip)
	dontProbe := true
	asn := geoIPUtils.GetIPASN(nameserver_ip_string, geoip_asn_db)
	country := geoIPUtils.GetIPCountry(nameserver_ip_string, geoip_country_db)
	if (isIPInDontProbeList(ip)) {
		fmt.Println("domain ", domain_name, "in DontProbeList", ip)
		//TODO Future: save DONTPROBELIST in nameserver? and Domain?
		dbController.SaveNSIP(nameserver_id, nameserver_ip_string, country, asn, dontProbe, run_id, db)
		return ""
	}
	dontProbe = false
	dbController.SaveNSIP(nameserver_id, nameserver_ip_string, country, asn, dontProbe, run_id, db)
	return nameserver_ip_string
}
func obtain_NS_IPv6_info(ip net.IP, domain_id int, domain_name string, nameserver_id int, run_id int, db *sql.DB) (nameserver_ip_string string){
	nameserver_ip_string = net.IP.String(ip)
	country:= geoIPUtils.GetIPCountry(nameserver_ip_string, geoip_country_db)
	asn := geoIPUtils.GetIPASN(nameserver_ip_string, geoip_asn_db)
	dbController.SaveNSIP(nameserver_id, nameserver_ip_string, country, asn, false, run_id, db)
	return nameserver_ip_string
}
func checkRecursivityAndEDNS(domain_name string, ns string)(recursion_available bool, EDNS bool){
	RecAndEDNS := new(dns.Msg)
	RecAndEDNS, rtt, err := dnsUtils.GetRecursivityAndEDNS(domain_name, ns, "53", dns_client)
	if (err != nil) {
		manageError(strings.Join([]string{"Rec and EDNS", domain_name, ns, err.Error(), rtt.String()}, ""))
	} else {
		if (RecAndEDNS.RecursionAvailable) {
			recursion_available = true
		}
		if (RecAndEDNS.IsEdns0() != nil) {
			EDNS = true
		}
	}
	return recursion_available, EDNS
}
func checkTCP(domain_name string, ns string)(TCP bool){
	dns_client.Net="tcp"
	tcp, _, err := dnsUtils.GetRecordSetTCP(domain_name, dns.TypeSOA, ns, dns_client)
	dns_client.Net="udp"
	if (err != nil) {
		//manageError(strings.Join([]string{"TCP", domain_name, ns.Ns, err.Error()},""))
		return false
	} else {
		TCP = false
		for _, tcpa := range tcp.Answer {
			if (tcpa != nil) {
				TCP = true
				break
			}
		}
		return TCP
	}
}
func checkZoneTransfer(domain_name string, ns string)(zone_transfer bool){
	zone_transfer = false
	zt, err := dnsUtils.ZoneTransfer(domain_name, ns)
	if (err != nil) {
		manageError(strings.Join([]string{"zoneTransfer", domain_name, ns, err.Error()}, ""))
	} else {
		val := <-zt
		if val != nil {
			if(val.Error!=nil) {
			} else{
				fmt.Printf("zone_transfer succeded oh oh!!")
				zone_transfer = true
			}
		}
	}
	return zone_transfer
}
func checkLOCQuery(domain_name string, ns string)(loc_query bool){
	loc_query = false
	loc, _, err := dnsUtils.GetRecordSet(domain_name, dns.TypeLOC, []string{ns}, dns_client)
	if (err != nil) {
		manageError(strings.Join([]string{"locQuery", domain_name, ns, err.Error()}, ""))
	} else {
		for _, loca := range loc.Answer {
			if _, ok := loca.(*dns.LOC); ok {
				loc_query = true
				break
			}
		}
	}
	return loc_query
}

func getAndSaveDomainIPv4(domain_name string, domain_name_servers []string, dns_client *dns.Client, domain_id int, run_id int , db *sql.DB)(){
	ipv4, err := dnsUtils.GetARecords(domain_name, domain_name_servers, dns_client)
	if (err != nil) {
		manageError(strings.Join([]string{"get a record", domain_name, err.Error()}, ""))
	} else {
		for _, ip := range ipv4 {
			ips := net.IP.String(ip)
			dbController.SaveDomainIp(ips, domain_id, run_id, db)
		}
	}
	return
}

func getAndSaveDomainIPv6(domain_name string, domain_name_servers []string, dns_client *dns.Client, domain_id int, run_id int, db *sql.DB)(){
			
	ipv6, err := dnsUtils.GetAAAARecords(domain_name, domain_name_servers, dns_client)
	if (err != nil) {

		manageError(strings.Join([]string{"get AAAA record", domain_name, err.Error()}, ""))
	} else {
		for _, ip := range ipv6 {
			ips := net.IP.String(ip)
			dbController.SaveDomainIp(ips, domain_id, run_id, db)
		}
	}
}
func collectDomainInfo(domain_name string, run_id int, db *sql.DB) {
	
	var domain_id int

	// Create domain and save it in database
	domain_id = dbController.SaveDomain(domain_name, run_id, db)

	/*Obtener NS del dominio*/
	var domain_name_servers [] string;
	{//Check NSs of the domain
		/*Obtener NSs del dominio*/	
		var domains_nameservers []dns.RR = getDomainsNameservers(domain_name)


		for _, nameserver:=  range domains_nameservers{//for each nameserver of the current domain_name
			if ns, ok := nameserver.(*dns.NS); ok {
				var nameserver_id int
				available, rtt, err := dnsUtils.CheckAvailability(domain_name, ns, dns_client) //check if IPv4 exists
				if (err != nil) {
					nameserver_id = dbController.CreateNS(ns, domain_id, run_id, db, false)
					manageError(strings.Join([]string{"checkAvailability", domain_name, ns.Ns, err.Error(), rtt.String()}, ""))
				} else {
					nameserver_id = dbController.CreateNS(ns, domain_id, run_id, db, available) //create NS in database

					//get A records for NS
					ipv4, err := dnsUtils.GetARecords(ns.Ns, config_servers, dns_client)
					if (err != nil) {
						manageError(strings.Join([]string{"getANS", domain_name, ns.Ns, err.Error()}, ""))
					} else {
						for _, ip := range ipv4 {
							nameserver_ip_string := obtain_NS_IPv4_info(ip, domain_id, domain_name, nameserver_id, run_id, db)
							if( nameserver_ip_string != ""){
								domain_name_servers = append(domain_name_servers, nameserver_ip_string)
							}
						}
					}
					//get AAAA records for NS
					ipv6, err := dnsUtils.GetAAAARecords(ns.Ns, config_servers, dns_client)
					if (err != nil) {
						manageError(strings.Join([]string{"getAAAANS", domain_name, ns.Ns, err.Error()}, ""))
					} else {
						for _, ip := range ipv6 {
							nameserver_ip_string := obtain_NS_IPv6_info(ip, domain_id, domain_name, nameserver_id, run_id, db)
							if( nameserver_ip_string != ""){
								domain_name_servers = append(domain_name_servers, nameserver_ip_string)
							}
						}
					}

					// if there is at least one nameserver with IP...
					if (len(domain_name_servers)!=0) {
						recursivity := false
						EDNS := false
						loc_query := false
						TCP := false
						zone_transfer := false
						if (available) {
							// edns, recursivity, tcp, zone_transfer, loc_query
							// Recursividad y EDNS
							recursivity, EDNS = checkRecursivityAndEDNS(domain_name, ns.Ns)
							// TCP
							TCP = checkTCP(domain_name, ns.Ns)
							// Zone transfer
							zone_transfer = checkZoneTransfer(domain_name, ns.Ns)
							// Wrong Queries (tipos extraños como loc)
							loc_query = checkLOCQuery(domain_name, ns.Ns)
						}
						dbController.SaveNS(recursivity, EDNS, TCP, zone_transfer, loc_query, nameserver_id, db)
					}
				}
			}
		}
	}// end check nameservers


	//Check domain info (asking to NS)
	if (len(domain_name_servers)!=0) {
		
		{//Get A and AAAA records
			getAndSaveDomainIPv4(domain_name, domain_name_servers, dns_client, domain_id, run_id, db)
			getAndSaveDomainIPv6(domain_name, domain_name_servers, dns_client, domain_id, run_id, db)
			
			ipv6, err := dnsUtils.GetAAAARecords(domain_name, domain_name_servers, dns_client)
			if (err != nil) {

				manageError(strings.Join([]string{"get AAAA record", domain_name, err.Error()}, ""))
			} else {
				for _, ip := range ipv6 {
					ips := net.IP.String(ip)
					dbController.SaveDomainIp(ips, domain_id, run_id, db)
				}
			}
		}

		/*check soa*/
		{
			/*check soa*/
			SOA := false
			soa, err := dnsUtils.CheckSOA(domain_name, domain_name_servers, dns_client)
			if (err != nil) {
				manageError(strings.Join([]string{"check soa", domain_name, err.Error()}, ""))

			} else {

				for _, soar := range soa.Answer {
					if _, ok := soar.(*dns.SOA); ok {
						SOA = true
					}
				}
			}
			dbController.SaveSoa(SOA, domain_id, db)

		}


		//var server string;


		/*check DNSSEC*/
		{
			/*check DNSSEC*/

			/*ds*/
			dss, _, err := dnsUtils.GetRecordSet(domain_name, dns.TypeDS, config_servers, dns_client)
			if (err != nil) {
				//manageError(strings.Join([]string{"DS record", domain_name, err.Error()}, ""))
			} else {
				ds_ok := false
				ds_found := false

				var ds_rrset []dns.RR
				for _, ds := range dss.Answer {
					if ds1, ok := ds.(*dns.DS); ok {
						ds_found=true
						ds_rrset = append(ds_rrset,ds1)
						var algorithm = int(ds1.Algorithm)
						var keyTag int = int(ds1.KeyTag)
						var digestType int = int(ds1.DigestType)
						digest := ds1.Digest
						dbController.SaveDS(domain_id, algorithm, keyTag, digestType, digest, run_id, db)
					}
				}
				if(ds_found) {
					rrsigs, _, err := dnsUtils.GetRecordSetWithDNSSEC(domain_name, dns.TypeDS, config_servers, dns_client)
					if (err != nil) {
						//manageError(strings.Join([]string{"DS record", domain_name, err.Error()}, ""))
					} else {
						for _, ds := range rrsigs.Answer {
							if rrsig, ok := ds.(*dns.RRSIG); ok {
								dbController.SaveRRSIG(rrsig, domain_id, run_id, db)
								expired := false
								key_found := false
								verified := false
								var dnskeys *dns.Msg

								if (rrsig.TypeCovered != dns.TypeDS) {
									continue
								}
								if !rrsig.ValidityPeriod(time.Now().UTC()) {
									expired = true
								}
								//---------------DNSKEY----------------------------
								dnskeys, _, _ = dnsUtils.GetRecordSetWithDNSSEC(rrsig.SignerName, dns.TypeDNSKEY, config_servers, dns_client)
								if (dnskeys != nil && dnskeys.Answer != nil) {
									key := dnsUtils.FindKey(dnskeys, rrsig)
									if (key != nil) {
										key_found = true
										if err := rrsig.Verify(key, ds_rrset); err != nil {
											fmt.Printf(";- Bogus signature, %s does not validate (DNSKEY %s/%d/%s) [%s] %s\n", (rrsig.Hdr.Name), key.Header().Name, key.KeyTag(), "net", err, expired)
											verified = false
										} else {
											verified = true
										}
									}
								} else {
									fmt.Println("DS error no key found")
								}
								if (key_found && verified && !expired) {
									ds_ok = true
									break
								}
							}
						}
					}
				}
				dbController.UpdateDomainDSInfo(domain_id, ds_found, ds_ok, db)
			}


			/*dnskeys*/



			dnskeys_domain_name, _, err := dnsUtils.GetRecordSetWithDNSSEC(domain_name, dns.TypeDNSKEY, domain_name_servers, dns_client)
			if (err != nil) {
				manageError(strings.Join([]string{"dnskey", domain_name, err.Error()}, ""))
			} else {
				if (len(dnskeys_domain_name.Answer) != 0) {
					dnskey_found := false
					dnskey_ok := false
					var dnskey_rrset []dns.RR
					/*si no tiene dnskey no busco nada de dnssec*/
					for _, dnskey := range dnskeys_domain_name.Answer {
						if dnskey1, ok := dnskey.(*dns.DNSKEY); ok {
							dnskey_found = true
							dnskey_rrset = append(dnskey_rrset,dnskey1)
							DSok:=false
							if(dnskey1.Flags==1){
								//check DS
								ds1 := dnskey1.ToDS(dnskey1.Algorithm)
								for _, ds := range dss.Answer {
									if ds2,ok:=ds.(*dns.DS);ok{
										if(ds2.Digest==ds1.Digest) {
											DSok = true;
										}
									}
								}
							}
							dbController.SaveDNSKEY(dnskey1, DSok, domain_id, run_id, db)
						}
					}
					rrsigs := dnskeys_domain_name
					for _, rrsig := range rrsigs.Answer {
						if rrsig1, ok := rrsig.(*dns.RRSIG); ok {
							if(rrsig1.TypeCovered!=dns.TypeDNSKEY){
								continue
							}
							dbController.SaveRRSIG(rrsig1, domain_id, run_id, db)
							expired := false
							key_found := false
							verified := false
							var dnskeys *dns.Msg

							if !rrsig1.ValidityPeriod(time.Now().UTC()) {
								expired = true
							}
							//---------------DNSKEY----------------------------
							dnskeys, _, _ = dnsUtils.GetRecordSetWithDNSSEC(rrsig1.SignerName, dns.TypeDNSKEY, config_servers, dns_client)
							if (dnskeys != nil && dnskeys.Answer != nil) {
								key := dnsUtils.FindKey(dnskeys, rrsig1)
								if (key != nil) {
									key_found = true
									if err := rrsig1.Verify(key, dnskey_rrset); err != nil {
										verified = false
									} else {
										verified = true
									}
								}
							} else {
								//fmt.Println("DS error no key found")
							}
							if (key_found && verified && !expired) {
								dnskey_ok = true
								break
							}
						}
					}

					dbController.UpdateDomainDNSKEYInfo(domain_id, dnskey_found, dnskey_ok, db)



					/*nsec/3*/
					{
						d := domain_name
						domain_name := weird_string_domain_name + "." + d
						t := dns.TypeA
						in, _, err := dnsUtils.GetRecordSetWithDNSSEC(domain_name, t, domain_name_servers, dns_client)
						if (err != nil) {
							fmt.Println(err.Error())
							manageError(strings.Join([]string{"nsec/3", domain_name, err.Error()}, ""))
						} else {
							non_existence_status := in.Rcode;
							dbController.UpdateNonExistence(domain_id, non_existence_status, db);

							for _, ans := range in.Ns {
								//authority section
								if nsec, ok := ans.(*dns.NSEC); ok {
									ncover := false
									ncoverwc := false
									niswc := false
									last := nsec.Hdr.Name
									next := nsec.NextDomain
									ttl := int(nsec.Hdr.Ttl)
									//save nsec
									nsec_id:=dbController.SaveNsec(domain_id, last, next, ttl, run_id, db)
									/*verify nsec in other task*/

									if (dnsUtils.Less(domain_name, last) == 0) {
										niswc=true
									}else {
										wildcarddomain_name := "*." + d
										if (dnsUtils.Less(wildcarddomain_name, next) < 0) {
											ncoverwc = true
										}
										if ((dnsUtils.Less(domain_name, next) < 0 && dnsUtils.Less(domain_name, last) > 0) || (dnsUtils.Less(domain_name, last) > 0 && next == d)) {
											ncover = true
										}
									}
									expired:=false
									key_found:= false
									verified := false
									for _, ats := range in.Ns {
										if rrsig, ok := ats.(*dns.RRSIG); ok {
											expired = false
											key_found = false
											verified = false
											var dnskeys *dns.Msg
											if (rrsig.TypeCovered != dns.TypeNSEC) {
												continue
											}
											if !rrsig.ValidityPeriod(time.Now().UTC()) {
												expired = true
											}
											//---------------DNSKEY----------------------------
											if (rrsig.SignerName != domain_name) {
												dnskeys, _, _ = dnsUtils.GetRecordSetWithDNSSEC(rrsig.SignerName, dns.TypeDNSKEY, config_servers, dns_client)
											} else {
												dnskeys = dnskeys_domain_name
											}



											if (dnskeys != nil && dnskeys.Answer != nil) {
												key := dnsUtils.FindKey(dnskeys, rrsig)
												if (key != nil) {
													key_found = true
													var rrset []dns.RR
													rrset = []dns.RR{nsec}
													if err := rrsig.Verify(key, rrset); err != nil {
														verified=false
													}else{
														verified=true

													}
												}
											}
											if (key_found && verified && !expired) {
												break
											}
										}
									}

									dbController.UpdateNSEC(key_found&&verified&&!expired, ncover, ncoverwc, niswc, nsec_id, db)

								} else
								if nsec3, ok := ans.(*dns.NSEC3); ok {
									hashed_name := nsec3.Hdr.Name
									next_hashed_name := nsec3.NextDomain
									iterations := int(nsec3.Iterations)
									hash_algorithm := int(nsec3.Hash)
									salt := nsec3.Salt
									nsec3_id:=dbController.SaveNsec3(domain_id, hashed_name, next_hashed_name, iterations, hash_algorithm, salt, run_id, db)
									n3cover := false
									n3coverwc := false
									n3match := false
									n3wc := false
									n3cover = nsec3.Cover(domain_name)
									n3coverwc = nsec3.Cover("*." + d)
									n3match = nsec3.Match(d)
									n3wc = nsec3.Match("*."+d)
									expired:=false
									key_found:= false
									verified := false
									firmas:
									for _, ats := range in.Ns {
										if rrsig, ok := ats.(*dns.RRSIG); ok {
											expired = false
											key_found = false
											verified = false

											var dnskeys *dns.Msg
											if (rrsig.TypeCovered != dns.TypeNSEC3) {
												continue firmas
											}
											if !rrsig.ValidityPeriod(time.Now().UTC()) {
												expired = true
											}
											//---------------DNSKEY----------------------------
											if (rrsig.SignerName != domain_name) {
												dnskeys, _, _ = dnsUtils.GetRecordSetWithDNSSEC(rrsig.SignerName, dns.TypeDNSKEY, config_servers, dns_client)
											} else {
												dnskeys = dnskeys_domain_name
											}
											if (dnskeys != nil && dnskeys.Answer != nil) {
												key := dnsUtils.FindKey(dnskeys, rrsig)
												if (key != nil) {
													key_found = true
													var rrset []dns.RR
													rrset = []dns.RR{nsec3}
													if err := rrsig.Verify(key, rrset); err != nil {
														verified=false
													}else{
														verified=true
													}
												}
											}
											if(key_found && verified && !expired){
												break
											}
										}
									}
									dbController.UpdateNSEC3(key_found&&verified&&!expired,key_found, verified, expired, n3match, n3cover, n3coverwc, n3wc, nsec3_id, db)
								}
							}
						}
					}
				}
			}
		}
	}
}



func isIPInDontProbeList(ip net.IP)(bool){
	var ipnet *net.IPNet
	for _,ipnet=range dontProbeList{
		if(ipnet.Contains(ip)){
			fmt.Println("DONT PROBE LIST ip: ",ip," found in: ",ipnet)
			return true;
		}
	}
	return false;
}