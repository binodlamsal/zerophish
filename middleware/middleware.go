package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/everycloud-technologies/phishing-simulation/auth"
	"github.com/everycloud-technologies/phishing-simulation/bakery"
	ctx "github.com/everycloud-technologies/phishing-simulation/context"
	log "github.com/everycloud-technologies/phishing-simulation/logger"
	"github.com/everycloud-technologies/phishing-simulation/models"
	"github.com/everycloud-technologies/phishing-simulation/notifier"
	"github.com/everycloud-technologies/phishing-simulation/usersync"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
)

var CSRFExemptPrefixes = []string{
	"/api",
}

func CSRFExceptions(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, prefix := range CSRFExemptPrefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				r = csrf.UnsafeSkipCheck(r)
				break
			}
		}
		handler.ServeHTTP(w, r)
	}
}

// GetContext wraps each request in a function which fills in the context for a given request.
// This includes setting the User and Session keys and values as necessary for use in later functions.
func GetContext(handler http.Handler) http.HandlerFunc {
	// Set the context here
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse the request form
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing request", http.StatusInternalServerError)
		}
		// Set the context appropriately here.
		// Set the session
		session, _ := auth.Store.Get(r, "gophish")
		// Put the session in the context so that we can
		// reuse the values in different handlers
		r = ctx.Set(r, "session", session)
		if id, ok := session.Values["id"]; ok {
			u, err := models.GetUser(id.(int64))
			if err != nil {
				r = ctx.Set(r, "user", nil)
			} else {
				(&u).DecryptApiKey()
				r = ctx.Set(r, "user", u)
			}
		} else {
			r = ctx.Set(r, "user", nil)
		}
		handler.ServeHTTP(w, r)
		// Remove context contents
		ctx.Clear(r)
	}
}

func RequireAPIKey(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "1000")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
			return
		}
		r.ParseForm()
		ak := r.Form.Get("api_key")
		// If we can't get the API key, we'll also check for the
		// Authorization Bearer token
		if ak == "" {
			tokens, ok := r.Header["Authorization"]
			if ok && len(tokens) >= 1 {
				ak = tokens[0]
				ak = strings.TrimPrefix(ak, "Bearer ")
			}
		}
		if ak == "" {
			JSONError(w, 400, "API Key not set")
			return
		}
		u, err := models.GetUserByAPIKey(ak)
		if err != nil {
			JSONError(w, 400, "Invalid API Key")
			return
		}
		r = ctx.Set(r, "user_id", u.Id)
		r = ctx.Set(r, "api_key", ak)
		handler.ServeHTTP(w, r)
	}
}

func RequireIP(ip string) func(http.Handler) http.HandlerFunc {
	return func(handler http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ipAddr, _, err := net.SplitHostPort(r.RemoteAddr)

			if err != nil {
				log.Errorf("Denied access - %s", err.Error())
				JSONError(w, 403, "Access denied")
				return
			}

			if ipAddr != ip && ipAddr != "127.0.0.1" {
				log.Errorf("Denied access for IP %s", ipAddr)
				JSONError(w, 403, "Access denied")
				return
			}

			handler.ServeHTTP(w, r)
		}
	}
}

// RequireRoles enforces user role id to be one among the given role ids
func RequireRoles(rids []int64) func(http.Handler) http.HandlerFunc {
	return func(handler http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var uid int64
			user := ctx.Get(r, "user")

			if user != nil {
				uid = user.(models.User).Id
			} else {
				uid = ctx.Get(r, "user_id").(int64)
			}

			role, err := models.GetUserRole(uid)

			if err != nil || !role.IsOneOf(rids) {
				JSONError(w, 403, "Access denied")
				return
			}

			handler.ServeHTTP(w, r)
		}
	}
}

// RequireLogin is a simple middleware which checks to see if the user is currently logged in.
// If not, the function returns a 302 redirect to the login page.
func RequireLogin(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u := ctx.Get(r, "user"); u != nil {
			if u.(models.User).IsLMSUser() {
				flash(w, r, "info", "LMS users are not allowed to access the awareness platform.")
				http.Redirect(w, r, "logout", 302)
				return
			}

			handler.ServeHTTP(w, r)
		} else {
			q := r.URL.Query()
			q.Set("next", r.URL.Path)
			http.Redirect(w, r, fmt.Sprintf("/login?%s", q.Encode()), 302)
		}
	}
}

// SSO handles user authentication via encrypted CHOCOLATECHIPSSL cookie.
// If an email address extracted from such cookie belongs to an existing user
// then a session is created for that user.
func SSO(handler http.Handler) http.HandlerFunc {
	roles := map[string]int{
		"administrator":           models.Administrator,
		"Partner":                 models.Partner,
		"Child User":              models.ChildUser,
		"LMS User":                models.LMSUser,
		"Security Awareness User": models.Customer,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		logoutWithError := func(err error) {
			log.Error(err)
			http.Redirect(w, r, "logout", 302)
		}

		if u := ctx.Get(r, "user"); u == nil {
			cookie, err := r.Cookie("CHOCOLATECHIPSSL")

			if err != nil {
				logoutWithError(err)
				return
			}

			c, err := bakery.ParseCookie(cookie.Value)

			if err != nil {
				if err == bakery.ErrUnknownUserRole {
					flash(w, r, "info", "Please contact support to access the awareness platform.")
				}

				logoutWithError(err)
				return
			}

			if !c.IsChocolateChip {
				logoutWithError(errors.New("Bad type of SSO cookie"))
				return
			}

			user := models.User{}
			user, err = models.GetUserByUsername(c.Email)

			if gorm.IsRecordNotFoundError(err) {
				rid, ok := roles[c.Role]

				if !ok {
					logoutWithError(errors.New("Could not determine user role from the SSO cookie"))
					return
				}

				newUser, err := models.CreateUser(c.User, "", c.Email, "qwerty", int64(rid), 0)

				if err != nil {
					logoutWithError(err)
					return
				}

				user = *newUser
				err = user.SetBakeryUserID(c.BakeryID)

				if err != nil {
					log.Error(err)
					return
				}

				if os.Getenv("USERSYNC_DISABLE") == "" {
					_, err := usersync.PushUser(
						user.Id,
						user.Username,
						user.Email, "", "",
						int64(rid),
						models.GetUserBakeryID(user.Partner),
						true,
					)

					if err != nil {
						logoutWithError(fmt.Errorf("Could not sync user data with the main server - %s", err.Error()))
						return
					}
				}

				if os.Getenv("DONT_NOTIFY_USERS") == "" &&
					(int64(rid) == models.Partner || int64(rid) == models.Customer) {
					partner := false

					if int64(rid) == models.Partner {
						partner = true
					}

					notifier.SendWelcomeEmail(user.Email, user.Username, user.Username, partner)
				}
			} else if err != nil {
				logoutWithError(fmt.Errorf("User lookup failed - %s", err.Error()))
				return
			} else {
				user.LastLoginAt = time.Now().UTC()
				models.PutUser(&user)
			}

			session := ctx.Get(r, "session").(*sessions.Session)
			session.Values["id"] = user.Id
			session.Save(r, w)
			http.Redirect(w, r, r.URL.Path, 302)
		} else if _, err := r.Cookie("CHOCOLATECHIPSSL"); err != nil {
			http.Redirect(w, r, "logout", 302)
		} else {
			handler.ServeHTTP(w, r)
		}
	}
}

// JSONError returns an error in JSON format with the given
// status code and message
func JSONError(w http.ResponseWriter, c int, m string) {
	cj, _ := json.MarshalIndent(models.Response{Success: false, Message: m}, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	fmt.Fprintf(w, "%s", cj)
}

func flash(w http.ResponseWriter, r *http.Request, t string, m string) {
	session := ctx.Get(r, "session").(*sessions.Session)

	session.AddFlash(models.Flash{
		Type:    t,
		Message: m,
	})

	session.Save(r, w)
}
