package util

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/csv"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/mail"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/jordan-wright/email"
)

var (
	firstNameRegex = regexp.MustCompile(`(?i)first[\s_-]*name`)
	lastNameRegex  = regexp.MustCompile(`(?i)last[\s_-]*name`)
	emailRegex     = regexp.MustCompile(`(?i)email`)
	positionRegex  = regexp.MustCompile(`(?i)position`)
)

// ParseMail takes in an HTTP Request and returns an Email object
// TODO: This function will likely be changed to take in a []byte
func ParseMail(r *http.Request) (email.Email, error) {
	e := email.Email{}
	m, err := mail.ReadMessage(r.Body)
	if err != nil {
		fmt.Println(err)
	}
	body, err := ioutil.ReadAll(m.Body)
	e.HTML = body
	return e, err
}

// ParseCSV contains the logic to parse the user provided csv file containing Target entries
func ParseCSV(r *http.Request) ([]models.Target, error) {
	mr, err := r.MultipartReader()
	ts := []models.Target{}
	if err != nil {
		return ts, err
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		// Skip the "submit" part
		if part.FileName() == "" {
			continue
		}
		defer part.Close()
		reader := csv.NewReader(part)
		reader.TrimLeadingSpace = true
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		fi := -1
		li := -1
		ei := -1
		pi := -1
		fn := ""
		ln := ""
		ea := ""
		ps := ""
		for i, v := range record {
			switch {
			case firstNameRegex.MatchString(v):
				fi = i
			case lastNameRegex.MatchString(v):
				li = i
			case emailRegex.MatchString(v):
				ei = i
			case positionRegex.MatchString(v):
				pi = i
			}
		}
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if fi != -1 && len(record) > fi {
				fn = record[fi]
			}
			if li != -1 && len(record) > li {
				ln = record[li]
			}
			if ei != -1 && len(record) > ei {
				csvEmail, err := mail.ParseAddress(record[ei])
				if err != nil {
					continue
				}
				ea = csvEmail.Address
			}
			if pi != -1 && len(record) > pi {
				ps = record[pi]
			}
			t := models.Target{
				BaseRecipient: models.BaseRecipient{
					FirstName: fn,
					LastName:  ln,
					Email:     ea,
					Position:  ps,
				},
			}
			ts = append(ts, t)
		}
	}
	return ts, nil
}

// CheckAndCreateSSL is a helper to setup self-signed certificates for the administrative interface.
func CheckAndCreateSSL(cp string, kp string) error {
	// Check whether there is an existing SSL certificate and/or key, and if so, abort execution of this function
	if _, err := os.Stat(cp); !os.IsNotExist(err) {
		return nil
	}
	if _, err := os.Stat(kp); !os.IsNotExist(err) {
		return nil
	}

	log.Infof("Creating new self-signed certificates for administration interface")

	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)

	notBefore := time.Now()
	// Generate a certificate that lasts for 10 years
	notAfter := notBefore.Add(10 * 365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	if err != nil {
		return fmt.Errorf("TLS Certificate Generation: Failed to generate a random serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Gophish"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		return fmt.Errorf("TLS Certificate Generation: Failed to create certificate: %s", err)
	}

	certOut, err := os.Create(cp)
	if err != nil {
		return fmt.Errorf("TLS Certificate Generation: Failed to open %s for writing: %s", cp, err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.OpenFile(kp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("TLS Certificate Generation: Failed to open %s for writing", kp)
	}

	b, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("TLS Certificate Generation: Unable to marshal ECDSA private key: %v", err)
	}

	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	keyOut.Close()

	log.Info("TLS Certificate Generation complete")
	return nil
}

// IsLocalBusinessTime tells if the given UTC time is within the given business hours
// defined in "AM/PM" format (tz time zone is also taken into account)
func IsLocalBusinessTime(utcTime time.Time, startTime string, endTime string, tz string) bool {
	var loc *time.Location
	var err error

	if tz != "" {
		loc, err = time.LoadLocation(tz)

		if err != nil {
			log.Warnf("%s: couldn't parse time-zone (assuming UTC instead)", err)
			loc, _ = time.LoadLocation("UTC")
		}
	} else {
		loc, _ = time.LoadLocation("UTC")
	}

	yearAndDate := utcTime.Format("2006-01-02")
	sTime, err := time.ParseInLocation("2006-01-02 3:04 PM", yearAndDate+" "+startTime, loc)

	if err != nil {
		log.Warnf("%s: couldn't parse start time", err)
		return false
	}

	eTime, err := time.ParseInLocation("2006-01-02 3:04 PM", yearAndDate+" "+endTime, loc)

	if err != nil {
		log.Warnf("%s: couldn't parse end time", err)
		return false
	}

	return utcTime.After(sTime.UTC()) && utcTime.Before(eTime.UTC())
}

// GenerateSecureKey creates a secure key to use
// as an API key
func GenerateSecureKey() string {
	// Inspired from gorilla/securecookie
	k := make([]byte, 32)
	io.ReadFull(rand.Reader, k)
	return fmt.Sprintf("%x", k)
}

// GenerateUsername generates a pseudo-unique username from the given combination of full name and email.
// Returns an empty string if both params are blank.
func GenerateUsername(fullname, email string) string {
	username := strings.Replace(strings.ToLower(fullname), " ", "", -1)

	if username == "" && email != "" {
		username = email[0:strings.LastIndex(email, "@")]
	}

	if username == "" {
		return username
	}

	return username + strconv.Itoa(len(email))
}

// IsEmail tells if the given string is a valid email address
func IsEmail(s string) bool {
	pattern := "^(((([a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+(\\.([a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+)*)|((\\x22)((((\\x20|\\x09)*(\\x0d\\x0a))?(\\x20|\\x09)+)?(([\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x7f]|\\x21|[\\x23-\\x5b]|[\\x5d-\\x7e]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(\\([\\x01-\\x09\\x0b\\x0c\\x0d-\\x7f]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}]))))*(((\\x20|\\x09)*(\\x0d\\x0a))?(\\x20|\\x09)+)?(\\x22)))@((([a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(([a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])([a-zA-Z]|\\d|-|\\.|_|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*([a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.)+(([a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(([a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])([a-zA-Z]|\\d|-|_|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*([a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.?$"
	return regexp.MustCompile(pattern).MatchString(s)
}

// IsValidDomain tells if the given string is a valid domain name
func IsValidDomain(domain string) bool {
	return regexp.
		MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z]{2,24})$`).
		MatchString(domain)
}
