var people = [];
var peopleTable;

function save(e) {
  var t = {};

  t.username = $("#username").val();
  t.full_name = $("#full_name").val();
  t.email = $("#email").val();
  t.domain = $("#domain").val();
  t.time_zone = $("#time_zone").val();
  t.current_password = $("#curpassword").val();
  t.new_password = $("#password").val();
  t.confirm_new_password = $("#confirm_password").val();
  t.api_key = $("#hidden_api_key").val();
  t.id = e;
  t.role = parseInt($("#roles").val());
  t.partner =
    parseInt($("#partner").val()) || parseInt($("#hidden_partner").val());
  t.plan_id =
    parseInt($("#plan_id").val()) || parseInt($("#hidden_plan_id").val());

  if ($("#expiration_date").length) {
    if ($("#expiration_date").val() != "") {
      t.expiration_date = $("#expiration_date").val() + "T23:59:59.000Z";
    }
  }

  api.userId
    .post(t)
    .success(function(e) {
      successFlash("User updated successfully!"), dismiss();
      load();
    })
    .error(function(e) {
      modalError(e.responseJSON.message);
    });
}

function create() {
  var t = {};
  t.username = $("#username").val();
  t.full_name = $("#full_name").val();
  t.email = $("#email").val();
  t.password = $("#password").val();
  t.role = parseInt($("#roles").val()) || null;
  t.partner = parseInt($("#partner").val()) || null;

  if (!isValidPassword(t.password)) {
    modalError(
      "Password must be at least 8 chars long with at least 1 letter, 1 number and 1 special character"
    );

    return;
  }

  api.users
    .post(t)
    .success(function(e) {
      successFlash("User created successfully!"), dismiss();
      load();
    })
    .error(function(e) {
      modalError(e.responseJSON.message);
    });
}

function edit(index) {
  $("#modal .modal-title").html("ADD USER");
  if (index != -1) {
    $("#modal .modal-title").html("EDIT USER");
    var user = people[index];
    var exp_date =
      user.subscription != undefined
        ? moment(user.subscription.expiration_date)
            .utc()
            .format("YYYY-MM-DD")
        : null;

    if (exp_date == null || user.role == "Administrator") {
      $("#expiration_date").attr("disabled", "disabled");

      if (user.role == "Administrator") {
        $("#plan_id").attr("disabled", "disabled");
      } else {
        $("#plan_id").removeAttr("disabled");
      }
    } else {
      $("#plan_id").removeAttr("disabled");
      $("#expiration_date").removeAttr("disabled");
    }

    if ($("#expiration_date").length) {
      setTimeout(function() {
        $("#expiration_date").val(exp_date);
      }, 500);
    }

    if (user.role == "LMS User") {
      $(".subscription").hide();
    } else {
      $(".subscription").show();
    }

    $("#modalSubmit")
      .unbind("click")
      .click(function() {
        save(user.id);
      });

    api.userId.get(user.id).success(function(e) {
      $("input[type=text], textarea").val("");

      $("#username").val(e.username);
      $("#full_name").val(e.full_name);
      $("#email").val(e.email);
      $("#domain").val(e.domain);

      $("#time_zone")
        .val(e.time_zone)
        .trigger("change");

      $("#hidden_hash").val(e.hash);
      $("#hidden_uid").val(e.id);
      $("#hidden_api_key").val(e.api_key);

      var partner = e.partner;

      api.roles.get().success(function(r) {
        $("#roles")
          .find("option")
          .each(function(i) {
            if ($(this).val() !== "") {
              $(this).remove();
            }
          });

        $.each(r, function(e, rr) {
          $("#roles").append(
            '<option value="' + rr.rid + '" >' + rr.name + "</option>"
          );
        });
      });

      //populate the roles
      api.rolesByUserId.get(user.id).success(function(r) {
        $("#roles option").prop("selected", false);
        $("#roles option[value=" + r.rid + "]").prop("selected", true);

        // hide partner field for non-customers and non-child-users
        if (r.rid !== 3 && r.rid !== 4) {
          $("#partner-container").css("display", "none");
        } else {
          $("#partner-container").css("display", "");
        }
      });

      //populate the partners
      api.users.partners().success(function(p) {
        $("#partner")
          .find("option")
          .each(function(i) {
            if ($(this).val() !== "") {
              $(this).remove();
            }
          });

        if (!$("#partner").length) {
          $("#hidden_partner").val(partner);
        } else {
          $("#hidden_partner").val("");
        }

        $.each(p, function(e, pp) {
          var selected = "";
          if (partner == pp.id) {
            selected = 'selected = "selected"';
          } else {
            selected = "";
          }

          $("#partner").append(
            '<option value="' +
              pp.id +
              '"  ' +
              selected +
              ">" +
              pp.username +
              "</option>"
          );
        });
      });

      if (canManageSubscriptions) {
        //populate the plans
        $("#hidden_plan_id").val("");

        api.plans.get().success(function(plans) {
          $("#plan_id")
            .find("option")
            .each(function(i) {
              if ($(this).val() !== "") {
                $(this).remove();
              }
            });

          $.each(plans, function(e, plan) {
            var selected = "";
            if (
              user.subscription != undefined &&
              user.subscription.plan_id == plan.id
            ) {
              selected = 'selected = "selected"';
            } else {
              selected = "";
            }

            $("#plan_id").append(
              '<option value="' +
                plan.id +
                '"  ' +
                selected +
                ">" +
                plan.name +
                "</option>"
            );
          });
        });
      } else {
        $("#hidden_plan_id").val(
          user.subscription !== null ? user.subscription.plan_id : ""
        );
      }

      // conditionally show and hide partner field depending on the selected role
      $("#roles").change(function() {
        if ($(this).val() == 3 || $(this).val() == 4) {
          $("#partner-container").css("display", "");
        } else {
          $("#partner-container").css("display", "none");
          $("#partner").val("");
        }
      });

      // conditionally enable/disable expiration date field depending on the selected plan
      $("#plan_id").change(function() {
        if ($(this).val() !== "") {
          $("#expiration_date").removeAttr("disabled");

          if ($("#expiration_date").val() == "") {
            $("#expiration_date").val(moment().format("YYYY-MM-DD"));
            $("#full_nameon_date").val(moment().format("YYYY-MM-DD"));
          }
        } else {
          $("#expiration_date").attr("disabled", "disabled");
        }
      });
    });
  } else {
    // create new user
    $(
      ".row.subscription, label[for=current_password], #curpassword, label[for=confirm_password], #confirm_password"
    ).hide();

    api.roles.get().success(function(r) {
      $("#roles")
        .find("option")
        .each(function(i) {
          if ($(this).val() !== "") {
            $(this).remove();
          }
        });

      $.each(r, function(e, rr) {
        $("#roles").append(
          '<option value="' + rr.rid + '" >' + rr.name + "</option>"
        );
      });

      if (r.length == 1) {
        $("#roles option")
          .last()
          .prop("selected", "selected");
      }
    });

    //populate the partners
    api.users.partners().success(function(p) {
      $("#partner")
        .find("option")
        .each(function(i) {
          if ($(this).val() !== "") {
            $(this).remove();
          }
        });

      if (role == "admin") {
        $.each(p, function(e, pp) {
          var selected = "";
          if (partner == pp.id) {
            selected = 'selected = "selected"';
          } else {
            selected = "";
          }

          $("#partner").append(
            '<option value="' +
              pp.id +
              '"  ' +
              selected +
              ">" +
              pp.username +
              "</option>"
          );
        });
      }
    });

    // conditionally show and hide partner field depending on the selected role
    $("#roles").change(function() {
      if ($(this).val() == 3 || $(this).val() == 4) {
        $("#partner-container").css("display", "");
      } else {
        $("#partner-container").css("display", "none");
        $("#partner").val("");
      }
    });

    $("#modalSubmit")
      .unbind("click")
      .click(function() {
        create();
      });
  }
}

function deleteUser(e) {
  swal({
    title: "Are you sure?",
    text: "This will delete the user. This can't be undone!",
    type: "warning",
    animation: !1,
    showCancelButton: !0,
    confirmButtonText: "Delete ",
    confirmButtonColor: "#428bca",
    reverseButtons: !0,
    allowOutsideClick: !1,
    preConfirm: function() {
      return new Promise(function(a, t) {
        api.userId
          .delete(e)
          .success(function(e) {
            a();
          })
          .error(function(e) {
            t(e.responseJSON.message);
          });
      });
    }
  }).then(function() {
    swal("User Deleted!", "This user has been deleted!", "success"),
      $('button:contains("OK")').on("click", function() {
        load();
      });
  });
}

function dismiss() {
  $("#modal\\.flashes").empty();
  $("#username").val("");
  $("#full_name").val("");
  $("#email").val("");
  $("#curpassword").val("");
  $("#password").val("");
  $("#confirm_password").val("");
  $("#partner").val("");
  $("#roles").val("");
  $("#modal").modal("hide");

  $(
    ".row.subscription, label[for=current_password], #curpassword, label[for=confirm_password], #confirm_password"
  ).show();
}

var labels = {
    "In progress": "label-primary",
    Queued: "label-info",
    Completed: "label-success",
    "Emails Sent": "label-success",
    Error: "label-danger"
  },
  campaigns = [],
  campaign = {};

$(document).ready(function() {
  $(".modal").on("hidden.bs.modal", function(e) {
    $(this).removeClass("fv-modal-stack"),
      $("body").data("fv_open_modals", $("body").data("fv_open_modals") - 1);
  }),
    $(".modal").on("shown.bs.modal", function(e) {
      void 0 === $("body").data("fv_open_modals") &&
        $("body").data("fv_open_modals", 0),
        $(this).hasClass("fv-modal-stack") ||
          ($(this).addClass("fv-modal-stack"),
          $("body").data(
            "fv_open_modals",
            $("body").data("fv_open_modals") + 1
          ),
          $(this).css("z-index", 1040 + 10 * $("body").data("fv_open_modals")),
          $(".modal-backdrop")
            .not(".fv-modal-stack")
            .css("z-index", 1039 + 10 * $("body").data("fv_open_modals")),
          $(".modal-backdrop")
            .not("fv-modal-stack")
            .addClass("fv-modal-stack"));
    }),
    $(document).on("hidden.bs.modal", ".modal", function() {
      $(".modal:visible").length && $(document.body).addClass("modal-open");
    }),
    $("#modal").on("hidden.bs.modal", function(e) {
      dismiss();
    });

  setTimeout(function() {
    $("#time_zone.form-control").select2({
      placeholder: "Select Timezone",
      data: moment.tz.names()
    });
  }, 1000);

  load();

  $.fn.select2.defaults.set("width", "100%"),
    $.fn.select2.defaults.set("dropdownParent", $("#modal_body")),
    $.fn.select2.defaults.set("theme", "bootstrap"),
    $.fn.select2.defaults.set("sorter", function(e) {
      return e.sort(function(e, a) {
        return e.text.toLowerCase() > a.text.toLowerCase()
          ? 1
          : e.text.toLowerCase() < a.text.toLowerCase()
          ? -1
          : 0;
      });
    });

  $("#expiration_date").change(function() {
    if ($(this).val() == "") {
      $("#plan_id").val("");
    }
  });
});

function isValidPassword(password) {
  if (password.length < 8) {
    return false;
  }

  if (password.match(/\s/) !== null) {
    return false;
  }

  var alphaMatches = password.match(/([a-zA-Z])/);
  var numMatches = password.match(/([0-9])/);
  var specialMatches = password.match(/([^a-zA-Z0-9\s])/);

  if (
    alphaMatches.length < 2 ||
    numMatches.length < 2 ||
    specialMatches.length < 2
  ) {
    return false;
  }

  return true;
}

function load() {
  if (peopleTable === undefined) {
    peopleTable = $("#peopleTable").DataTable({
      columnDefs: [
        {
          orderable: !1,
          targets: "no-sort"
        },
        { targets: 4, orderData: 6 },
        { targets: 6, visible: false }
      ],
      order: [[4, "desc"]]
    });
  } else {
    peopleTable.clear();
    peopleTable.draw();
  }

  api.users
    .get()
    .success(function(e) {
      people = e;
      $("#loading").hide(),
        people.length > 0
          ? ($("#peopleTable").show(),
            $.each(people, function(i, a) {
              // label = labels[a.status] || "label-default";
              // var t;
              // if (moment(a.launch_date).isAfter(moment())) {
              //     t = "Scheduled to start: " + moment(a.launch_date).format("MMMM Do YYYY, h:mm:ss a");
              //     var n = t + "<br><br>Number of recipients: " + a.stats.total
              // } else {
              //     t = "Launch Date: " + moment(a.launch_date).format("MMMM Do YYYY, h:mm:ss a");
              //     var n = t + "<br><br>Number of recipients: " + a.stats.total + "<br><br>Emails opened: " + a.stats.opened + "<br><br>Emails clicked: " + a.stats.clicked + "<br><br>Submitted Credentials: " + a.stats.submitted_data + "<br><br>Errors : " + a.stats.error + "Reported : " + a.stats.reported
              // }

              peopleTable.row
                .add([
                  '<img style="max-height: 40px" src="' +
                    (a.avatar_id
                      ? "/avatars/" + a.avatar_id
                      : "/images/noavatar.png") +
                    '"> ' +
                    a.username,
                  a.full_name,
                  a.email,
                  a.role,
                  moment(a.last_login_at).year() !== 1
                    ? moment(a.last_login_at).fromNow()
                    : "never",
                  a.subscription
                    ? a.subscription.plan +
                      (a.subscription.expired ? " (expired)" : " ✔")
                    : "✖",
                  moment(a.last_login_at).format("X"),
                  "<div class='pull-right'>" +
                    (role == "admin" ||
                    (role == "partner" && a.role !== "LMS User") ||
                    (role == "child_user" &&
                      a.role !== "Partner" &&
                      a.role !== "Child User" &&
                      a.role !== "LMS User")
                      ? "<span data-toggle='modal' data-backdrop='static' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='' onclick='edit(" +
                        i +
                        ")' data-original-title='Edit Member'>  <i class='fa fa-pencil'></i> </button> </span> " +
                        " <span data-backdrop='static' data-target='#modal'><button class='btn btn-danger' onclick='deleteUser(" +
                        a.id +
                        ")' data-toggle='tooltip' data-placement='left' title='Delete User'> <i class='fa fa-trash-o'></i></button></span>"
                      : "") +
                    "</div>"
                ])
                .draw();
            }))
          : $("#emptyMessage").show();
    })
    .error(function() {
      $("#loading").hide(), errorFlash("Error fetching peoples");
    });
}
