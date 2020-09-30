package dbController

import (
	"database/sql"
	"fmt"
	"github.com/miekg/dns"
	"strings"
	"time"
)

func CreateTables(db *sql.DB, drop bool) {
	DropTable("runs", db, drop)
	_, err := db.Exec("CREATE TABLE  IF NOT EXISTS runs ( id SERIAL PRIMARY KEY, tstmp timestamp, correct_run bool, duration int)")
	if err != nil {
		fmt.Println("OpenConnections", db.Stats())
		panic(err)
	}

	DropTable("domain", db, drop)
	_, err = db.Exec("CREATE TABLE  IF NOT EXISTS domain ( id SERIAL PRIMARY KEY, run_id integer REFERENCES runs(id),name varchar(253), soa bool, non_existence_status int, nsec bool, nsecok bool, nsec3 bool, nsec3ok bool, wildcard bool, dnssec_ok bool, ds_found bool, ds_ok bool, dnskey_found bool, dnskey_ok bool)")
	if err != nil {
		fmt.Println("OpenConnections", db.Stats())
		panic(err)
	}

	DropTable("nameserver", db, drop)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS nameserver ( id SERIAL PRIMARY KEY, run_id integer REFERENCES runs(id), domain_id  integer REFERENCES domain(id), name varchar(253), response bool, edns bool, recursivity bool, tcp bool, zone_transfer bool, loc_query bool)")
	if err != nil {
		fmt.Println("OpenConnections", db.Stats())
		panic(err)
	}
	DropTable("nameserver_ip", db, drop)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS nameserver_ip ( id SERIAL PRIMARY KEY, run_id integer REFERENCES runs(id), nameserver_id integer REFERENCES nameserver(id), ip inet, country varchar(30), asn varchar(10), dont_probe bool )")
	if err != nil {
		fmt.Println("OpenConnections", db.Stats())
		panic(err)
	}
	DropTable("domain_ip", db, drop)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS domain_ip ( id SERIAL PRIMARY KEY, run_id integer REFERENCES runs(id), domain_id integer REFERENCES domain(id), ip inet)")
	if err != nil {
		fmt.Println("OpenConnections", db.Stats())
		panic(err)
	}
	DropTable("dnskey", db, drop)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS dnskey ( id SERIAL PRIMARY KEY, run_id integer REFERENCES runs(id), domain_id integer REFERENCES domain(id), public_key varchar(4096), owner varchar(253), ttl integer, type integer, protocol integer, algorithm integer, keytag integer, DSok bool)")
	if err != nil {
		fmt.Println("OpenConnections", db.Stats())
		panic(err)
	}
	DropTable("rrsig", db, drop)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS rrsig (id SERIAL PRIMARY KEY, run_id integer REFERENCES runs(id), domain_id integer REFERENCES domain(id), owner varchar(253), type_covered varchar(5), algorithm integer, labels integer, ttl integer, signature_expiration varchar(50), signature_inception varchar(50), keytag integer, signers_name varchar(48), signature varchar(1024))")
	if err != nil {
		fmt.Println("OpenConnections", db.Stats())
		panic(err)
	}
	DropTable("nsec", db, drop)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS nsec ( id SERIAL PRIMARY KEY, run_id integer REFERENCES runs(id), domain_id integer REFERENCES domain(id), name varchar(253),  next_name varchar(253), ttl integer, rrsig_ok bool, cover bool, coverwc bool, iswc bool)")
	if err != nil {
		fmt.Println("OpenConnections", db.Stats())
		panic(err)
	}
	DropTable("nsec3", db, drop)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS nsec3 ( id SERIAL PRIMARY KEY, run_id integer REFERENCES runs(id), domain_id integer REFERENCES domain(id), hashed_name varchar(253),  next_hashed_name varchar(253), iterations integer, hash_algorithm integer, salt varchar(255), rrsig_ok bool, match bool, cover bool, coverwc bool, n3wc bool, key_found bool, verified bool, expired bool)")
	if err != nil {
		fmt.Println("OpenConnections", db.Stats())
		panic(err)
	}

	DropTable("ds", db, drop)

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS ds ( id SERIAL PRIMARY KEY, run_id integer REFERENCES runs(id), domain_id integer REFERENCES domain(id), algorithm int, hashed_name varchar(253) , key_tag integer, digest_type integer, digest varchar(255), ds_ok bool)")
	if err != nil {
		fmt.Println("OpenConnections", db.Stats())
		panic(err)
	}
}
func DropTable(table string, db *sql.DB, drop bool) {
	if drop {
		_, err := db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE")
		if err != nil {
			fmt.Println("OpenConnections", db.Stats())
			panic(err)
		}
	}
}
func NewRun(db *sql.DB) int {
	var run_id int
	err := db.QueryRow("INSERT INTO runs(tstmp) VALUES($1) RETURNING id", time.Now()).Scan(&run_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats())
		panic(err)
	}
	return run_id
}
func SaveCorrectRun(run_id int, duration int, correct bool, db *sql.DB) {
	_, err := db.Exec("UPDATE runs SET duration = $1, correct_run = $2 WHERE id = $3", int(duration/1000000000), correct, run_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " run_id", run_id)
		panic(err)
	}
}
func SaveDomain(line string, run_id int, db *sql.DB) int {
	var domainid int
	err := db.QueryRow("INSERT INTO domain(name, run_id) VALUES($1,$2) RETURNING id", line, run_id).Scan(&domainid)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), "domain name", line)
		if strings.Contains(err.Error(), "too many open files") {
			return SaveDomain(line, run_id, db)
		}
		panic(err)
	}
	return domainid
}
func SaveNSIP(nameserverid int, ip string, country string, asn string, dontProbe bool, run_id int, db *sql.DB) {
	_, err := db.Exec("INSERT INTO nameserver_ip(ip, nameserver_id,country, asn, dont_probe,run_id) VALUES($1::inet, $2,$3, $4, $5,$6)", ip, nameserverid, country, asn, dontProbe, run_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " nameserverid: ", nameserverid)
		panic(err)
	}
}
func SaveSoa(soa bool, domainid int, db *sql.DB) {
	_, err := db.Exec("UPDATE domain SET soa = $1 WHERE id = $2", soa, domainid)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " DomainId: ", domainid)
		panic(err)
	}
}
func SaveDNSKEY(dnskey *dns.DNSKEY, domain_id int, run_id int, db *sql.DB) {
	_, err := db.Exec("INSERT INTO dnskey(domain_id, public_key, owner, ttl, type, protocol, algorithm, keytag, run_id)VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)", domain_id, dnskey.PublicKey, dnskey.Hdr.Name, dnskey.Hdr.Ttl, dnskey.Hdr.Rrtype, dnskey.Protocol, dnskey.Algorithm, dnskey.KeyTag(), run_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " DomainId: ", domain_id)
		panic(err)
	}
}
func SaveRRSIG(rrsig *dns.RRSIG, domain_id int, run_id int, db *sql.DB) {

	var inception string = dns.TimeToString(rrsig.Inception)
	var expiration string = dns.TimeToString(rrsig.Expiration)
	_, err := db.Exec("INSERT INTO rrsig(domain_id, owner,  type_covered,  algorithm, labels, ttl, signature_expiration, signature_inception, keytag, signers_name, signature, run_id) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)", domain_id, rrsig.Hdr.Name, rrsig.TypeCovered, rrsig.Algorithm, rrsig.Labels, rrsig.OrigTtl, expiration, inception, rrsig.KeyTag, rrsig.SignerName, rrsig.Signature, run_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " DomainId: ", domain_id)
		fmt.Println(domain_id, rrsig.Hdr.Name, rrsig.TypeCovered, rrsig.Algorithm, rrsig.Labels, rrsig.OrigTtl, expiration, inception, rrsig.KeyTag, rrsig.SignerName, rrsig.Signature)
		panic(err)
	}
}
func SaveDS(domainid int, algorithm int, keyTag int, digestType int, digest string, run_id int, db *sql.DB) {
	_, err := db.Exec("INSERT INTO ds(domain_id, algorithm, key_tag, digest_type, digest, run_id)VALUES($1, $2, $3, $4, $5, $6)", domainid, algorithm, keyTag, digestType, digest, run_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " DomainId: ", domainid)
		panic(err)
	}
}
func SaveDomainIp(ip string, domainid int, run_id int, db *sql.DB) {
	_, err := db.Exec("INSERT INTO domain_ip(ip, domain_id, run_id) VALUES($1, $2, $3)", ip, domainid, run_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " DomainId: ", domainid)
		panic(err)
	}
}
func CreateNS(ns *dns.NS, domain_id int, run_id int, db *sql.DB, available bool) int {
	var nameserverid int

	err := db.QueryRow("INSERT INTO nameserver(name, domain_id, response, run_id) VALUES($1, $2, $3, $4) RETURNING id", ns.Ns, domain_id, available, run_id).Scan(&nameserverid)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " DomainId: ", domain_id)
		panic(err)
	}
	return nameserverid
}
func SaveNS(recursivity bool, EDNS bool, TCP bool, zone_transfer bool, loc_query bool, nameserverid int, db *sql.DB) {
	_, err := db.Exec("UPDATE nameserver SET recursivity = $1, edns = $2, tcp = $3, zone_transfer = $4, loc_query = $5 WHERE id = $6", recursivity, EDNS, TCP, zone_transfer, loc_query, nameserverid)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " nameserverid: ", nameserverid)
		panic(err)

	}
}
func SaveNsec(domainid int, name string, next_name string, ttl int, run_id int, db *sql.DB) int {
	var nsecid int
	err := db.QueryRow("INSERT INTO nsec(domain_id, name, next_name, ttl, run_id) VALUES($1, $2, $3, $4, $5)RETURNING id", domainid, name, next_name, ttl, run_id).Scan(&nsecid)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " DomainId: ", domainid)
		panic(err)
	}
	return nsecid
}
func SaveNsec3(domainid int, hashed_name string, next_hashed_name string, iterations int, hash_algorithm int, salt string, run_id int, db *sql.DB) int {
	var nsec3id int
	err := db.QueryRow("INSERT INTO nsec3(domain_id, hashed_name, next_hashed_name, iterations, hash_algorithm, salt, run_id) VALUES($1, $2, $3, $4, $5, $6, $7)RETURNING id", domainid, hashed_name, next_hashed_name, iterations, hash_algorithm, salt, run_id).Scan(&nsec3id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " DomainId: ", domainid)
		panic(err)
	}
	return nsec3id
}
func UpdateNonExistence(domainid int, non_existence_status int, db *sql.DB) {
	_, err := db.Exec("UPDATE domain SET non_existence_status = $1 WHERE id = $2", non_existence_status, domainid)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " DomainId: ", domainid)
		panic(err)
	}
}
func UpdateDomainNSECInfo(domain_id int, nsecok bool, nsec bool, wildcard bool, db *sql.DB) {
	_, err := db.Exec("UPDATE domain SET nsec = $1, nsecok=$2, wildcard=$3 WHERE id = $4", nsec, nsecok, wildcard, domain_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " DomainId: ", domain_id)
		panic(err)
	}
}
func UpdateDomainNSEC3Info(domain_id int, nsec3ok bool, nsec3 bool, wildcard bool, db *sql.DB) {
	_, err := db.Exec("UPDATE domain SET nsec3 = $1, nsec3ok=$2, wildcard=$3 WHERE id = $4", nsec3, nsec3ok, wildcard, domain_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " DomainId: ", domain_id)
		panic(err)
	}
}
func GetNonExistenceStatus(domain_id int, db *sql.DB) (string, int, error) {
	var name string
	var non_existence_status int
	err := db.QueryRow("SELECT name, non_existence_status FROM domain WHERE id=$1", domain_id).Scan(&name, &non_existence_status)
	return name, non_existence_status, err
}
func GetDomains(run_id int, db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT id FROM domain WHERE run_id=$1", run_id)

	return rows, err
}
func UpdateNSEC(rrsig_ok bool, cover bool, coverwc bool, iswc bool, nsec_id int, db *sql.DB) {
	_, err := db.Exec("UPDATE nsec SET rrsig_ok = $1, cover=$2, coverwc=$3, iswc=$4 WHERE id = $5", rrsig_ok, cover, coverwc, iswc, nsec_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " nsecId: ", nsec_id)
		panic(err)
	}
}
func GetNSEC3s(domain_id int, db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT rrsig_ok, match, cover, coverwc, n3wc FROM nsec3 where domain_id = $1", domain_id)
	return rows, err
}
func UpdateNSEC3(rrsig_ok bool, key_found bool, verified bool, expired bool, match bool, cover bool, coverwc bool, n3wc bool, nsec3_id int, db *sql.DB) {
	_, err := db.Exec("UPDATE nsec3 SET rrsig_ok = $1, match=$2, cover=$3, coverwc=$4, n3wc=$5, key_found=$6, verified=$7, expired=$8 WHERE id = $9", rrsig_ok, match, cover, coverwc, n3wc, key_found, verified, expired, nsec3_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " nsec3Id: ", nsec3_id)
		panic(err)
	}
}
func UpdateDomainDSInfo(domain_id int, ds_found bool, ds_ok bool, db *sql.DB) {
	_, err := db.Exec("UPDATE domain SET ds_found = $1, ds_ok=$2 WHERE id = $3", ds_found, ds_ok, domain_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " domain_id: ", domain_id)
		panic(err)
	}
}
func UpdateDomainDNSKEYInfo(domain_id int, dnskey_found bool, dnskey_ok bool, db *sql.DB) {
	_, err := db.Exec("UPDATE domain SET dnskey_found = $1, dnskey_ok=$2 WHERE id = $3", dnskey_found, dnskey_ok, domain_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " domain_id: ", domain_id)
		panic(err)
	}
}
func GetNSECsInfo(domain_id int, db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT rrsig_ok, cover, coverwc, iswc FROM nsec where domain_id = $1", domain_id)
	return rows, err
}
func GetDSInfo(domain_id int, db *sql.DB) (ds_found bool, ds_ok bool) {
	err := db.QueryRow("SELECT ds_found, ds_ok FROM domain WHERE id=$1", domain_id).Scan(&ds_found, &ds_ok)
	if err != nil {
		ds_found = false
		ds_ok = false
	}
	return
}
func GetDNSKEYInfo(domain_id int, db *sql.DB) (dnskey_found bool, dnskey_ok bool) {
	err := db.QueryRow("SELECT dnskey_found, dnskey_ok FROM domain WHERE id=$1", domain_id).Scan(&dnskey_found, &dnskey_ok)
	if err != nil {
		dnskey_found = false
		dnskey_ok = false
	}
	return
}
func UpdateDomainDNSSEC(domain_id int, dnssec_ok bool, db *sql.DB) {
	//TODO fix add dnssec_ok to domain table in  database(using wildcard_ok for now)
	_, err := db.Exec("UPDATE domain SET dnssec_ok = $1 WHERE id = $2", dnssec_ok, domain_id)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats(), " domain_id: ", domain_id)
		panic(err)
	}
}
func CountNSPerDomain(run_id int, db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT numNs, COUNT(domain_id) AS num FROM(SELECT domain_id, COUNT(nameserver.id) AS numNs FROM nameserver WHERE run_id=$1 GROUP BY domain_id) AS counts GROUP BY numNs ORDER BY numNs;", run_id)
	return rows, err
}
func CountASNPerDomain(run_id int, db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT COUNT(domain_id) as numDomains, asnCount FROM("+
		"SELECT NNN.domain_id, NCA.asnCount FROM("+
		"(Select domain_id, Count(nameserver.id) as numNs FROM nameserver WHERE run_id=$1 GROUP BY domain_id order by domain_id)AS NNN "+
		"JOIN (SELECT nameserver.domain_id, COUNT(distinct nameserver_ip.asn) AS asnCount FROM ("+
		"(SELECT * FROM nameserver WHERE run_id=$1)as nameserver "+
		"INNER JOIN (SELECT * FROM nameserver_ip WHERE run_id=$1)as nameserver_ip "+
		"ON nameserver.id=nameserver_ip.nameserver_id) GROUP BY nameserver.domain_id ORDER BY nameserver.domain_id) AS NCA "+
		"ON NNN.domain_id=NCA.domain_id))AS CNNN GROUP BY asnCount ORDER BY asnCount;", run_id)
	return rows, err
}
func CountCountryPerDomain(run_id int, db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(
		"SELECT COUNT(domain_id) as numDomains, countryCount FROM("+
			"SELECT NNN.domain_id, NCA.countryCount FROM("+
			"(Select domain_id, Count(nameserver.id) as numNs FROM (SELECT * FROM nameserver WHERE run_id=$1) AS nameserver GROUP BY domain_id order by domain_id)AS NNN "+
			"JOIN (SELECT nameserver.domain_id, COUNT(distinct nameserver_ip.country) AS countryCount  FROM "+
			"((SELECT * FROM nameserver WHERE run_id=$1) AS nameserver "+
			"INNER JOIN (SELECT * FROM nameserver_ip WHERE run_id=$1) AS nameserver_ip "+
			"ON nameserver.id=nameserver_ip.nameserver_id) GROUP BY nameserver.domain_id ORDER BY nameserver.domain_id) "+
			"AS NCA	ON NNN.domain_id=NCA.domain_id)"+
			")"+
			"AS CNNN GROUP BY countryCount ORDER BY countryCount ;", run_id)
	return rows, err
}
func CountNSCountryASNPerDomain(run_id int, db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("SELECT COUNT(domain_id) as numDomains, numNs, asnCount, countryCount FROM"+
		"(SELECT NNN.domain_id, NNN.numNS, NCA.asnCount, NCA.countryCount FROM("+
		"(Select domain_id, Count(nameserver.id) as numNs FROM (SELECT * FROM nameserver WHERE run_id=$1) AS nameserver GROUP BY domain_id order by domain_id) AS NNN "+
		"JOIN (SELECT nameserver.domain_id, COUNT(distinct nameserver_ip.country) AS countryCount, COUNT(distinct nameserver_ip.asn) AS asnCount FROM "+
		"((SELECT * FROM nameserver WHERE run_id=$1) AS nameserver INNER JOIN (SELECT * FROM nameserver_ip WHERE run_id=$1) AS nameserver_ip ON nameserver.id=nameserver_ip.nameserver_id) GROUP BY nameserver.domain_id ORDER BY nameserver.domain_id) AS NCA "+
		"ON NNN.domain_id=NCA.domain_id)"+
		")AS CNNN GROUP BY numNs, asnCount, countryCount ORDER BY numNs, asnCount, countryCount;", run_id)
	return rows, err
}
func CountDistinctNSWithIPv4(run_id int, db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("select count(distinct nameserver_id) from nameserver_ip where family(ip)=4 and run_id=$1;", run_id)
	return rows, err
}
func CountDistinctNSWithIPv6(run_id int, db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("select count(distinct nameserver_id) from nameserver_ip where family(ip)=6 and run_id=$1;", run_id)
	return rows, err
}
func CountDomainsWithCountNSIp(run_id int, db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(`
	SELECT COUNT(domain_id) AS numDomains, ipsCount, ipsv4Count, ipsv6Count FROM(
		SELECT domain_id, SUM(ipCount2) AS ipsCount, SUM(ipv4Count2) AS ipsv4Count ,SUM(ipv6Count2) AS ipsv6Count FROM(
			select domain_id, (CASE WHEN ipCount IS NULL THEN 0 ELSE ipCount END) AS ipCount2,
			(CASE WHEN ipv4Count IS NULL THEN 0 ELSE ipv4Count END) as ipv4Count2,
			(CASE WHEN ipv6Count IS NULL THEN 0 ELSE ipv6Count END) as ipv6Count2 from(
				(select id, domain_id from (select * from nameserver where run_id=$1) AS nameserver1) as nameserver LEFT JOIN
				(SELECT nameserver_id, COUNT(ip) AS ipCount, SUM(CASE family(ip) WHEN 4 THEN 1 ELSE 0 END) AS ipv4Count,
				SUM(CASE family(ip) WHEN 6 THEN 1 ELSE 0 END) AS ipv6Count FROM
					 (select * from nameserver_ip where run_id=$1) AS nameserver_ip GROUP BY(nameserver_id)
				)AS IPC
				ON nameserver.id=IPC.nameserver_id
			)AS CDN1
		)as CDN2 GROUP BY domain_id
	)AS CDN1
	GROUP BY ipsCount, ipsv4Count, ipsv6Count ORDER BY ipsCount, ipsv4Count, ipsv6Count;`, run_id)

	//"SELECT COUNT(domain_id) AS numDomains, ipsCount, ipsv4Count, ipsv6Count FROM (SELECT domain_id, SUM(ipCount) AS ipsCount, SUM(ipv4Count) AS ipsv4Count , SUM(ipv6Count) AS ipsv6Count FROM(nameserver left JOIN (SELECT nameserver_id, COUNT(ip) AS ipCount, SUM(CASE family(ip) WHEN 4 THEN 1 ELSE 0 END) AS ipv4Count, SUM(CASE family(ip) WHEN 6 THEN 1 ELSE 0 END) AS ipv6Count FROM nameserver_ip GROUP BY(nameserver_id)) AS IPC ON nameserver.id=IPC.nameserver_id) GROUP BY domain_id)AS CDN GROUP BY ipsCount, ipsv4Count, ipsv6Count ORDER BY ipsCount, ipsv4Count, ipsv6Count;")
	return rows, err
	//Count Distinct domains that have distinct counts of ips of nameservers.
}
func CountDomainsWithCountNSIPExclusive(run_id int, db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query("select sum(numDomains) as numDomains,family from(SELECT numDomains, (CASE  WHEN ipsv4Count>0 THEN (CASE WHEN ipsv6Count>0 THEN 10 ELSE 4 END) ELSE  (CASE WHEN ipsv6Count>0 THEN 6 ELSE 0 END)  END) AS family FROM(SELECT COUNT(domain_id) AS numDomains, ipsCount, ipsv4Count, ipsv6Count FROM "+
		"(SELECT domain_id, SUM(ipCount) AS ipsCount, SUM(ipv4Count) AS ipsv4Count , SUM(ipv6Count) AS ipsv6Count FROM"+
		"((select * from nameserver where run_id=$1) as nameserver1 left "+
		"JOIN (SELECT nameserver_id, COUNT(ip) AS ipCount, SUM(CASE family(ip) WHEN 4 THEN 1 ELSE 0 END) AS ipv4Count, SUM(CASE family(ip) WHEN 6 THEN 1 ELSE 0 END) AS ipv6Count FROM (select * from nameserver_ip where run_id=$1) as nameserver_ip GROUP BY(nameserver_id)) AS IPC "+
		"ON nameserver1.id=IPC.nameserver_id) GROUP BY domain_id)AS CDN GROUP BY ipsCount, ipsv4Count, ipsv6Count ORDER BY ipsCount, ipsv4Count, ipsv6Count) AS familyCount)as groupFamily GROUP BY family;", run_id)
	return rows, err
}
func GetRunTimestamp(run_id int, db *sql.DB) string {
	var ts string
	err := db.QueryRow("SELECT tstmp FROM runs WHERE id=$1", run_id).Scan(&ts)
	if err != nil {
		fmt.Println("OpenConnections", db.Stats())
		panic(err)
	}
	return ts
}
func CountDomainsWithDNSSEC(run_id int, db *sql.DB) (dnssec_wrong int, dnssec_ok int, no_dnssec int) {
	err := db.QueryRow(""+
		"select f.fid as dnssec_wrong, ok.cid as dnssec_ok, no.no_dnssec  from "+
		"(select count(id)as no_dnssec from domain where dnskey_found=false or dnskey_found is null and run_id=$1)as no,"+
		"(select count(id)as fid from domain where dnskey_found=true and dnssec_ok=false and run_id=$1)as f, "+
		"(select count(id)as cid from domain where dnssec_ok=true and run_id=$1)as ok;", run_id).Scan(&dnssec_wrong, &dnssec_ok, &no_dnssec)
	if err != nil {
		return 0, 0, 0
	}
	return
}
func CountDomainsWithDNSSECErrors(run_id int, db *sql.DB) (denial_proof int, dnskey_validation int, ds_validation int) {
	query := `SELECT Denial.count as denial_proof, DNSKEY.count AS dnskey_validation, DS.count AS ds_validation from
		(select count(id) from domain where dnskey_found=true and run_id=$1 AND dnssec_ok=false AND (((nsec=false OR nsec IS NULL) AND (nsec3=false OR nsec3 IS NULL))OR((nsecok=false OR nsec IS NULL)AND (nsec3ok=false OR nsec3 IS NULL))))AS Denial,
		(select count(id) from domain where dnskey_found=true and run_id=$1 AND dnssec_ok=false AND dnskey_ok=false) AS DNSKEY,
		(select count(id) from domain where dnskey_found=true and run_id=$1 AND dnssec_ok=false AND (ds_found=false OR ds_ok=false)) AS DS;`
	err := db.QueryRow(query, run_id).Scan(&denial_proof, &dnskey_validation, &ds_validation)
	if err != nil {
		return 0, 0, 0
	}
	return
}
func CountNameserverCharacteristics(run_id int, db *sql.DB) (recursivity int, no_recursivity int, edns int, no_edns int, tcp int, no_tcp int, zone_transfer int, no_zone_transfer int, loc_query int, no_loc_query int) {
	query := `SELECT
		    SUM(CASE WHEN edns = true then 1 ELSE 0 END) as edns, SUM(CASE WHEN edns = false then 1 ELSE 0 END)  as no_edns,
		    SUM(CASE WHEN recursivity = false then 1 ELSE 0 END) as no_recursivity, SUM(CASE WHEN recursivity = true then 1 ELSE 0 END)  as recursivity,
		    SUM(CASE WHEN tcp = false then 1 ELSE 0 END) as no_tcp, SUM(CASE WHEN tcp = true then 1 ELSE 0 END)  as tcp,
		    SUM(CASE WHEN zone_transfer = false then 1 ELSE 0 END) as no_zone_transfer, SUM(CASE WHEN zone_transfer = true then 1 ELSE 0 END)  as zone_transfer,
		    SUM(CASE WHEN loc_query = false then 1 ELSE 0 END) as no_loc_query, SUM(CASE WHEN loc_query = true then 1 ELSE 0 END)  as loc_query
		from (select * from nameserver where run_id=$1 and response=true) as NS;`
	err := db.QueryRow(query, run_id).Scan(&edns, &no_edns, &no_recursivity, &recursivity, &no_tcp, &tcp, &no_zone_transfer, &zone_transfer, &no_loc_query, &loc_query)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	}
	return
}

/*func getNSs(run_id int, db *sql.DB){
	query:= `SELECT
		    SUM(CASE WHEN edns = true then 1 ELSE 0 END) as edns, SUM(CASE WHEN edns = false then 1 ELSE 0 END)  as no_edns,
		    SUM(CASE WHEN recursivity = false then 1 ELSE 0 END) as no_recursivity, SUM(CASE WHEN recursivity = true then 1 ELSE 0 END)  as recursivity,
		    SUM(CASE WHEN tcp = false then 1 ELSE 0 END) as no_tcp, SUM(CASE WHEN tcp = true then 1 ELSE 0 END)  as tcp,
		    SUM(CASE WHEN zone_transfer = false then 1 ELSE 0 END) as no_zone_transfer, SUM(CASE WHEN zone_transfer = true then 1 ELSE 0 END)  as zone_transfer,
		    SUM(CASE WHEN loc_query = false then 1 ELSE 0 END) as no_loc_query, SUM(CASE WHEN loc_query = true then 1 ELSE 0 END)  as loc_query
		from (select * from nameserver where run_id=$1 and response=true) as NS;`
}*/
