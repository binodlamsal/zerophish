package controllers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/everycloud-technologies/phishing-simulation/job"

	"github.com/PuerkitoBio/goquery"
	"github.com/everycloud-technologies/phishing-simulation/auth"
	"github.com/everycloud-technologies/phishing-simulation/config"
	ctx "github.com/everycloud-technologies/phishing-simulation/context"
	log "github.com/everycloud-technologies/phishing-simulation/logger"
	"github.com/everycloud-technologies/phishing-simulation/models"
	"github.com/everycloud-technologies/phishing-simulation/notifier"
	"github.com/everycloud-technologies/phishing-simulation/usersync"
	"github.com/everycloud-technologies/phishing-simulation/util"
	"github.com/everycloud-technologies/phishing-simulation/worker"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/jordan-wright/email"
	"github.com/sirupsen/logrus"
)

// Worker is the worker that processes phishing events and updates campaigns.
var Worker *worker.Worker

func init() {
	Worker = worker.New()
	go Worker.Start()
}

// API (/api/reset) resets a user's API key
func API_Reset(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "POST":
		u := ctx.Get(r, "user").(models.User)
		u.ApiKey = util.GenerateSecureKey()
		err := models.PutUser(&u)
		if err != nil {
			http.Error(w, "Error setting API Key", http.StatusInternalServerError)
		} else {
			(&u).DecryptApiKey()
			JSONResponse(w, models.Response{Success: true, Message: "API Key successfully reset!", Data: u.PlainApiKey}, http.StatusOK)
		}
	}
}

// API_Campaigns returns a list of campaigns if requested via GET.
// If requested via POST, API_Campaigns creates a new campaign and returns a reference to it.
func API_Campaigns(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		cs, err := models.GetCampaigns(ctx.Get(r, "user_id").(int64))
		if err != nil {
			log.Error(err)
		}
		JSONResponse(w, cs, http.StatusOK)
	//POST: Create a new campaign and return it as JSON
	case r.Method == "POST":
		u, err := models.GetUser(ctx.Get(r, "user_id").(int64))

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		if !u.CanCreateCampaign() {
			JSONResponse(
				w, models.Response{
					Success: false,
					Message: `You’re currently using a free one-off phish subscription. If you’d like to run more campaigns, please <a href="https://www.everycloud.com/talk-us" target="_blank">contact us</a> to upgrade.`,
				},
				http.StatusConflict)
			return
		}

		c := models.Campaign{}
		// Put the request into a campaign
		err = json.NewDecoder(r.Body).Decode(&c)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON structure"}, http.StatusBadRequest)
			return
		}

		proto := "http://"

		if config.Conf.PhishConf.UseTLS || os.Getenv("VIA_PROXY") != "" {
			proto = "https://"
		}

		if os.Getenv("PHISHING_DOMAIN") != "" {
			c.URL = proto + os.Getenv("PHISHING_DOMAIN")
		} else {
			parsedURL, err := url.Parse(proto + r.Host)

			if err != nil {
				JSONResponse(w, models.Response{Success: false, Message: "Could not determine hostname"}, http.StatusInternalServerError)
				return
			}

			c.URL = proto + parsedURL.Hostname()
		}

		err = models.PostCampaign(&c, ctx.Get(r, "user_id").(int64))
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		// If the campaign is scheduled to launch immediately, send it to the worker.
		// Otherwise, the worker will pick it up at the scheduled time
		if c.Status == models.CAMPAIGN_IN_PROGRESS {
			go Worker.LaunchCampaign(c)
		}
		JSONResponse(w, c, http.StatusCreated)
	}
}

// API_Users returns a list of Users if requested via GET.
func API_Users(w http.ResponseWriter, r *http.Request) {
	type userWithExtraProps struct {
		models.User
		Role         string               `json:"role"`
		Subscription *models.Subscription `json:"subscription"`
		AvatarId     int64                `json:"avatar_id"`
	}

	type response []userWithExtraProps
	resp := response{}

	switch {
	case r.Method == "GET":
		users, err := models.GetUsers(ctx.Get(r, "user_id").(int64))

		if err != nil {
			log.Error(err)
		}

		for i := 0; i < len(users); i++ {
			user := users[i]
			var roleName string
			role, err := models.GetUserRole(user.Id)

			if err != nil {
				roleName = "Unknown"
			} else {
				roleName = role.DisplayName()
			}

			var aid int64

			if a := user.GetAvatar(); a != nil {
				aid = a.Id
			}

			resp = append(resp, userWithExtraProps{user, roleName, user.GetSubscription(), aid})
		}

		JSONResponse(w, resp, http.StatusOK)

	case r.Method == "POST":
		user := ctx.Get(r, "user").(models.User)
		role, err := models.GetUserRole(user.Id)

		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		ud := struct {
			Username string `json:"username"`
			FullName string `json:"full_name"`
			Email    string `json:"email" `
			Password string `json:"password" `
			Role     int64  `json:"role" `
			Partner  int64  `json:"partner"`
		}{}

		if err := json.NewDecoder(r.Body).Decode(&ud); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Malformed request body"}, http.StatusBadRequest)
			return
		}

		if !role.Is(models.Administrator) {
			if role.Is(models.ChildUser) {
				ud.Partner = user.Partner
			} else {
				ud.Partner = user.Id
			}
		}

		iu, err := models.CreateUser(ud.Username, ud.FullName, ud.Email, ud.Password, ud.Role, ud.Partner)

		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		if os.Getenv("USERSYNC_DISABLE") == "" {
			uid, err := usersync.PushUser(
				iu.Id,
				iu.Username,
				iu.Email, "",
				ud.Password,
				ud.Role,
				models.GetUserBakeryID(ud.Partner),
				false,
			)

			if err != nil {
				_, _ = models.DeleteUser(iu.Id)
				log.Error(err)
				JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
				return
			}

			err = (*iu).SetBakeryUserID(uid)

			if err != nil {
				log.Error(err)
				JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
				return
			}
		}

		if os.Getenv("DONT_NOTIFY_USERS") == "" &&
			(ud.Role == models.Partner || ud.Role == models.Customer) {
			partner := false

			if ud.Role == models.Partner {
				partner = true
			}

			notifier.SendWelcomeEmail(iu.Email, iu.FullName, iu.Username, partner)
		}

		JSONResponse(w, models.Response{Success: true, Message: "User has been created"}, http.StatusOK)
	}
}

// API_User_ByRole returns a list of users of a certain role.
func API_User_ByRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	role := vars["role"]

	switch {
	case r.Method == "GET":
		var rid int64

		if role == "admin" {
			rid = models.Administrator
		} else if role == "partner" {
			rid = models.Partner
		} else if role == "customer" {
			rid = models.Customer
		} else {
			JSONResponse(w, models.Response{Success: false, Message: "Unknown role"}, http.StatusBadRequest)
			return
		}

		users, err := models.GetUsersByRoleID(rid)

		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		JSONResponse(w, users, http.StatusOK)
	}
}

// API_Roles returns a list of roles if requested via GET
func API_Roles(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		uid := ctx.Get(r, "user_id").(int64)
		role, err := models.GetUserRole(uid)

		if err != nil {
			log.Error(err)
		}

		roles, err := models.GetRoles()

		if err != nil {
			log.Error(err)
		}

		JSONResponse(w, roles.AvailableFor(role), http.StatusOK)
	}
}

// API_Users_Id returns details about the requested User. If the User is not
// valid, API_Users_Id returns null.
func API_Users_Id(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	c, err := models.GetUser(id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "User not found"}, http.StatusNotFound)
		return
	}
	switch {
	case r.Method == "GET":
		JSONResponse(w, c, http.StatusOK)

	case r.Method == "DELETE":
		buid, err := models.DeleteUser(id)

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting user"}, http.StatusInternalServerError)
			return
		}

		if os.Getenv("USERSYNC_DISABLE") == "" {
			usersync.DeleteTrainingCampaigns(id)

			if err := usersync.DeleteUser(buid); err != nil {
				log.Error(err)
				JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
				return
			}
		}

		JSONResponse(w, models.Response{Success: true, Message: "User deleted successfully!"}, http.StatusOK)

	case r.Method == "POST":
		err = auth.UpdateSettingsByAdmin(r)
		msg := models.Response{Success: true, Message: "Settings Updated Successfully"}
		if err == auth.ErrInvalidPassword {
			msg.Message = "Invalid Password"
			msg.Success = false
			JSONResponse(w, msg, http.StatusBadRequest)
			return
		}
		if err != nil {
			msg.Message = err.Error()
			msg.Success = false
			JSONResponse(w, msg, http.StatusBadRequest)
			return
		}
		JSONResponse(w, msg, http.StatusOK)

	}
}

// API_Users_Id_ResetPassword handles password reset requests
func API_Users_Id_ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not supported"}, http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	u, err := models.GetUser(id)

	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "User not found"}, http.StatusNotFound)
		return
	}

	buid := models.GetUserBakeryID(u.Id)

	if buid == 0 {
		err := fmt.Errorf("Could not find bakery id of user with id %d", u.Id)
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
		return
	}

	if err := usersync.ResetPassword(buid); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, models.Response{Success: true, Message: "Password-reset email has been sent"}, http.StatusOK)
}

// API_Roles_Id returns details about the requested User. If the User is not
// valid, API_Roles_Id returns null.
func API_Roles_Id(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	c, err := models.GetUserRole(id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "User not found"}, http.StatusNotFound)
		return
	}
	switch {
	case r.Method == "GET":
		JSONResponse(w, c, http.StatusOK)
	}
}

// API_Tags returns all the list of the tags for email templates and landing pages in the site
func API_Tags(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		cs, err := models.GetTags(ctx.Get(r, "user_id").(int64))
		if err != nil {
			log.Error(err)
		}
		JSONResponse(w, cs, http.StatusOK)

	case r.Method == "POST":
		t := models.Tags{}
		// Put the request into a page
		err := json.NewDecoder(r.Body).Decode(&t)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}
		err = models.PostTags(&t)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, t, http.StatusCreated)
	}
}

func API_Tags_Single(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	c, err := models.GetTagById(id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Tag not found"}, http.StatusNotFound)
		return
	}
	switch {
	case r.Method == "GET":
		JSONResponse(w, c, http.StatusOK)
	case r.Method == "PUT":
		t := models.Tags{}
		err = json.NewDecoder(r.Body).Decode(&t)
		if err != nil {
			log.Error(err)
		}
		err = models.PutTags(&t)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, t, http.StatusOK)
	case r.Method == "DELETE":
		err = models.DeleteTags(id)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting campaign"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Campaign deleted successfully!"}, http.StatusOK)
	}
}

// API_Campaigns_Summary returns the summary for the current user's campaigns
func API_Campaigns_Summary(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		cs, err := models.GetCampaignSummaries(ctx.Get(r, "user_id").(int64), r.URL.Query().Get("filter"))

		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		JSONResponse(w, cs, http.StatusOK)
	}
}

// API_Campaigns_Id returns details about the requested campaign. If the campaign is not
// valid, API_Campaigns_Id returns null.
func API_Campaigns_Id(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)

	if !models.IsCampaignAccessibleByUser(id, uid) {
		JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
		return
	}

	c, err := models.GetCampaign(id)

	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}

	switch {
	case r.Method == "GET":
		JSONResponse(w, c, http.StatusOK)
	case r.Method == "DELETE":
		err = models.DeleteCampaign(id)

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting campaign"}, http.StatusInternalServerError)
			return
		}

		JSONResponse(w, models.Response{Success: true, Message: "Campaign deleted successfully!"}, http.StatusOK)
	}
}

// API_Campaigns_Id_Results returns just the results for a given campaign to
// significantly reduce the information returned.
func API_Campaigns_Id_Results(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)

	if !models.IsCampaignAccessibleByUser(id, uid) {
		JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
		return
	}

	if models.IsCampaignLockedForUser(id, uid) {
		log.Errorf("Campaign %d is locked for user %d", id, uid)
		JSONResponse(w, models.Response{Success: false, Message: "Campaign is locked"}, http.StatusForbidden)
		return
	}

	cr, err := models.GetCampaignResults(id)

	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}

	if r.Method == "GET" {
		JSONResponse(w, cr, http.StatusOK)
		return
	}
}

// API_Campaigns_Id_Summary returns just the summary for a given campaign.
func API_Campaign_Id_Summary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	switch {
	case r.Method == "GET":
		cs, err := models.GetCampaignSummary(id, ctx.Get(r, "user_id").(int64))
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
			} else {
				JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			}
			log.Error(err)
			return
		}
		JSONResponse(w, cs, http.StatusOK)
	}
}

// API_Campaigns_Id_Complete effectively "ends" a campaign.
// Future phishing emails clicked will return a simple "404" page.
func API_Campaigns_Id_Complete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)

	switch {
	case r.Method == "GET":
		if !models.IsCampaignAccessibleByUser(id, uid) {
			JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
			return
		}

		err := models.CompleteCampaign(id)

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error completing campaign"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Campaign completed successfully!"}, http.StatusOK)
	}
}

// API_Groups returns a list of groups if requested via GET.
// If requested via POST, API_Groups creates a new group and returns a reference to it.
func API_Groups(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		gs, err := models.GetGroups(ctx.Get(r, "user_id").(int64))
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "No groups found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, gs, http.StatusOK)
	//POST: Create a new group and return it as JSON
	case r.Method == "POST":
		uid := ctx.Get(r, "user_id").(int64)
		u, err := models.GetUser(uid)

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		if !u.CanCreateGroup() {
			JSONResponse(
				w, models.Response{
					Success: false,
					Message: "It's not possible to create more groups for this subscription plan",
				},
				http.StatusConflict)
			return
		}

		g := models.Group{}
		// Put the request into a group
		err = json.NewDecoder(r.Body).Decode(&g)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON structure"}, http.StatusBadRequest)
			return
		}
		_, err = models.GetGroupByName(g.Name, uid)
		if err != gorm.ErrRecordNotFound {
			JSONResponse(w, models.Response{Success: false, Message: "Group name already in use"}, http.StatusConflict)
			return
		}
		g.ModifiedDate = time.Now().UTC()
		g.UserId = uid

		if !u.CanHaveXTargetsInAGroup(len(g.Targets)) {
			JSONResponse(w,
				models.Response{
					Success: false,
					Message: fmt.Sprintf("It's not possible to have %d targets in a group for this subscription plan", len(g.Targets)),
				}, http.StatusConflict)
			return
		}

		if (u.IsPartner() || u.IsChildUser()) && g.ContainsTargetsOutsideOfDomain(u.Domain) {
			JSONResponse(w,
				models.Response{
					Success: false,
					Message: fmt.Sprintf("There is one or more target emails not in %s domain", u.Domain),
				}, http.StatusConflict)
			return
		}

		err = models.PostGroup(&g)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, g, http.StatusCreated)
	}
}

// API_Groups_Summary returns a summary of the groups owned by the current user.
func API_Groups_Summary(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		gs, err := models.GetGroupSummaries(ctx.Get(r, "user_id").(int64), r.URL.Query().Get("filter"))
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, gs, http.StatusOK)
	}
}

// API_Groups_Id returns details about the requested group.
// If the group is not valid, API_Groups_Id returns null.
func API_Groups_Id(w http.ResponseWriter, r *http.Request) {
	uid := ctx.Get(r, "user_id").(int64)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)

	if !models.IsGroupAccessibleByUser(id, uid) {
		JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
		return
	}

	g, err := models.GetGroup(id)

	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Group not found"}, http.StatusNotFound)
		return
	}

	switch {
	case r.Method == "GET":
		JSONResponse(w, g, http.StatusOK)
	case r.Method == "DELETE":
		err = models.DeleteGroup(&g)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting group"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Group deleted successfully!"}, http.StatusOK)
	case r.Method == "PUT":
		// Change this to get from URL and uid (don't bother with id in r.Body)
		g = models.Group{}
		err = json.NewDecoder(r.Body).Decode(&g)

		if g.Id != id {
			JSONResponse(w, models.Response{Success: false, Message: "Error: /:id and group_id mismatch"}, http.StatusInternalServerError)
			return
		}

		g.ModifiedDate = time.Now().UTC()
		u, err := models.GetUser(uid)

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		if !u.CanHaveXTargetsInAGroup(len(g.Targets)) {
			JSONResponse(w,
				models.Response{
					Success: false,
					Message: fmt.Sprintf("It's not possible to have %d targets in a group for this subscription plan", len(g.Targets)),
				}, http.StatusConflict)
			return
		}

		err = models.PutGroup(&g)

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}

		JSONResponse(w, g, http.StatusOK)
	}
}

// API_Groups_Id_Summary returns a summary of the groups owned by the current user.
func API_Groups_Id_Summary(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		vars := mux.Vars(r)
		id, _ := strconv.ParseInt(vars["id"], 0, 64)
		g, err := models.GetGroupSummary(id, ctx.Get(r, "user_id").(int64))
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Group not found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, g, http.StatusOK)
	}
}

// API_Groups_Id_LMS handles creation and removal of LMS users
func API_Groups_Id_LMS(w http.ResponseWriter, r *http.Request) {
	uid := ctx.Get(r, "user_id").(int64)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	g, err := models.GetGroup(id)

	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Group not found"}, http.StatusNotFound)
		return
	}

	if !models.IsGroupAccessibleByUser(id, uid) {
		JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
		return
	}

	u, err := models.GetUser(uid)

	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Could not determine current user"}, http.StatusInternalServerError)
		return
	}

	if !u.IsAdministrator() && !u.IsSubscribed() {
		JSONResponse(w, models.Response{Success: false, Message: "LMS user management requires a valid subscription"}, http.StatusPreconditionFailed)
		return
	}

	var tids []int64

	if err := json.NewDecoder(r.Body).Decode(&tids); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Malformed request body"}, http.StatusBadRequest)
		return
	}

	if !g.HasTargets(tids) {
		JSONResponse(
			w, models.Response{
				Success: false,
				Message: "One or more target ids belong to a different user group",
			},

			http.StatusBadRequest,
		)

		return
	}

	log.Info(tids)

	switch {
	case r.Method == "POST":
		ts, err := models.GetTargetsByIds(tids)

		if err != nil {
			err = fmt.Errorf("Could not retrieve group targets - %s", err.Error())
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		if len(ts) == 0 {
			JSONResponse(w, models.Response{Success: false, Message: "No users selected"}, http.StatusBadRequest)
			return
		}

		groupOwner, err := models.GetUser(g.UserId)

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Could not determine group owner"}, http.StatusInternalServerError)
			return
		}

		partner := groupOwner.Id

		if groupOwner.IsChildUser() {
			partner = groupOwner.Partner
		}

		j := job.New(ts)

		j.Start(func(j *job.Job) {
			targets := j.Params().([]models.Target)

			calcProgress := func(current, total int) int {
				return int(current * 100 / total)
			}

			for i, t := range targets {
				fullname := t.FirstName + " " + t.LastName
				username := util.GenerateUsername(fullname, t.Email)
				u, err := models.CreateUser(username, fullname, t.Email, "qwerty", models.LMSUser, partner)

				if err != nil {
					j.Progress <- calcProgress(i, len(targets))
					j.Errors <- fmt.Errorf("Could not create LMS user - %s", err.Error())
					continue
				}

				if os.Getenv("USERSYNC_DISABLE") == "" {
					buid, err := usersync.PushUser(
						u.Id,
						u.Username,
						u.Email,
						u.FullName,
						"qwerty",
						models.LMSUser,
						models.GetUserBakeryID(partner),
						false,
					)

					if err != nil {
						email := u.Email
						_, _ = models.DeleteUser(u.Id)
						j.Progress <- calcProgress(i, len(targets))
						j.Errors <- fmt.Errorf("Could not push user (%s) to the main server - %s", email, err.Error())
						continue
					}

					err = (*u).SetBakeryUserID(buid)

					if err != nil {
						log.Error(err)
					}
				}

				j.Progress <- calcProgress(i, len(targets))
			}

			j.Progress <- 100
			j.Done <- true
		})

		JSONResponse(w, models.Response{Success: true, Message: "Accepted", Data: j.ID()}, http.StatusOK)

	case r.Method == "DELETE":
		ts, err := models.GetTargetsByIds(tids)

		if err != nil {
			err = fmt.Errorf("Could not retrieve group targets - %s", err.Error())
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		if len(ts) == 0 {
			JSONResponse(w, models.Response{Success: false, Message: "No users selected"}, http.StatusBadRequest)
			return
		}

		j := job.New(ts)

		j.Start(func(j *job.Job) {
			targets := j.Params().([]models.Target)

			calcProgress := func(current, total int) int {
				return int(current * 100 / total)
			}

			for i, t := range targets {
				u, err := models.GetUserByUsername(t.Email)

				if err != nil {
					j.Progress <- calcProgress(i, len(targets))
					j.Errors <- fmt.Errorf("Could not find user with email %s - %s", t.Email, err.Error())
					continue
				}

				buid, err := models.DeleteUser(u.Id)

				if err != nil {
					j.Progress <- calcProgress(i, len(targets))
					j.Errors <- fmt.Errorf("Could not delete user with id %d - %s", u.Id, err.Error())
					continue
				}

				if os.Getenv("USERSYNC_DISABLE") == "" {
					usersync.DeleteTrainingCampaigns(u.Id)

					if err := usersync.DeleteUser(buid); err != nil {
						j.Errors <- err
					}
				}

				j.Progress <- calcProgress(i, len(targets))
			}

			j.Progress <- 100
			j.Done <- true
		})

		JSONResponse(w, models.Response{Success: true, Message: "Accepted", Data: j.ID()}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusBadRequest)
	}
}

// API_Groups_Id_LMS_Jobs_Id provides info on LMS user creation job status
func API_Groups_Id_LMS_Jobs_Id(w http.ResponseWriter, r *http.Request) {
	uid := ctx.Get(r, "user_id").(int64)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	jid := vars["jid"]
	_, err := models.GetGroup(id)

	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Group not found"}, http.StatusNotFound)
		return
	}

	if !models.IsGroupAccessibleByUser(id, uid) {
		JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
		return
	}

	switch {
	case r.Method == "GET":
		result := &lmsUserCreationResult{}
		j := job.Get(jid)

		if j == nil {
			JSONResponse(w, models.Response{Success: false, Message: "Wrong job id"}, http.StatusBadRequest)
			return
		}

		result.Progress = j.GetProgress()
		result.Errors = j.GetErrors()
		JSONResponse(w, models.Response{Success: true, Message: "Status", Data: result}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusBadRequest)
	}
}

// API_Templates handles the functionality for the /api/templates endpoint
func API_Templates(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		ts, err := models.GetTemplates(ctx.Get(r, "user_id").(int64), r.URL.Query().Get("filter"))
		if err != nil {
			log.Error(err)
		}
		JSONResponse(w, ts, http.StatusOK)
	//POST: Create a new template and return it as JSON
	case r.Method == "POST":
		t := models.Template{}
		// Put the request into a template
		err := json.NewDecoder(r.Body).Decode(&t)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON structure"}, http.StatusBadRequest)
			return
		}
		_, err = models.GetTemplateByName(t.Name, ctx.Get(r, "user_id").(int64))
		if err != gorm.ErrRecordNotFound {
			JSONResponse(w, models.Response{Success: false, Message: "Template name already in use"}, http.StatusConflict)
			return
		}
		t.ModifiedDate = time.Now().UTC()
		t.UserId = ctx.Get(r, "user_id").(int64)
		err = models.PostTemplate(&t)

		if err == models.ErrTemplateNameNotSpecified ||
			err == models.ErrTemplateMissingParameter ||
			err == models.ErrTemplateFromAddressNotSpecified ||
			err == models.ErrTemplateCategoryNotSpecified {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error inserting template into database"}, http.StatusInternalServerError)
			log.Error(err)
			return
		}
		JSONResponse(w, t, http.StatusCreated)
	}
}

// API_Templates_Id handles the functions for the /api/templates/:id endpoint
func API_Templates_Id(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)

	if !models.IsTemplateAccessibleByUser(id, uid) {
		JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
		return
	}

	t, err := models.GetTemplate(id)

	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Template not found"}, http.StatusNotFound)
		return
	}

	switch {
	case r.Method == "GET":
		JSONResponse(w, t, http.StatusOK)

	case r.Method == "DELETE":
		if !models.IsTemplateWritableByUser(id, uid) {
			JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
			return
		}

		err = models.DeleteTemplate(id)

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting template"}, http.StatusInternalServerError)
			return
		}

		JSONResponse(w, models.Response{Success: true, Message: "Template deleted successfully!"}, http.StatusOK)

	case r.Method == "PUT":
		t = models.Template{}
		err = json.NewDecoder(r.Body).Decode(&t)

		if err != nil {
			log.Error(err)
		}

		if t.Id != id {
			JSONResponse(w, models.Response{Success: false, Message: "Error: /:id and template_id mismatch"}, http.StatusBadRequest)
			return
		}

		if !t.IsWritableByUser(uid) {
			JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
			return
		}

		t.ModifiedDate = time.Now().UTC()
		err = models.PutTemplate(&t)

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}

		JSONResponse(w, t, http.StatusOK)
	}
}

// API_Templates_Id_Preview handles the functions for the /api/templates/:id/preview endpoint
func API_Templates_Id_Preview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)

	if !models.IsTemplateAccessibleByUser(id, uid) {
		JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
		return
	}

	t, err := models.GetTemplate(id)

	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Template not found"}, http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not supported"}, http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)

	if t.HTML != "" {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, t.HTML)
	} else {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, t.Text)
	}
}

// API_Pages_Id_Preview handles the functions for the /api/pages/:id/preview endpoint
func API_Pages_Id_Preview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)

	if !models.IsPageAccessibleByUser(id, uid) {
		JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
		return
	}

	t, err := models.GetPage(id)

	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Page not found"}, http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not supported"}, http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)

	if t.HTML != "" {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, t.HTML)
	} else {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "Page not found")
	}
}

// API_Pages handles requests for the /api/pages/ endpoint
func API_Pages(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		ps, err := models.GetPages(ctx.Get(r, "user_id").(int64), r.URL.Query().Get("filter"))
		if err != nil {
			log.Error(err)
		}
		JSONResponse(w, ps, http.StatusOK)
	//POST: Create a new page and return it as JSON
	case r.Method == "POST":
		p := models.Page{}
		// Put the request into a page
		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}
		// Check to make sure the name is unique
		_, err = models.GetPageByName(p.Name, ctx.Get(r, "user_id").(int64))
		if err != gorm.ErrRecordNotFound {
			JSONResponse(w, models.Response{Success: false, Message: "Page name already in use"}, http.StatusConflict)
			log.Error(err)
			return
		}
		p.ModifiedDate = time.Now().UTC()
		p.UserId = ctx.Get(r, "user_id").(int64)
		err = models.PostPage(&p)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, p, http.StatusCreated)
	}
}

// API_Pages_Id contains functions to handle the GET'ing, DELETE'ing, and PUT'ing
// of a Page object
func API_Pages_Id(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)

	if !models.IsPageAccessibleByUser(id, uid) {
		JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
		return
	}

	p, err := models.GetPage(id)

	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Page not found"}, http.StatusNotFound)
		return
	}

	switch {
	case r.Method == "GET":
		JSONResponse(w, p, http.StatusOK)
	case r.Method == "DELETE":
		if !models.IsPageWritableByUser(id, uid) {
			JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
			return
		}

		err = models.DeletePage(id)

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting page"}, http.StatusInternalServerError)
			return
		}

		JSONResponse(w, models.Response{Success: true, Message: "Page Deleted Successfully"}, http.StatusOK)
	case r.Method == "PUT":
		p = models.Page{}
		err = json.NewDecoder(r.Body).Decode(&p)

		if err != nil {
			log.Error(err)
		}

		if p.Id != id {
			JSONResponse(w, models.Response{Success: false, Message: "/:id and /:page_id mismatch"}, http.StatusBadRequest)
			return
		}

		if !p.IsWritableByUser(uid) {
			JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
			return
		}

		p.ModifiedDate = time.Now().UTC()
		err = models.PutPage(&p)

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error updating page: " + err.Error()}, http.StatusInternalServerError)
			return
		}

		JSONResponse(w, p, http.StatusOK)
	}
}

// API_SMTP handles requests for the /api/smtp/ endpoint
func API_SMTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		ss, err := models.GetSMTPs(ctx.Get(r, "user_id").(int64))
		if err != nil {
			log.Error(err)
		}
		JSONResponse(w, ss, http.StatusOK)
	//POST: Create a new SMTP and return it as JSON
	case r.Method == "POST":
		s := models.SMTP{}
		// Put the request into a page
		err := json.NewDecoder(r.Body).Decode(&s)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}
		// Check to make sure the name is unique
		_, err = models.GetSMTPByName(s.Name)
		if err != gorm.ErrRecordNotFound {
			JSONResponse(w, models.Response{Success: false, Message: "SMTP name already in use"}, http.StatusConflict)
			log.Error(err)
			return
		}
		s.ModifiedDate = time.Now().UTC()
		s.UserId = ctx.Get(r, "user_id").(int64)
		err = models.PostSMTP(&s)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, s, http.StatusCreated)
	}
}

// API_SMTP handles requests for the /api/smtp/domains endpoint
func API_SMTP_domains(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		ss, err := models.GetAllSMTPs()
		if err != nil {
			log.Error(err)
		}
		JSONResponse(w, prepare(ss, r), http.StatusOK)
	}
}

// API_SMTP_Id contains functions to handle the GET'ing, DELETE'ing, and PUT'ing
// of a SMTP object
func API_SMTP_Id(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	s, err := models.GetSMTP(id, ctx.Get(r, "user_id").(int64))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "SMTP not found"}, http.StatusNotFound)
		return
	}
	switch {
	case r.Method == "GET":
		JSONResponse(w, s, http.StatusOK)
	case r.Method == "DELETE":
		err = models.DeleteSMTP(id, ctx.Get(r, "user_id").(int64))
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting SMTP"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "SMTP Deleted Successfully"}, http.StatusOK)
	case r.Method == "PUT":
		s = models.SMTP{}
		err = json.NewDecoder(r.Body).Decode(&s)
		if err != nil {
			log.Error(err)
		}
		if s.Id != id {
			JSONResponse(w, models.Response{Success: false, Message: "/:id and /:smtp_id mismatch"}, http.StatusBadRequest)
			return
		}
		err = s.Validate()
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		s.ModifiedDate = time.Now().UTC()
		s.UserId = ctx.Get(r, "user_id").(int64)
		err = models.PutSMTP(&s)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error updating page"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, s, http.StatusOK)
	}
}

// API_Import_Group imports a CSV of group members
func API_Import_Group(w http.ResponseWriter, r *http.Request) {
	ts, err := util.ParseCSV(r)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Error parsing CSV"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, ts, http.StatusOK)
	return
}

// API_Import_Email allows for the importing of email.
// Returns a Message object
func API_Import_Email(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusBadRequest)
		return
	}
	ir := struct {
		Content      string `json:"content"`
		ConvertLinks bool   `json:"convert_links"`
	}{}
	err := json.NewDecoder(r.Body).Decode(&ir)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Error decoding JSON Request"}, http.StatusBadRequest)
		return
	}
	e, err := email.NewEmailFromReader(strings.NewReader(ir.Content))
	if err != nil {
		log.Error(err)
	}
	// If the user wants to convert links to point to
	// the landing page, let's make it happen by changing up
	// e.HTML
	if ir.ConvertLinks {
		d, err := goquery.NewDocumentFromReader(bytes.NewReader(e.HTML))
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		d.Find("a").Each(func(i int, a *goquery.Selection) {
			a.SetAttr("href", "{{.URL}}")
		})
		h, err := d.Html()
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		e.HTML = []byte(h)
	}
	er := emailResponse{
		Subject: e.Subject,
		Text:    string(e.Text),
		HTML:    string(e.HTML),
	}
	JSONResponse(w, er, http.StatusOK)
	return
}

// API_Import_Site allows for the importing of HTML from a website
// Without "include_resources" set, it will merely place a "base" tag
// so that all resources can be loaded relative to the given URL.
func API_Import_Site(w http.ResponseWriter, r *http.Request) {
	cr := cloneRequest{}
	if r.Method != "POST" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusBadRequest)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&cr)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Error decoding JSON Request"}, http.StatusBadRequest)
		return
	}
	if err = cr.validate(); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(cr.URL)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	// Insert the base href tag to better handle relative resources
	d, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	// Assuming we don't want to include resources, we'll need a base href
	if d.Find("head base").Length() == 0 {
		d.Find("head").PrependHtml(fmt.Sprintf("<base href=\"%s\">", cr.URL))
	}
	forms := d.Find("form")
	forms.Each(func(i int, f *goquery.Selection) {
		// We'll want to store where we got the form from
		// (the current URL)
		url := f.AttrOr("action", cr.URL)
		if !strings.HasPrefix(url, "http") {
			url = fmt.Sprintf("%s%s", cr.URL, url)
		}
		f.PrependHtml(fmt.Sprintf("<input type=\"hidden\" name=\"__original_url\" value=\"%s\"/>", url))
	})
	h, err := d.Html()
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	cs := cloneResponse{HTML: h}
	JSONResponse(w, cs, http.StatusOK)
	return
}

// API_Send_Test_Email sends a test email using the template name
// and Target given.
func API_Send_Test_Email(w http.ResponseWriter, r *http.Request) {
	parsedURL, err := url.Parse("https://" + r.Host)

	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Could not determine hostname"}, http.StatusInternalServerError)
		return
	}

	proto := "http://"

	if config.Conf.PhishConf.UseTLS || os.Getenv("VIA_PROXY") != "" {
		proto = "https://"
	}

	url := proto + parsedURL.Hostname()

	if os.Getenv("PHISHING_DOMAIN") != "" {
		url = proto + os.Getenv("PHISHING_DOMAIN")
	}

	s := &models.EmailRequest{
		ErrorChan: make(chan error),
		UserId:    ctx.Get(r, "user_id").(int64),
		URL:       url,
	}
	if r.Method != "POST" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusBadRequest)
		return
	}
	err = json.NewDecoder(r.Body).Decode(s)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Error decoding JSON Request"}, http.StatusBadRequest)
		return
	}

	storeRequest := false

	// If a Template is not specified use a default
	if s.Template.Name == "" {
		//default message body
		text := "It works!\n\nThis is an email letting you know that your security awareness training\nconfiguration was successful.\n" +
			"Here are the details:\n\nWho you sent from: {{.From}}\n\nWho you sent to: \n" +
			"{{if .FirstName}} First Name: {{.FirstName}}\n{{end}}" +
			"{{if .LastName}} Last Name: {{.LastName}}\n{{end}}" +
			"{{if .Position}} Position: {{.Position}}\n{{end}}" +
			"\nNow go send some phish!"
		t := models.Template{
			Subject: "Default Email from Security Awraeness Training",
			Text:    text,
		}
		s.Template = t
	} else {
		// Get the Template requested by name
		s.Template, err = models.GetTemplateByName(s.Template.Name, s.UserId)
		if err == gorm.ErrRecordNotFound {
			log.WithFields(logrus.Fields{
				"template": s.Template.Name,
			}).Error("Template does not exist")
			JSONResponse(w, models.Response{Success: false, Message: models.ErrTemplateNotFound.Error()}, http.StatusBadRequest)
			return
		} else if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		s.TemplateId = s.Template.Id
		// We'll only save the test request to the database if there is a
		// user-specified template to use.
		storeRequest = true
	}

	if s.Page.Name != "" {
		s.Page, err = models.GetPageByName(s.Page.Name, s.UserId)
		if err == gorm.ErrRecordNotFound {
			log.WithFields(logrus.Fields{
				"page": s.Page.Name,
			}).Error("Page does not exist")
			JSONResponse(w, models.Response{Success: false, Message: models.ErrPageNotFound.Error()}, http.StatusBadRequest)
			return
		} else if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		s.PageId = s.Page.Id
	}

	// If a complete sending profile is provided use it
	if err := s.SMTP.Validate(); err != nil {
		// Otherwise get the SMTP requested by name
		smtp, lookupErr := models.GetSMTPByName(s.SMTP.Name)
		// If the Sending Profile doesn't exist, let's err on the side
		// of caution and assume that the validation failure was more important.
		if lookupErr != nil {
			log.Error(err)
			msg := err.Error()

			if err == models.ErrFromAddressNotSpecified {
				msg = "Please select a sending domain before sending a test email"
			}

			JSONResponse(w, models.Response{Success: false, Message: msg}, http.StatusBadRequest)
			return
		}
		s.SMTP = smtp
	}

	if s.FromAddress != "" {
		if _, err = mail.ParseAddress(s.FromAddress); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Bad From: address"}, http.StatusBadRequest)
			return
		}
	} else if s.Template.FromAddress != "" {
		if _, err = mail.ParseAddress(s.Template.FromAddress); err != nil {
			s.FromAddress = s.SMTP.FromAddress
		} else {
			s.FromAddress = s.Template.FromAddress
		}
	} else {
		s.FromAddress = s.SMTP.FromAddress
	}

	// Validate the given request
	if err = s.Validate(); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}

	// Store the request if this wasn't the default template
	if storeRequest {
		err = models.PostEmailRequest(s)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
	}
	// Send the test email
	err = Worker.SendTestEmail(s)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Email Sent"}, http.StatusOK)
	return
}

// API_Plans handles requests for the /api/plans/ endpoint
func API_Plans(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		plans, err := models.GetPlans()
		if err != nil {
			log.Error(err)
		}
		JSONResponse(w, plans, http.StatusOK)
	//POST: Create a new page and return it as JSON
	case r.Method == "POST":
		plan := models.Plan{}
		// Put the request into a page
		err := json.NewDecoder(r.Body).Decode(&plan)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}

		_, err = models.GetPlanByName(plan.Name)

		if err != gorm.ErrRecordNotFound {
			JSONResponse(w, models.Response{Success: false, Message: "Plan name already in use"}, http.StatusConflict)
			log.Error(err)
			return
		}

		err = models.PostPlan(&plan)

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		JSONResponse(w, plan, http.StatusCreated)
	}
}

// API_Subscriptions handles requests for the /api/subscriptions/ endpoint
func API_Subscriptions(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		subscriptions, err := models.GetSubscriptions()
		if err != nil {
			log.Error(err)
		}
		JSONResponse(w, subscriptions, http.StatusOK)
	case r.Method == "POST":
		subscription := models.Subscription{}

		err := json.NewDecoder(r.Body).Decode(&subscription)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}

		err = models.PostSubscription(&subscription)

		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}

		JSONResponse(w, subscription, http.StatusCreated)
	}
}

// API_Subscription handles subscription cancellation requests
func API_Subscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	u := ctx.Get(r, "user").(models.User)
	s := u.GetSubscription()

	if s == nil || !s.IsActive() {
		JSONResponse(w, models.Response{Success: false, Message: "Not subscribed"}, http.StatusBadRequest)
		return
	}

	if err := models.DeleteUserSubscriptions(u.Id); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}

	JSONResponse(w, models.Response{Success: true, Message: "Subscription cancelled"}, http.StatusOK)
}

// API_Auth_LAK handles generation of limited access keys
func API_Auth_LAK(w http.ResponseWriter, r *http.Request) {
	uid := ctx.Get(r, "user_id").(int64)
	ip := r.RemoteAddr

	if ipAddr, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = ipAddr
	}

	r.ParseForm()
	route := r.Form.Get("route")

	if route == "" {
		JSONResponse(w, models.Response{Success: false, Message: "Missing route param"}, http.StatusBadRequest)
		return
	}

	key := auth.GenerateLimitedAccessKey(uid, ip, route)

	if key == "" {
		JSONResponse(w, models.Response{Success: false, Message: "Could not generate limited access key"}, http.StatusInternalServerError)
		return
	}

	JSONResponse(w, models.Response{Success: true, Message: "Limited access key for route " + route, Data: key}, http.StatusOK)
}

// API_User handles account deletion requests
func API_User(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	u := ctx.Get(r, "user").(models.User)
	u.ToBeDeleted = true
	err := models.PutUser(&u)

	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	if os.Getenv("DONT_NOTIFY_USERS") == "" {
		if role, err := models.GetUserRole(u.Id); err == nil {
			var partnerAddr, partnerName string

			if u.Partner != 0 {
				if partner, err := models.GetUser(u.Partner); err == nil {
					partnerAddr = partner.Email
					partnerName = partner.FullName
				}
			}

			notifier.SendDeletionRequestEmail(partnerAddr, partnerName, u.Username, u.FullName, role.DisplayName())
		}
	}

	JSONResponse(w, models.Response{Success: true, Message: "Account will be deleted"}, http.StatusOK)
}

// API_UserSync handles updates and deletions of users
func API_UserSync(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	buid, _ := strconv.ParseInt(vars["buid"], 0, 64)

	switch {
	case r.Method == "DELETE":
		if dump, err := httputil.DumpRequest(r, true); err == nil {
			log.WithFields(map[string]interface{}{"tag": "usersync.DELETE <-"}).Infof("%q", dump)
		}

		u, err := models.GetUserByBakeryID(buid)

		if err != nil {
			log.Error(err)
			LoggableJSONResponse(w,
				models.Response{Success: false, Message: "User not found"},
				http.StatusNotFound, "usersync.DELETE ->")

			return
		}

		_, err = models.DeleteUser(u.Id)

		if err != nil {
			LoggableJSONResponse(w,
				models.Response{Success: false, Message: "Error deleting user"},
				http.StatusInternalServerError, "usersync.DELETE ->")

			return
		}

		LoggableJSONResponse(w,
			models.Response{Success: true, Message: "User deleted"},
			http.StatusOK, "usersync.DELETE ->")

	case r.Method == "PUT":
		if dump, err := httputil.DumpRequest(r, true); err == nil {
			log.WithFields(map[string]interface{}{"tag": "usersync.PUT <-"}).Infof("%q", dump)
		}

		u, err := models.GetUserByBakeryID(buid)

		if err != nil {
			log.Error(err)
			LoggableJSONResponse(w,
				models.Response{Success: false, Message: "User not found"},
				http.StatusNotFound, "usersync.PUT ->")

			return
		}

		props := struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			FullName string `json:"fullname"`
			Domain   string `json:"domain"`
			Role     int64  `json:"role"`
			Partner  int64  `json:"partner"` // bakery user id
		}{}

		if err := json.NewDecoder(r.Body).Decode(&props); err != nil {
			log.Error(err)

			LoggableJSONResponse(w,
				models.Response{Success: false, Message: fmt.Sprintf("Invalid request - %s", err.Error())},
				http.StatusBadRequest, "usersync.PUT ->")

			return
		}

		if props.Partner != 0 {
			partner, err := models.GetUserByBakeryID(props.Partner)

			if err != nil {
				err = fmt.Errorf("Could not find partner user with bakery user id %d - %s", props.Partner, err.Error())
				log.Error(err)

				LoggableJSONResponse(w,
					models.Response{Success: false, Message: err.Error()},
					http.StatusInternalServerError, "usersync.PUT ->")

				return
			}

			props.Partner = partner.Id
		}

		if err := models.UpdateUser(
			&models.User{
				Id:       u.Id,
				Username: props.Username,
				Email:    props.Email,
				FullName: props.FullName,
				Domain:   props.Domain,
				Partner:  props.Partner,
			}); err != nil {
			log.Error(err)

			LoggableJSONResponse(w,
				models.Response{
					Success: false,
					Message: fmt.Sprintf("Could not update user with id %d - %s", u.Id, err.Error())},
				http.StatusBadRequest, "usersync.PUT ->")

			return
		}

		if props.Role > 0 && props.Role < 6 {
			if err = models.SetUserRole(u.Id, props.Role); err != nil {
				LoggableJSONResponse(w,
					models.Response{
						Success: false,
						Message: fmt.Sprintf("Could not update role of user with id %d - %s", u.Id, err.Error())},
					http.StatusInternalServerError, "usersync.PUT ->")

				return
			}
		}

		LoggableJSONResponse(w,
			models.Response{Success: true, Message: "User updated"},
			http.StatusOK, "usersync.PUT ->")
	}
}

// API_PhishAlarm handles sending of phish alarm emails
func API_PhishAlarm(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	if dump, err := httputil.DumpRequest(r, true); err == nil {
		log.WithFields(map[string]interface{}{"tag": "phishalarm.POST <-"}).Infof("%q", dump)
	}

	props := struct {
		ID      string `json:"id"`
		From    string `json:"from"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
		Domain  string `json:"domain"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&props); err != nil {
		LoggableJSONResponse(w,
			models.Response{Success: false, Message: fmt.Sprintf("Invalid request - %s", err.Error())},
			http.StatusBadRequest, "phishalarm.POST ->")

		return
	}

	if props.Domain == "" {
		LoggableJSONResponse(w,
			models.Response{Success: false, Message: "Missing domain"},
			http.StatusBadRequest, "phishalarm.POST ->")

		return
	}

	u, err := models.GetUserByDomain(props.Domain)

	if err != nil {
		LoggableJSONResponse(w,
			models.Response{Success: false, Message: fmt.Sprintf("Could not find user with domain - %s", props.Domain)},
			http.StatusBadRequest, "phishalarm.POST ->")

		return
	}

	to := u.Email

	if u.AdminEmail != "" {
		to = u.AdminEmail
	}

	notifier.SendPhishAlarmEmail(to, props.From, props.ID, props.Subject, props.Body)

	LoggableJSONResponse(w,
		models.Response{Success: true, Message: "Phish alarm e-mail sent to " + to},
		http.StatusOK, "phishalarm.POST ->")
}

// JSONResponse attempts to set the status code, c, and marshal the given interface, d, into a response that
// is written to the given ResponseWriter.
func JSONResponse(w http.ResponseWriter, d interface{}, c int) {
	dj, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
		log.Error(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	fmt.Fprintf(w, "%s", dj)
}

// LoggableJSONResponse does the same as JSONResponse and logs the response with the given tag
func LoggableJSONResponse(w http.ResponseWriter, d interface{}, c int, tag string) {
	dj, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
		log.Error(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	fmt.Fprintf(w, "%s", dj)
	log.WithFields(map[string]interface{}{"tag": tag}).Infof("%q", dj)
}

// prepare filters out sensitive fields based on user role
func prepare(data interface{}, r *http.Request) interface{} {
	uid := ctx.Get(r, "user_id").(int64)
	role, err := models.GetUserRole(uid)

	if err != nil {
		return nil
	}

	if smtps, ok := data.([]models.SMTP); ok {
		if role.Is(models.Administrator) {
			return smtps
		}

		resp := []map[string]interface{}{}

		for _, smtp := range smtps {
			resp = append(resp, map[string]interface{}{
				"id":   smtp.Id,
				"name": smtp.Name,
				"host": smtp.Host,
			})
		}

		return resp
	}

	return data
}

type cloneRequest struct {
	URL              string `json:"url"`
	IncludeResources bool   `json:"include_resources"`
}

func (cr *cloneRequest) validate() error {
	if cr.URL == "" {
		return errors.New("No URL Specified")
	}
	return nil
}

type cloneResponse struct {
	HTML string `json:"html"`
}

type emailResponse struct {
	Text    string `json:"text"`
	HTML    string `json:"html"`
	Subject string `json:"subject"`
}

type lmsUserCreationResult struct {
	Progress int      `json:"progress"`
	Errors   []string `json:"errors"`
}
