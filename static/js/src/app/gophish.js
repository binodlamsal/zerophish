function errorFlash(e) {
  $("#flashes").empty(),
    $("#flashes").append(
      '<div style="text-align:center" class="alert alert-danger">        <i class="fa fa-exclamation-circle"></i> ' +
        e +
        "</div>"
    );
}

function successFlash(e) {
  $("#flashes").empty(),
    $("#flashes").append(
      '<div style="text-align:center" class="alert alert-success">        <i class="fa fa-check-circle"></i> ' +
        e +
        "</div>"
    );
}

function modalError(e) {
  $("#modal\\.flashes")
    .empty()
    .append(
      '<div style="text-align:center" class="alert alert-danger">        <i class="fa fa-exclamation-circle"></i> ' +
        e +
        "</div>"
    );
}

function scrollToError(e) {
  $("#modal").animate(
    {
      scrollTop: $("#modal\\.flashes").offset().top
    },
    200
  );
}

function query(e, t, n, r) {
  const sep = e.includes("?") ? "&" : "?";

  return $.ajax({
    url: "/api" + e,
    headers: { Authorization: "Bearer " + user.api_key },
    async: r,
    method: t,
    type: t,
    data: t !== "GET" ? JSON.stringify(n) : null,
    dataType: "json",
    contentType: "application/json"
  });
}

function escapeHtml(e) {
  return $("<div/>")
    .text(e)
    .html();
}

function unescapeHtml(e) {
  return $("<div/>")
    .html(e)
    .text();
}
var capitalize = function(e) {
    return e.charAt(0).toUpperCase() + e.slice(1);
  },
  api = {
    campaigns: {
      get: function() {
        return query("/campaigns/", "GET", {}, !1);
      },
      post: function(e) {
        return query("/campaigns/", "POST", e, !1);
      },
      summary: function(filter) {
        return query(
          "/campaigns/summary" + (filter ? "?filter=" + filter : ""),
          "GET",
          {},
          !1
        );
      }
    },
    users: {
      get: function() {
        return query("/people", "GET", {}, !1);
      },
      post: function(e) {
        return query("/people", "POST", e, true);
      },
      admins: function() {
        return query("/people/by_role/admin", "GET", {}, !0);
      },
      partners: function() {
        return query("/people/by_role/partner", "GET", {}, !0);
      },
      customers: function() {
        return query("/people/by_role/customer", "GET", {}, !0);
      }
    },
    plans: {
      get: function() {
        return query("/plans", "GET", {}, !1);
      }
    },
    roles: {
      get: function() {
        return query("/roles", "GET", {}, !1);
      }
    },
    rolesByUserId: {
      get: function(e) {
        return query("/roles/" + e, "GET", e, !1);
      }
    },
    userId: {
      get: function(e) {
        return query("/people/" + e, "GET", {}, !1);
      },
      post: function(e) {
        return query("/people/" + e.id, "POST", e, true);
      },
      resetPassword: function(e) {
        return query("/people/" + e + "/reset_password", "POST", {}, true);
      },
      delete: function(e) {
        return query("/people/" + e, "DELETE", {}, !1);
      }
    },
    campaignId: {
      get: function(e) {
        return query("/campaigns/" + e, "GET", {}, !0);
      },
      delete: function(e) {
        return query("/campaigns/" + e, "DELETE", {}, !1);
      },
      results: function(e) {
        return query("/campaigns/" + e + "/results", "GET", {}, !0);
      },
      complete: function(e) {
        return query("/campaigns/" + e + "/complete", "GET", {}, !0);
      },
      summary: function(e) {
        return query("/campaigns/" + e + "/summary", "GET", {}, !0);
      }
    },
    groups: {
      get: function() {
        return query("/groups/", "GET", {}, !1);
      },
      post: function(e) {
        return query("/groups/", "POST", e, !1);
      },
      summary: function(filter) {
        return query(
          "/groups/summary" + (filter ? "?filter=" + filter : ""),
          "GET",
          {},
          !0
        );
      }
    },
    groupId: {
      get: function(e) {
        return query("/groups/" + e, "GET", {}, !1);
      },
      put: function(e) {
        return query("/groups/" + e.id, "PUT", e, !1);
      },
      delete: function(e) {
        return query("/groups/" + e, "DELETE", {}, !1);
      },
      lms: {
        post: function(gid, ids) {
          return query("/groups/" + gid + "/lms_users", "POST", ids, true);
        },
        delete: function(gid, ids) {
          return query("/groups/" + gid + "/lms_users", "DELETE", ids, true);
        },
        jobs: {
          get: function(gid, jid) {
            return query(
              "/groups/" + gid + "/lms_users/jobs/" + jid,
              "GET",
              {},
              true
            );
          }
        }
      }
    },
    templates: {
      get: function(filter) {
        return query(
          "/templates/" + (filter ? "?filter=" + filter : ""),
          "GET",
          {},
          !1
        );
      },
      post: function(e) {
        return query("/templates/", "POST", e, !1);
      }
    },
    phishtags: {
      get: function() {
        return query("/phishtags/", "GET", {}, !1);
      },
      post: function(e) {
        return query("/phishtags/", "POST", e, !1);
      },
      single: function(e) {
        return query("/phishtagssingle/" + e, "GET", {}, !1);
      },
      put: function(e) {
        return query("/phishtagssingle/" + e.id, "PUT", e, !1);
      },
      delete: function(e) {
        return query("/phishtagssingle/" + e, "DELETE", e, !1);
      }
    },
    templateId: {
      get: function(e) {
        return query("/templates/" + e, "GET", {}, !0);
      },
      put: function(e) {
        return query("/templates/" + e.id, "PUT", e, !1);
      },
      delete: function(e) {
        return query("/templates/" + e, "DELETE", {}, !1);
      }
    },
    pages: {
      get: function(filter) {
        return query(
          "/pages/" + (filter ? "?filter=" + filter : ""),
          "GET",
          {},
          !1
        );
      },
      post: function(e) {
        return query("/pages/", "POST", e, !1);
      }
    },
    pageId: {
      get: function(e) {
        return query("/pages/" + e, "GET", {}, !1);
      },
      put: function(e) {
        return query("/pages/" + e.id, "PUT", e, !1);
      },
      delete: function(e) {
        return query("/pages/" + e, "DELETE", {}, !1);
      }
    },
    SMTP: {
      get: function() {
        return query("/smtp/", "GET", {}, !1);
      },
      domains: function() {
        return query("/sendingdomains", "GET", {}, !0);
      },
      post: function(e) {
        return query("/smtp/", "POST", e, !1);
      }
    },
    SMTPId: {
      get: function(e) {
        return query("/smtp/" + e, "GET", {}, !1);
      },
      put: function(e) {
        return query("/smtp/" + e.id, "PUT", e, !1);
      },
      delete: function(e) {
        return query("/smtp/" + e, "DELETE", {}, !1);
      }
    },
    import_email: function(e) {
      return query("/import/email", "POST", e, !1);
    },
    clone_site: function(e) {
      return query("/import/site", "POST", e, !1);
    },
    send_test_email: function(e) {
      return query("/util/send_test_email", "POST", e, !0);
    },
    reset: function() {
      return query("/reset", "POST", {}, !0);
    },
    subscription: {
      cancel: function() {
        return query("/subscription", "DELETE", {}, !0);
      }
    },
    user: {
      put: function(e) {
        return query("/user", "PUT", e, !0);
      },
      delete: function() {
        return query("/user", "DELETE", {}, !0);
      }
    },
    auth: {
      lak: {
        get: function(e) {
          return query("/auth/lak?route=" + e, "GET", {}, !1);
        }
      }
    }
  };
$(document).ready(function() {
  $.fn.dataTable.moment("MMMM Do YYYY, h:mm:ss a");
  $('[data-toggle="tooltip"]').tooltip();
});
