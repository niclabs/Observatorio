package dataCollector

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/miekg/dns"
	"github.com/niclabs/Observatorio/dbController"
	"github.com/niclabs/Observatorio/dnsUtils"
	"github.com/niclabs/Observatorio/geoIPUtils"
	"github.com/niclabs/Observatorio/utils"
	"github.com/oschwald/geoip2-golang"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"
)

var concurrency int = 100
var dontProbeListFile string
var dontProbeList []*net.IPNet

var totalTime int = 0

var debug = false
var err error
var geoipCountryDb *geoip2.Reader
var geoipAsnDb *geoip2.Reader

var configServers []string

var weirdStringSubdomainName = "zskldhoisdh123dnakjdshaksdjasmdnaksjdh" //potentially unexistent subdomain To use with NSEC

var dnsClient *dns.Client

func InitCollect(dontProbeFileName string, drop bool, user string, password string, host string, port int, dbname string, geoipdb *geoIPUtils.GeoipDB, dnsServers []string) error {
	//check Dont probelist file
	dontProbeList = InitializeDontProbeList(dontProbeFileName)

	//Init geoip
	geoipCountryDb = geoipdb.CountryDb
	geoipAsnDb = geoipdb.AsnDb

	url := fmt.Sprintf("postgres://%v:%v@%v:%v/%v?sslmode=disable",
		user,
		password,
		host,
		port,
		dbname)
	//initialize database (create tables if not created already and drop database if indicated)
	database, err := sql.Open("postgres", url)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	dbController.CreateTables(database, drop)
	database.Close()

	//set maximum number of cpus
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println("num CPU:", runtime.NumCPU())

	//obtain config default dns servers
	//config, _ := dns.ClientConfigFromFile("/etc/resolv.conf")
	configServers = dnsServers //config.Servers

	//dns client to use in future queries.
	dnsClient = new(dns.Client)

	return nil //no error.

}


func InitializeDontProbeList(dpf string) (dontProbeList []*net.IPNet) {
	dontProbeListFile := dpf
	if len(dontProbeListFile) == 0 {
		fmt.Println("no dont Pobe list file found")
		return
	}
	domainNames, err := utils.ReadLines(dontProbeListFile)
	if err != nil {
		fmt.Println(err.Error())

	}
	fmt.Println("don probe file ok")
	for _, domainName := range domainNames {

		if strings.Contains(domainName, "#") || len(domainName) == 0 {
			continue
		}
		_, ipNet, err := net.ParseCIDR(domainName)
		if err != nil {
			fmt.Println("no CIDR in DontProbeList:", domainName)
		}
		dontProbeList = append(dontProbeList, ipNet)
	}
	return dontProbeList
}

func StartCollect(input string, c int, dbname string, user string, password string, host string, port int, debugBool bool) (runId int){
	url := fmt.Sprintf("postgres://%v:%v@%v:%v/%v?sslmode=disable",
		user,
		password,
		host,
		port,
		dbname)
	database, err := sql.Open("postgres", url)
	if err != nil {
		fmt.Println(err)
		return
	}
	/*Initialize*/
	concurrency = c
	runId = dbController.NewRun(database)
	debug = debugBool

	/*Collect data*/
	createCollectorRoutines(database, input, runId)

	fmt.Println("TotalTime(nsec):", totalTime, " (sec) ", totalTime/1000000000, " (min:sec) ", totalTime/60000000000, ":", totalTime%60000000000/1000000000)

	database.Close()
	return runId
}

func createCollectorRoutines(db *sql.DB, inputFile string, runId int) {
	startTime := time.Now()

	fmt.Println("EXECUTING WITH ", concurrency, " GOROUTINES;")

	domainsList, err := utils.ReadLines(inputFile)
	if err != nil {
		fmt.Println("Error reading domains list" + err.Error())
		return
	}

	//CREATES THE ROUTINES
	domainsQueue := make(chan string, concurrency)
	wg := sync.WaitGroup{}
	wg.Add(concurrency)
	/*Init n routines to read the queue*/
	for i := 0; i < concurrency; i++ {
		go func(runId int) {
			j := 0
			for domainName := range domainsQueue {
				//t2:=time.Now()
				collectSingleDomainInfo(domainName, runId, db)
				//duration := time.Since(t2)
				j++
			}
			wg.Done()
		}(runId)
	}

	//fill the queue with data to obtain
	for _, domainName := range domainsList {
		domainName := dns.Fqdn(domainName)
		domainsQueue <- domainName
	}

	/*Close the queue*/
	close(domainsQueue)

	/*wait for routines to finish*/
	wg.Wait()

	totalTime = (int)(time.Since(startTime).Nanoseconds())
	dbController.SaveCorrectRun(runId, totalTime, true, db)
	fmt.Println("Successful Run. run_id:", runId)
	db.Close()
}

func manageError(err string) {
	if debug {
		fmt.Println(err)
	}
}

func getDomainsNameservers(domainName string) (nameservers []dns.RR) {

	nss, _, err := dnsUtils.GetRecordSet(domainName, dns.TypeNS, configServers, dnsClient)
	if err != nil {
		manageError(strings.Join([]string{"get NS", domainName, err.Error()}, ""))
		//fmt.Println("Error asking for NS", domainName, err.Error())
		return nil
	} else {
		if len(nss.Answer) == 0 || nss.Answer == nil {
			return nil
		}
		return nss.Answer
	}
}

func obtain_NS_IPv4_info(ip net.IP, domain_id int, domainName string, nameserverId int, runId int, db *sql.DB) (nameserverIpString string) {
	nameserverIpString = net.IP.String(ip)
	dontProbe := true
	asn := geoIPUtils.GetIPASN(nameserverIpString, geoipAsnDb)
	country := geoIPUtils.GetIPCountry(nameserverIpString, geoipCountryDb)
	if isIPInDontProbeList(ip) {
		fmt.Println("domain ", domainName, "in DontProbeList", ip)
		//TODO Future: save DONTPROBELIST in nameserver? and Domain?
		dbController.SaveNSIP(nameserverId, nameserverIpString, country, asn, dontProbe, runId, db)
		return ""
	}
	dontProbe = false
	dbController.SaveNSIP(nameserverId, nameserverIpString, country, asn, dontProbe, runId, db)
	return nameserverIpString
}
func obtain_NS_IPv6_info(ip net.IP, domain_id int, domain_name string, nameserverId int, runId int, db *sql.DB) (nameserverIpString string) {
	nameserverIpString = net.IP.String(ip)
	country := geoIPUtils.GetIPCountry(nameserverIpString, geoipCountryDb)
	asn := geoIPUtils.GetIPASN(nameserverIpString, geoipAsnDb)
	dbController.SaveNSIP(nameserverId, nameserverIpString, country, asn, false, runId, db)
	return nameserverIpString
}
func checkRecursivityAndEDNS(domainName string, ns string) (recursionAvailable bool, EDNS bool) {
	RecAndEDNS := new(dns.Msg)
	RecAndEDNS, rtt, err := dnsUtils.GetRecursivityAndEDNS(domainName, ns, "53", dnsClient)
	if err != nil {
		manageError(strings.Join([]string{"Rec and EDNS", domainName, ns, err.Error(), rtt.String()}, ""))
	} else {
		if RecAndEDNS.RecursionAvailable {
			recursionAvailable = true
		}
		if RecAndEDNS.IsEdns0() != nil {
			EDNS = true
		}
	}
	return recursionAvailable, EDNS
}
func checkTCP(domainName string, ns string) (TCP bool) {
	dnsClient.Net = "tcp"
	tcp, _, err := dnsUtils.GetRecordSetTCP(domainName, dns.TypeSOA, ns, dnsClient)
	dnsClient.Net = "udp"
	if err != nil {
		//manageError(strings.Join([]string{"TCP", domain_name, ns.Ns, err.Error()},""))
		return false
	} else {
		TCP = false
		for _, tcpa := range tcp.Answer {
			if tcpa != nil {
				TCP = true
				break
			}
		}
		return TCP
	}
}
func checkZoneTransfer(domainName string, ns string) (zoneTransfer bool) {
	zoneTransfer = false
	zt, err := dnsUtils.ZoneTransfer(domainName, ns)
	if err != nil {
		manageError(strings.Join([]string{"zoneTransfer", domainName, ns, err.Error()}, ""))
	} else {
		val := <-zt
		if val != nil {
			if val.Error != nil {
			} else {
				fmt.Printf("zone_transfer succeded oh oh!!")
				zoneTransfer = true
			}
		}
	}
	return zoneTransfer
}
func checkLOCQuery(domainName string, ns string) (locQuery bool) {
	locQuery = false
	loc, _, err := dnsUtils.GetRecordSet(domainName, dns.TypeLOC, []string{ns}, dnsClient)
	if err != nil {
		manageError(strings.Join([]string{"locQuery", domainName, ns, err.Error()}, ""))
	} else {
		for _, loca := range loc.Answer {
			if _, ok := loca.(*dns.LOC); ok {
				locQuery = true
				break
			}
		}
	}
	return locQuery
}

func getAndSaveDomainIPv4(domainName string, domainNameServers []string, domainId int, runId int, db *sql.DB) {
	ipv4, err := dnsUtils.GetARecords(domainName, domainNameServers, dnsClient)
	if err != nil {
		manageError(strings.Join([]string{"get a record", domainName, err.Error()}, ""))
	} else {
		for _, ip := range ipv4 {
			ips := net.IP.String(ip)
			dbController.SaveDomainIp(ips, domainId, runId, db)
		}
	}
	return
}

func getAndSaveDomainIPv6(domainName string, domainNameServers []string, domainId int, runId int, db *sql.DB) {

	ipv6, err := dnsUtils.GetAAAARecords(domainName, domainNameServers, dnsClient)
	if err != nil {

		manageError(strings.Join([]string{"get AAAA record", domainName, err.Error()}, ""))
	} else {
		for _, ip := range ipv6 {
			ips := net.IP.String(ip)
			dbController.SaveDomainIp(ips, domainId, runId, db)
		}
	}
}

func getAndSaveDomainSOA(domainName string, domainNameServers []string, domainId int, run_id int, db *sql.DB) {
	/*check soa*/
	SOA := false
	soa, err := dnsUtils.CheckSOA(domainName, domainNameServers, dnsClient)
	if err != nil {
		manageError(strings.Join([]string{"check soa", domainName, err.Error()}, ""))
	} else {
		for _, soar := range soa.Answer {
			if _, ok := soar.(*dns.SOA); ok {
				SOA = true
			}
		}
	}
	dbController.SaveSoa(SOA, domainId, db)
}

/*
func checkAndSaveDSs(domain_name string, servers []string, domain_id int, run_id int, db *sql.DB)(ds_found bool, ds_ok bool, ds_rrset []dns.RR){
	ds_found = false
	ds_ok = false
	dss, _, err := dnsUtils.GetRecordSet(domain_name, dns.TypeDS, config_servers, dns_client)
	if (err != nil) {
		//manageError(strings.Join([]string{"DS record", domain_name, err.Error()}, ""))
		return ds_found, ds_ok, nil
	}
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
	return ds_found, ds_ok,

}*/

func getAndSaveDS(domainName string, servers []string, domainId int, runId int, db *sql.DB) { //Find DS in superior level
	superiorDomain := strings.SplitN(domainName, ".", 2)[1] //removes first section in domain_name "a.b.c" -> ["a", "b.c"] //TODO check value.. .cl. wont work
	//if(superior_domain == ""){ not sure if necessary
	//	superior_domain = "."
	//}
	//find NS of superior domain
	var superiorNameservers []string
	superiorNss, _, err := dnsUtils.GetRecordSetWithDNSSEC(superiorDomain, dns.TypeNS, servers, dnsClient)
	if err != nil {
		manageError(err.Error())
		return
	}

	for _, superiorNs := range superiorNss.Answer {
		if currentSuperiorNs, ok := superiorNs.(*dns.NS); ok {
			superiorNameservers = append(superiorNameservers, currentSuperiorNs.Ns)
		}
	}
	//ask for DS to superior NSs
	dsRrset, _, err := dnsUtils.GetRecordSetWithDNSSEC(domainName, dns.TypeDS, superiorNameservers, dnsClient)
	if err != nil {
		manageError(err.Error())
		return
	}
	for _, ds := range dsRrset.Answer {
		if currentDs, ok := ds.(*dns.DS); ok {
			var algorithm = int(currentDs.Algorithm)
			var keyTag int = int(currentDs.KeyTag)
			var digestType int = int(currentDs.DigestType)
			digest := currentDs.Digest
			dbController.SaveDS(domainId, algorithm, keyTag, digestType, digest, runId, db)
		}
	}
}

func getAndSaveDNSKEYs(dnskeyRrs *dns.Msg, domain_name string, servers []string, domainId int, runId int, db *sql.DB) (dnskeyFound bool) {
	fmt.Println("getAndSaveDNSKEYs")
	dnskeyFound = true
	for _, dnskey := range dnskeyRrs.Answer {
		if dnskey1, ok := dnskey.(*dns.DNSKEY); ok {
			dnskeyFound = true
			//dnskey_rrset = append(dnskey_rrset,dnskey1)
			dbController.SaveDNSKEY(dnskey1, domainId, runId, db)
			/* do this in analysis
			dnskey_ds_ok = false;
			if(dnskey1.Flags&1 == 1){ //SEP (Secure Entry Point)
				//check DS agains dnskey
				ds1 := dnskey1.ToDS(dnskey1.Algorithm)
				for _, ds := range ds_rrset.Answer {
					if ds2,ok:=ds.(*dns.DS);ok{
						if(ds2.Digest==ds1.Digest) {
							dnskey_ds_ok = true;
							DSok = true
						}
					}
				}
			}
			dbController.SaveDNSKEY(dnskey1, dnskey_ds_ok, domain_id, run_id, db)
			dnskey_ds_ok = false;
			*/
		}
	}
	//get and save DNSKEY RRSIGS
	rrsigs := dnskeyRrs
	for _, rrsig := range rrsigs.Answer {
		if rrsig1, ok := rrsig.(*dns.RRSIG); ok {
			if rrsig1.TypeCovered != dns.TypeDNSKEY {
				continue
			}
			dbController.SaveRRSIG(rrsig1, domainId, runId, db)
		}
	}
	return dnskeyFound
}

func getAndSaveNSECinfo(domainName string, servers []string, domainId int, runId int, db *sql.DB) {
	d := domainName
	domainName = weirdStringSubdomainName + "." + d
	in, _, err := dnsUtils.GetRecordSetWithDNSSEC(domainName, dns.TypeA, servers, dnsClient)
	if err != nil {
		fmt.Println(err.Error())
		manageError(strings.Join([]string{"nsec/3", domainName, err.Error()}, ""))
	} else {
		nonExistenceStatus := in.Rcode
		dbController.UpdateNonExistence(domainId, nonExistenceStatus, db)

		for _, ans := range in.Ns {
			//authority section
			if nsec, ok := ans.(*dns.NSEC); ok {

				last := nsec.Hdr.Name
				next := nsec.NextDomain
				ttl := int(nsec.Hdr.Ttl)

				//nsec_id:=dbController.SaveNsec(domain_id, last, next, ttl, run_id, db)
				dbController.SaveNsec(domainId, last, next, ttl, runId, db)
				/*ToDo do this in analysis
				ncover := false
				ncoverwc := false
				niswc := false
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
							dnskeys, _, _ = dnsUtils.GetRecordSetWithDNSSEC(rrsig.SignerName, dns.TypeDNSKEY, servers, dns_client)
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
				dbController.UpdateNSEC(false, ncover, ncoverwc, niswc, nsec_id, db) //TODO check this
				*/
			} else {
				if nsec3, ok := ans.(*dns.NSEC3); ok {
					hashedName := nsec3.Hdr.Name
					nextHashedName := nsec3.NextDomain
					iterations := int(nsec3.Iterations)
					hashAlgorithm := int(nsec3.Hash)
					salt := nsec3.Salt
					//nsec3_id:=dbController.SaveNsec3(domain_id, hashed_name, next_hashed_name, iterations, hash_algorithm, salt, run_id, db)
					dbController.SaveNsec3(domainId, hashedName, nextHashedName, iterations, hashAlgorithm, salt, runId, db)
					/* TODO do this in analysis
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
								dnskeys, _, _ = dnsUtils.GetRecordSetWithDNSSEC(rrsig.SignerName, dns.TypeDNSKEY, servers, dns_client)
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
					dbController.UpdateNSEC3(false, verified, expired, n3match, n3cover, n3coverwc, n3wc, nsec3_id, db)//TODO check this
					*/
				}
			}
		}
	}
	return
}

func getAndSaveDNSSECinfo(domainName string, domainNameServers []string, domainId int, runId int, db *sql.DB) (dnskeyFound bool) {

	// Get DNSKEYS
	dnskeyRrs, _, err := dnsUtils.GetRecordSetWithDNSSEC(domainName, dns.TypeDNSKEY, domainNameServers, dnsClient)
	dnskeyFound = false
	if err != nil {
		manageError(strings.Join([]string{"dnskey", domainName, err.Error()}, ""))
		return dnskeyFound
	}
	if len(dnskeyRrs.Answer) == 0 {
		return dnskeyFound
	}

	//if any dnskey found, continue... else return (above)
	dnskeyFound = getAndSaveDNSKEYs(dnskeyRrs, domainName, domainNameServers, domainId, runId, db)

	//DSok := false
	//dnskey_ds_ok := false
	//var ds_rrset []dns.RR;
	getAndSaveDS(domainName, configServers, domainId, runId, db)

	dbController.UpdateDomainDNSKEYInfo(domainId, dnskeyFound, false, db) //TODO update this, second argument always false

	//get and save nsec/3 info

	return dnskeyFound
}

// Collects info from a single domain (ran by a routine) and save it to the databses.
func collectSingleDomainInfo(domainName string, runId int, db *sql.DB) {

	var domainId int
	// Create domain and save it in database
	domainId = dbController.SaveDomain(domainName, runId, db)

	/*Obtener NS del dominio*/
	var domainNameServers []string
	{ //Check NSs of the domain
		/*Obtener NSs del dominio*/
		var domainsNameservers []dns.RR = getDomainsNameservers(domainName)

		for _, nameserver := range domainsNameservers { //for each nameserver of the current domain_name
			if ns, ok := nameserver.(*dns.NS); ok {
				var nameserverId int
				available, rtt, err := dnsUtils.CheckAvailability(domainName, ns, dnsClient) //check if IPv4 exists
				if err != nil {
					nameserverId = dbController.CreateNS(ns, domainId, runId, db, false)
					manageError(strings.Join([]string{"checkAvailability", domainName, ns.Ns, err.Error(), rtt.String()}, ""))
				} else {
					nameserverId = dbController.CreateNS(ns, domainId, runId, db, available) //create NS in database

					//get A records for NS
					ipv4, err := dnsUtils.GetARecords(ns.Ns, configServers, dnsClient)
					if err != nil {
						manageError(strings.Join([]string{"getANS", domainName, ns.Ns, err.Error()}, ""))
					} else {
						for _, ip := range ipv4 {
							nameserverIpString := obtain_NS_IPv4_info(ip, domainId, domainName, nameserverId, runId, db)
							if nameserverIpString != "" {
								domainNameServers = append(domainNameServers, nameserverIpString)
							}
						}
					}
					//get AAAA records for NS
					ipv6, err := dnsUtils.GetAAAARecords(ns.Ns, configServers, dnsClient)
					if err != nil {
						manageError(strings.Join([]string{"getAAAANS", domainName, ns.Ns, err.Error()}, ""))
					} else {
						for _, ip := range ipv6 {
							nameserverIpString := obtain_NS_IPv6_info(ip, domainId, domainName, nameserverId, runId, db)
							if nameserverIpString != "" {
								domainNameServers = append(domainNameServers, nameserverIpString)
							}
						}
					}

					// if there is at least one nameserver with IP...
					if len(domainNameServers) != 0 {
						recursivity := false
						EDNS := false
						locQuery := false
						TCP := false
						zoneTransfer := false
						if available {
							// edns, recursivity, tcp, zone_transfer, loc_query
							// Recursividad y EDNS
							recursivity, EDNS = checkRecursivityAndEDNS(domainName, ns.Ns)
							// TCP
							TCP = checkTCP(domainName, ns.Ns)
							// Zone transfer
							zoneTransfer = checkZoneTransfer(domainName, ns.Ns)
							// Wrong Queries (tipos extraños como loc)
							locQuery = checkLOCQuery(domainName, ns.Ns)
						}
						dbController.SaveNS(recursivity, EDNS, TCP, zoneTransfer, locQuery, nameserverId, db)
					}
				}
			}
		}
	} // end check nameservers

	//Check domain info (asking to NS)
	if len(domainNameServers) != 0 {

		//Get A and AAAA records
		getAndSaveDomainIPv4(domainName, domainNameServers, domainId, runId, db)
		getAndSaveDomainIPv6(domainName, domainNameServers, domainId, runId, db)
		// Check SOA record
		getAndSaveDomainSOA(domainName, domainNameServers, domainId, runId, db)
		// Get DNSSEC info
		getAndSaveDNSSECinfo(domainName, domainNameServers, domainId, runId, db)
	}

}

func isIPInDontProbeList(ip net.IP) bool {
	var ipnet *net.IPNet
	for _, ipnet = range dontProbeList {
		if ipnet.Contains(ip) {
			fmt.Println("DONT PROBE LIST ip: ", ip, " found in: ", ipnet)
			return true
		}
	}
	return false
}
