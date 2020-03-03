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


func InitCollect(dplf string , drop bool, user string, password string, dbname string){
	//check Dont probelist file
	dontProbeList = InitializeDontProbeList(dplf)

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

func collectDomainInfo(domain_name string, run_id int, db *sql.DB) {
	c:=new(dns.Client)
	var domain_id int
	/*create domain*/
	domain_id = dbController.SaveDomain(domain_name, run_id, db)
	/*Obtener NS del dominio*/
	var domain_name_servers string;
	{
		/*Obtener NS del dominio*/
		nss, _, err := dnsUtils.GetRecordSet(domain_name, dns.TypeNS, config_servers,c)
		if (err != nil) {
			if(nss!=nil){
				fmt.Println("error but answer",nss.String())
			}
			manageError(strings.Join([]string{"get NS", domain_name, err.Error()}, ""))
			fmt.Println("Error asking for NS", domain_name, err.Error())
		} else {
			if(len(nss.Answer)==0 || nss.Answer==nil){
				if(nss.Answer==nil){
					fmt.Println("no answer","no NS asociated to domain:", domain_name, nss.Answer)
				} else{
					fmt.Println("0 answer","no NS asociated to domain:", domain_name, nss.Answer)

					for _, ns := range nss.Answer {
						fmt.Println(ns.String())
					}
				}
			}
			NameServer:
			for _, ns := range nss.Answer {
				if ns, ok := ns.(*dns.NS); ok {
					var nameserverid int
					available, rtt, err := dnsUtils.CheckAvailability(domain_name, ns,c)
					if (err != nil) {
						nameserverid = dbController.CreateNS(ns, domain_id, run_id, db, false)
						manageError(strings.Join([]string{"checkAvailability", domain_name, ns.Ns, err.Error(), rtt.String()}, ""))
					} else {
						/*create ns*/
						{
							/*create ns*/
							nameserverid = dbController.CreateNS(ns, domain_id, run_id, db, available)
						}

						/*get A and AAAA*/
						{
							/*get A and AAAA*/
							//getANSRecord:
							ipv4, err := dnsUtils.GetARecords(ns.Ns, config_servers,c)
							if (err != nil) {

								manageError(strings.Join([]string{"getANS", domain_name, ns.Ns, err.Error()}, ""))
							} else {

								for _, ip := range ipv4 {
									ips := net.IP.String(ip)
									dontProbe := true
									asn := geoIPUtils.GetIPASN(ips, geoip_asn_db)
									country := geoIPUtils.GetIPCountry(ips, geoip_country_db)
									if (isIPInDontProbeList(ip)) {
										fmt.Println("domain ", domain_name, "in DontProbeList", ip)
										//TODO Future: save DONTPROBELIST in nameserver? and Domain?
										dbController.SaveNSIP(nameserverid, ips, country, asn, dontProbe, run_id, db)
										continue NameServer
									}
									dontProbe= false
									domain_name_servers = ips;
									dbController.SaveNSIP(nameserverid, ips, country, asn, dontProbe, run_id, db)
								}
							}
							//getAAAANSRecord:
							ipv6, err := dnsUtils.GetAAAARecords(ns.Ns, config_servers,c)
							if (err != nil) {

								manageError(strings.Join([]string{"getAAAANS", domain_name, ns.Ns, err.Error()}, ""))
							} else {
								for _, ip := range ipv6 {
									ips := net.IP.String(ip)
									country:= geoIPUtils.GetIPCountry(ips, geoip_country_db)
									asn := geoIPUtils.GetIPASN(ips, geoip_asn_db)
									dbController.SaveNSIP(nameserverid, ips, country, asn, false, run_id, db)
								}
							}
						}

						if (len(domain_name_servers)!=0) {
							recursivity := false
							EDNS := false
							loc_query := false
							TCP := false
							zone_transfer := false
							if (available) {
								//edns, recursivity, tcp, zone_transfer, loc_query
								//------------------------------Recursividad y EDNS----------------------
								RecAndEDNS := new(dns.Msg)
								RecAndEDNS, rtt, err = dnsUtils.GetRecursivityAndEDNS(domain_name, ns.Ns, "53",c)
								if (err != nil) {
									manageError(strings.Join([]string{"Rec and EDNS", domain_name, ns.Ns, err.Error(), rtt.String()}, ""))
								} else {
									if (RecAndEDNS.RecursionAvailable) {
										recursivity = true
									}
									if (RecAndEDNS.IsEdns0() != nil) {
										EDNS = true
									}
								}



								//TCP---------------------
								c.Net="tcp"
								tcp, _, err := dnsUtils.GetRecordSetTCP(domain_name, dns.TypeSOA, []string{ns.Ns},c)
								c.Net="udp"
								if (err != nil) {
									//manageError(strings.Join([]string{"TCP", domain_name, ns.Ns, err.Error()},""))
								} else {
									for _, tcpa := range tcp.Answer {
										if (tcpa != nil) {
											TCP = true
											break
										}
									}
								}

								//Zone transfer---------------------
								zt, err := dnsUtils.ZoneTransfer(domain_name, ns.Ns)
								if (err != nil) {
									manageError(strings.Join([]string{"zoneTransfer", domain_name, ns.Ns, err.Error()}, ""))
								} else {
									val := <-zt
									if val != nil {
										if(val.Error!=nil) {
										} else{
											zone_transfer = true
										}


									}
								}



								//Wrong Queries (tipos extraños como loc)
								loc, _, err := dnsUtils.GetRecordSet(domain_name, dns.TypeLOC, []string{ns.Ns},c)
								if (err != nil) {
									manageError(strings.Join([]string{"locQuery", domain_name, ns.Ns, err.Error()}, ""))
								} else {
									for _, loca := range loc.Answer {
										if _, ok := loca.(*dns.LOC); ok {
											loc_query = true
											break
										}
									}
								}
							}
							dbController.SaveNS(recursivity, EDNS, TCP, zone_transfer, loc_query, nameserverid, db)
						}

					}

				}

			}
		}
	}
	//checkServer


	if (len(domain_name_servers)!=0) {
		//Get A and AAAA records
		{
			//Get A and AAAA records

			ipv4, err := dnsUtils.GetARecords(domain_name, []string{domain_name_servers},c)

			if (err != nil) {
				manageError(strings.Join([]string{"get a record", domain_name, err.Error()}, ""))
			} else {
				for _, ip := range ipv4 {
					ips := net.IP.String(ip)
					dbController.SaveDomainIp(ips, domain_id, run_id, db)
				}
			}

			ipv6, err := dnsUtils.GetAAAARecords(domain_name, []string{domain_name_servers},c)
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
			soa, err := dnsUtils.CheckSOA(domain_name, []string{domain_name_servers},c)
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
			dss, _, err := dnsUtils.GetRecordSet(domain_name, dns.TypeDS, config_servers,c)
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
					rrsigs, _, err := dnsUtils.GetRecordSetWithDNSSEC(domain_name, dns.TypeDS, config_servers[0], c)
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
								dnskeys, _, _ = dnsUtils.GetRecordSetWithDNSSEC(rrsig.SignerName, dns.TypeDNSKEY, config_servers[0], c)
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



			dnskeys_domain_name, _, err := dnsUtils.GetRecordSetWithDNSSEC(domain_name, dns.TypeDNSKEY, domain_name_servers,c)
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
							dnskeys, _, _ = dnsUtils.GetRecordSetWithDNSSEC(rrsig1.SignerName, dns.TypeDNSKEY, config_servers[0], c)
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
						domain_name := "zskldhoisdh123dnakjdshaksdjasmdnaksjdh" + "." + d
						t := dns.TypeA
						in, _, err := dnsUtils.GetRecordSetWithDNSSEC(domain_name, t, domain_name_servers,c)
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
												dnskeys, _, _ = dnsUtils.GetRecordSetWithDNSSEC(rrsig.SignerName, dns.TypeDNSKEY, config_servers[0], c)
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
												dnskeys, _, _ = dnsUtils.GetRecordSetWithDNSSEC(rrsig.SignerName, dns.TypeDNSKEY, config_servers[0],c)
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