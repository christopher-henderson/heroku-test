/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"github.com/mozilla/OneCRL-Tools/oneCRL"
	"io/ioutil"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	certPtr := flag.String("cert", "", "a DER or PEM encoded certificate file")
	revocationTypePtr := flag.String("type", "issuer-serial", "What type of revocation you want (options: issuer-serial, subject-pubkey)")
	checkValidityPtr := flag.String("checkvalidity", "no", "should validity be checked? (yes / no)")
	flag.Parse()

	var certData []byte
	if nil != certPtr && len(*certPtr) > 0 {
		// Get the cert from the args
		var err error
		certData, err = ioutil.ReadFile(*certPtr)
		check(err)
	}

	if len(certData) > 0 {
		// Maybe it's PEM; try to parse as PEM, if that fails, just use the bytes
		// We only care about the first block for now
		block, _ := pem.Decode(certData)
		if nil == block {
			panic(errors.New("There was a problem decoding the certificate"))
		}
		certData = block.Bytes

		cert, err := x509.ParseCertificate(certData)
		check(err)

		// Check to see if the cert is still valid (if it's not, we don't want
		// an entry
		if "yes" == *checkValidityPtr {
			if time.Now().After(cert.NotAfter) {
				panic(errors.New(fmt.Sprintf("Cert is no longer valid (NotAfter %v)", cert.NotAfter)))
			}
		}


		var record oneCRL.Record
		switch *revocationTypePtr {
		case "issuer-serial":
			issuerString := base64.StdEncoding.EncodeToString(cert.RawIssuer)

			if marshalled, err := asn1.Marshal(cert.SerialNumber); err == nil {
				serialString := base64.StdEncoding.EncodeToString(marshalled[2:])
				record = oneCRL.Record{IssuerName: issuerString, SerialNumber: serialString}
			}

		case "subject-pubkey":
			subjectString := base64.StdEncoding.EncodeToString(cert.RawSubject)

			if pubKeyData, err := x509.MarshalPKIXPublicKey(cert.PublicKey); err == nil {
				hash := sha256.Sum256(pubKeyData)
				base64EncodedHash := base64.StdEncoding.EncodeToString(hash[:])
				record = oneCRL.Record{Subject: subjectString, PubKeyHash: base64EncodedHash}
			}
		default:
		}

		if recordJson, err := json.MarshalIndent(record, "  ", "  "); nil == err {
			fmt.Printf("%s\n", recordJson)
		} else {
			panic(err)
		}
	}
}
