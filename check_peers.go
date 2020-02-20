package main

import (
	"errors"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	yaml "gopkg.in/yaml.v2"
)

var (
	nodeConfigFile = "node-config.yaml"
	scrapeFrom     = "https://adapools.org/peers"
)

// Usage:
// check_peers -config=node.yaml -scrapeFrom=peers.txt
func main() {
	// -config
	flag.StringVar(&nodeConfigFile, "config", nodeConfigFile, "jormungandr node yaml config file location")
	// -scrapeFrom
	flag.StringVar(&scrapeFrom, "scrapeFrom", scrapeFrom, "peer list file in yaml format")
	flag.Parse()

	fmt.Println("# Scraping peers from " + scrapeFrom)

	// workingPeers is list of working peers from adapools.org
	var workingPeers []Peer
	if err := scrapeWorkingPeers(&workingPeers); err != nil {
		log.Fatal(err)
	}

	fmt.Println("# Connecting to peers")

	// trustedPeers is list of successful connected peers
	var trustedPeers []Peer
	// connecting to peers and return good peers only
	for _, peer := range workingPeers {
		if peer.tcpping() {
			trustedPeers = append(trustedPeers, peer)
		}
	}

	if len(trustedPeers) < 1 {
		log.Fatal("empty peers")
	}

	// now we have list of good peers, lets replce the trusted_peers in yaml config file
	if err := updateTrustedPeers(&trustedPeers); err != nil {
		log.Fatal(err)
	}
	fmt.Println("# Working peers list has been updated in " + nodeConfigFile)
}

type Peer struct {
	Address, Id string
}

// we have got list of valid peers, lets tcpping to them and only return working peers
func (p *Peer) tcpping() bool {
	var (
		sourceStr = strings.Split(p.Address, "/") // eg: /ipv4/0.0.0.0/tcp/3000
		host      = sourceStr[2]
		port      = sourceStr[len(sourceStr)-1]
	)

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), time.Second)
	if err != nil {
		fmt.Printf("[ERR] %v\r\n", err)
		return false
	}

	// connection OK
	if conn != nil {
		defer conn.Close()
		fmt.Printf("[OK] %v\r\n", host)
		return true
	}

	return false
}

// Update connected working peers list in provided yaml config file
func updateTrustedPeers(trustedPeers *[]Peer) error {
	// read yaml config
	yamlFile, err := ioutil.ReadFile(nodeConfigFile)
	if err != nil {
		return fmt.Errorf("read yaml: %v", err)
	}

	// decode to map
	var y map[string]interface{}
	if err := yaml.Unmarshal(yamlFile, &y); err != nil {
		return fmt.Errorf("unmarshal yaml: %v", err)
	}

	// update p2p.trusted_peers in map
	y["p2p"].(map[interface{}]interface{})["trusted_peers"] = *trustedPeers

	// encode map
	b, err := yaml.Marshal(&y)
	if err != nil {
		return fmt.Errorf("marshal yaml: %v", err)
	}

	// write to yaml config
	if err := ioutil.WriteFile(nodeConfigFile, b, 0644); err != nil {
		return fmt.Errorf("io write yaml: %v", err)
	}

	return nil
}

// scrape peers list from given file or from https://adapools.org/peers by default
func scrapeWorkingPeers(workingPeers *[]Peer) error {
	var scrapedText string
	var scrapeFromHost bool = true

	// check if string is URL
	_, err := url.ParseRequestURI(scrapeFrom)
	if err != nil {
		// not a valid url, must be file path
		scrapeFromHost = false
	}

	if scrapeFromHost {
		// Request the HTML page.
		res, err := http.Get(scrapeFrom)
		if err != nil {
			return err
		}

		defer res.Body.Close()

		if res.StatusCode != 200 {
			return fmt.Errorf("web page status: %v", res.Status)
		}

		// Load the HTML document
		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			return fmt.Errorf("goquery: %v", err)
		}

		// Find the review items
		doc.Find("textarea.form-control").Each(func(i int, s *goquery.Selection) {
			scrapedText, _ = s.Html()
		})

	} else {

		b, err := ioutil.ReadFile(scrapeFrom)
		if err != nil {
			return err
		}
		scrapedText = string(b)
	}

	if scrapedText == "" {
		return errors.New("scraped peer list is empty")
	}

	scrapedText = `peers:
` + html.UnescapeString(scrapedText)

	var p map[string][]Peer
	if err = yaml.Unmarshal([]byte(scrapedText), &p); err != nil {
		return fmt.Errorf("unmarshal yaml: %v", err)
	}

	*workingPeers = p["peers"]

	return nil
}
