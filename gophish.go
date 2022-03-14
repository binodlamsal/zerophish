package main

/*
gophish - Open-Source Phishing Framework

The MIT License (MIT)

Copyright (c) 2013 Jordan Wright

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	"github.com/jinzhu/gorm"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/NYTimes/gziphandler"
	"github.com/binodlamsal/zerophish/auth"
	"github.com/binodlamsal/zerophish/config"
	"github.com/binodlamsal/zerophish/controllers"
	"github.com/binodlamsal/zerophish/encryption"
	log "github.com/binodlamsal/zerophish/logger"
	"github.com/binodlamsal/zerophish/mailer"
	"github.com/binodlamsal/zerophish/models"
	"github.com/binodlamsal/zerophish/util"
	"github.com/binodlamsal/zerophish/worker"
	"github.com/gorilla/handlers"
	"github.com/howeyc/gopass"
)

var (
	configPath     = kingpin.Flag("config", "Location of config.json.").Default("./config.json").String()
	disableMailer  = kingpin.Flag("disable-mailer", "Disable the mailer (for use with multi-system deployments)").Bool()
	encryptApiKeys = kingpin.Flag("encrypt-api-keys", "Encrypt all unencrypted API keys and exit").Bool()
	decryptApiKeys = kingpin.Flag("decrypt-api-keys", "Decrypt all encrypted API keys and exit").Bool()
	encryptEmails  = kingpin.Flag("encrypt-emails", "Encrypt all unencrypted emails and exit").Bool()
	decryptEmails  = kingpin.Flag("decrypt-emails", "Decrypt all encrypted emails and exit").Bool()
)

func main() {
	// Setup encryption
	const pipe = "input.pipe"

	if os.Getenv("ENCRYPTION_DISABLE") == "" {
		encKey := os.Getenv("ENCRYPTION_KEY")

		if encKey != "" {
			if len(encKey) == 32 {
				if err := encryption.SetKey([]byte(encKey)); err != nil {
					log.Fatal(err)
				}
			} else if len(encKey) == 64 {
				key, err := hex.DecodeString(encKey)

				if err != nil {
					log.Fatal(err)
				}

				if err := encryption.SetKey(key); err != nil {
					log.Fatal(err)
				}
			} else {
				log.Fatal(errors.New("wrong encryption key length (must be 32 character string or 64 bytes hex-encoded string)"))
			}
		} else if input, err := os.OpenFile(pipe, os.O_RDONLY, os.ModeNamedPipe); err == nil {
			passphrase, err := bufio.NewReader(input).ReadBytes('\n')

			if err == nil {
				h := sha256.New()
				h.Write(passphrase[:len(passphrase)-1])
				key := h.Sum(nil)

				if err := encryption.SetKey(key); err != nil {
					os.Remove(pipe)
					log.Fatal(err)
				}
			}

			os.Remove(pipe)
		} else {
			fmt.Printf("Enter passphrase: ")
			passphrase, err := gopass.GetPasswdMasked()

			if err != nil {
				log.Fatal(err)
			}

			h := sha256.New()
			h.Write(passphrase)
			key := h.Sum(nil)

			if err := encryption.SetKey(key); err != nil {
				log.Fatal(err)
			}
		}

		log.Info("Encryption is enabled")
	} else {
		encryption.Disabled = true
		log.Info("Encryption is disabled")
	}

	// Load the version

	version, err := ioutil.ReadFile("./VERSION")
	if err != nil {
		log.Fatal(err)
	}
	kingpin.Version(string(version))

	// Parse the CLI flags and load the config
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Parse()

	// Load the config
	err = config.LoadConfig(*configPath)

	if err != nil {
		log.Fatal(err)
	}

	config.Version = string(version)

	// Setup logging
	err = log.Setup()

	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup the global variables and settings
	err = models.Setup()

	if err != nil {
		log.Fatal(err)
	}

	if *encryptApiKeys {
		models.EncryptApiKeys()
		return
	} else if *decryptApiKeys {
		models.DecryptApiKeys()
		return
	}

	if *encryptEmails {
		if encryption.Disabled {
			log.Fatal("Unable to encrypt emails because encryption is disabled.")
		}

		models.EncryptUserEmails()
		models.EncryptTargetEmails()
		models.EncryptResultEmails()
		models.EncryptRequestEmails()
		models.EncryptEventEmails()
		return
	} else if *decryptEmails {
		if encryption.Disabled {
			log.Fatal("Unable to decrypt emails because encryption is disabled.")
		}

		models.DecryptUserEmails()
		models.DecryptTargetEmails()
		models.DecryptResultEmails()
		models.DecryptRequestEmails()
		models.DecryptEventEmails()
		return
	}

	// Validate encryption key by attempting to retrieve a single user record from the database
	if _, err := models.GetUser(1); err != nil && err != gorm.ErrRecordNotFound {
		log.Fatal(err)
	}

	if os.Getenv("CACHE_DISABLE") == "" {
		models.WarmUpCache()
	}

	w := worker.New()
	controllers.SetWorker(w)
	go w.Start()

	// Provide the option to disable the built-in mailer
	if !*disableMailer {
		go mailer.Mailer.Start(ctx)
	}

	// Unlock any maillogs that may have been locked for processing
	// when Gophish was last shutdown.
	err = models.UnlockAllMailLogs()
	if err != nil {
		log.Fatal(err)
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	// Start the web servers
	go func() {
		defer wg.Done()
		gzipWrapper, _ := gziphandler.NewGzipLevelHandler(gzip.BestCompression)
		adminHandler := gzipWrapper(controllers.CreateAdminRouter())
		auth.Store.Options.Secure = config.Conf.AdminConf.UseTLS || os.Getenv("VIA_PROXY") != ""
		if config.Conf.AdminConf.UseTLS { // use TLS for Admin web server if available
			err := util.CheckAndCreateSSL(config.Conf.AdminConf.CertPath, config.Conf.AdminConf.KeyPath)
			if err != nil {
				log.Fatal(err)
			}
			log.Infof("Starting admin server at https://%s", config.Conf.AdminConf.ListenURL)
			log.Info(http.ListenAndServeTLS(config.Conf.AdminConf.ListenURL, config.Conf.AdminConf.CertPath, config.Conf.AdminConf.KeyPath,
				handlers.CombinedLoggingHandler(log.Writer(), adminHandler)))
		} else {
			log.Infof("Starting admin server at http://%s", config.Conf.AdminConf.ListenURL)
			log.Info(http.ListenAndServe(config.Conf.AdminConf.ListenURL, handlers.CombinedLoggingHandler(os.Stdout, adminHandler)))
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		phishHandler := gziphandler.GzipHandler(controllers.CreatePhishingRouter())
		if config.Conf.PhishConf.UseTLS { // use TLS for Phish web server if available
			log.Infof("Starting phishing server at https://%s", config.Conf.PhishConf.ListenURL)
			log.Info(http.ListenAndServeTLS(config.Conf.PhishConf.ListenURL, config.Conf.PhishConf.CertPath, config.Conf.PhishConf.KeyPath,
				handlers.CombinedLoggingHandler(log.Writer(), phishHandler)))
		} else {
			log.Infof("Starting phishing server at http://%s", config.Conf.PhishConf.ListenURL)
			log.Fatal(http.ListenAndServe(config.Conf.PhishConf.ListenURL, handlers.CombinedLoggingHandler(os.Stdout, phishHandler)))
		}
	}()
	wg.Wait()
}
