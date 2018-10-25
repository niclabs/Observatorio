package dnsUtils

import (
	"bytes"

	"github.com/miekg/dns"
	"database/sql"
	"github.com/niclabs/Observatorio/dbController"
	"strings"
	"time"
	"fmt"
	"net"
)

/* functions Less, doDDD, isDigit and dddToByte took from somewhere i don't remember TODO search source*/

// less returns <0 when a is less than b, 0 when they are equal and
// >0 when a is larger than b.
// The function orders names in DNSSEC canonical order: RFC 4034s section-6.1
//
// See http://bert-hubert.blogspot.co.uk/2015/10/how-to-do-fast-canonical-ordering-of.html
// for a blog article on this implementation, although here we still go label by label.
//
// The values of a and b are *not* lowercased before the comparison!
func Less(a, b string) int {
	i := 1
	aj := len(a)
	bj := len(b)
	for {
		ai, oka := dns.PrevLabel(a, i)
		bi, okb := dns.PrevLabel(b, i)
		if oka && okb {
			return 0
		}

		// sadly this []byte will allocate... TODO(miek): check if this is needed
		// for a name, otherwise compare the strings.
		ab := []byte(a[ai:aj])
		bb := []byte(b[bi:bj])
		doDDD(ab)
		doDDD(bb)

		res := bytes.Compare(ab, bb)
		if res != 0 {
			return res
		}

		i++
		aj, bj = ai, bi
	}
}
func doDDD(b []byte) {
	lb := len(b)
	for i := 0; i < lb; i++ {
		if i+3 < lb && b[i] == '\\' && isDigit(b[i+1]) && isDigit(b[i+2]) && isDigit(b[i+3]) {
			b[i] = dddToByte(b[i:])
			for j := i + 1; j < lb-3; j++ {
				b[j] = b[j+3]
			}
			lb -= 3
		}
	}
}
func isDigit(b byte) bool     { return b >= '0' && b <= '9' }
func dddToByte(s []byte) byte { return (s[1]-'0')*100 + (s[2]-'0')*10 + (s[3] - '0') }

func ExchangeWithRetry(m *dns.Msg, c *dns.Client, server []string)(*dns.Msg, time.Duration, error){
	var records *dns.Msg;
	var err error;
	var rtt time.Duration;
	for retry:=3; retry>0; retry-- {
		records, rtt, err = c.Exchange(m, server[retry%len(server)] +":53")
		if (err == nil){
			if (len(records.Answer)>0){

				break;
			}
		}else if(strings.IndexAny(err.Error(),"timeout")<0){//si el error no es timeout
			break;
		}
	}
	return records, rtt, err
}

func GetRecordSet(line string, t uint16, server []string, c *dns.Client)(*dns.Msg, time.Duration, error){
	m := new(dns.Msg)
	m.SetQuestion(line, t)
	return ExchangeWithRetry(m,c,server)
}

func manageError(err string, debug bool){
	if(debug){
		fmt.Println(err)
	}
}

func checkSOA(line string, servers []string, c *dns.Client)(*dns.Msg, error){
	var soa_records *dns.Msg;
	var err error;
	soa_records, _/*rtt*/, err = GetRecordSet(line, dns.TypeSOA, servers,c)
	return soa_records,err;
}

//check SOA
func cSoa(line string, run_id int, db *sql.DB, servers []string, c *dns.Client, debug bool){
	var domainid int
	domainid = dbController.SaveDomain(line, run_id, db)
	if (len(servers)!=0) {
		{
			SOA := false
			soa, err := checkSOA(line, servers,c)
			if (err != nil) {
				manageError(strings.Join([]string{"check soa", line, err.Error()}, ""), debug)

			} else {

				for _, soar := range soa.Answer {
					if _, ok := soar.(*dns.SOA); ok {
						SOA = true
					}
				}
			}
			dbController.SaveSoa(SOA, domainid, db)

		}
	}else{
	fmt.Println("Length of servers is 0")
	}
}


func CheckAvailability(domain string,ns *dns.NS, c *dns.Client)(bool, time.Duration, error){
	_, rtt, err := GetRecordSet(domain, dns.TypeA, []string{ns.Ns},c)
	if err != nil {
		return false, rtt, err
	}
	return true, rtt,nil

}

func FindKey(dnskeys *dns.Msg, rrsig *dns.RRSIG)(*dns.DNSKEY){
	var key *dns.DNSKEY
	for _, dnskey := range dnskeys.Answer {//Encontrar la llave que firma este RRSIG
		//RRset of type DNSKEY
		if dnskey1, ok := dnskey.(*dns.DNSKEY); ok {
			//DNSKEYset that signs the RRset to chase:
			if dnskey1.KeyTag() == rrsig.KeyTag {
				// RRSIG of the DNSKEYset that signs the RRset to chase:
				key = dnskey1
			}
		}
	}
	return key
}

func GetAAAARecords(line string, servers []string, c *dns.Client)([]net.IP, error){
	var aaaa_records *dns.Msg;
	var err error;
	aaaa_records, _, err = GetRecordSet(line,dns.TypeAAAA, servers,c)
	if(err!=nil){
		return nil, err
	}
	IPv6s:=[]net.IP{}
	for _,a := range aaaa_records.Answer{
		if a1, ok := a.(*dns.AAAA); ok{
			IPv6s = append(IPv6s,a1.AAAA)
		}
	}
	return IPv6s,nil
}

func GetARecords(line string, servers []string, c *dns.Client)([]net.IP, error){
	var a_records *dns.Msg;
	var err error;
	a_records, _, err = GetRecordSet(line,dns.TypeA, servers,c)
	if(err!=nil){
		return nil, err
	}
	IPv4s:=[]net.IP{}
	for _,a := range a_records.Answer{
		if a1, ok := a.(*dns.A); ok{
			IPv4s = append(IPv4s,a1.A)
		}
	}
	return IPv4s,nil
}

func GetRecordSetTCP(line string, t uint16, servers []string, c *dns.Client)(*dns.Msg, time.Duration, error){
	m := new(dns.Msg)
	m.SetQuestion(line, t)
	c.Net="tcp"
	return ExchangeWithRetry(m,c,servers)
}

func GetRecordSetWithDNSSEC(line string, t uint16, server string, c *dns.Client)(*dns.Msg, time.Duration, error){
	m := new(dns.Msg)
	m.SetQuestion(line, t)
	m.SetEdns0(4096,true)
	c= new(dns.Client)
	c.Net="tcp"
	return ExchangeWithRetry(m,c,[]string{server})
}

func GetRecordSetWithDNSSECformServer(line string, t uint16, server string, c *dns.Client)(*dns.Msg, time.Duration, error) {
	m := new(dns.Msg)
	m.SetQuestion(line, t)
	m.SetEdns0(4096,true)
	c= new(dns.Client)
	c.Net="tcp"
	return ExchangeWithRetry(m,c,[]string{server})
}

func GetRecursivityAndEDNS(line string, ns string, port string, c *dns.Client)(*dns.Msg,time.Duration, error){
	m := new(dns.Msg)
	m.SetEdns0(4096, true)
	m.SetQuestion(line, dns.TypeSOA)
	return ExchangeWithRetry(m,c,[]string{ns})
}

/*func IsIPInDontProbeList(ip net.IP)(bool){
	var ipnet *net.IPNet
	for _,ipnet=range dontProbeList{
		if(ipnet.Contains(ip)){
			fmt.Println("DONT PROBE LIST ip: ",ip," found in: ",ipnet)
			return true;
		}
	}
	return false;
}*/

func ZoneTransfer(line string, ns string )(chan *dns.Envelope, error){
	m := new(dns.Msg)
	m.Id = dns.Id()
	m.SetAxfr(line)
	t := new(dns.Transfer)
	zt, err := t.In(m, ns + ":53")
	if(err==nil){
		t.Close()
	}
	return zt, err
}