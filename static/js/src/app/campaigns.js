var campaignTable;
var categories = [];

function launch() {
  console.log($("#start_time").val());
  swal({
    title: "Are you sure?",
    text: "This will schedule the campaign to be launched.",
    type: "question",
    animation: !1,
    showCancelButton: !0,
    confirmButtonText: "Launch",
    confirmButtonColor: "#428bca",
    reverseButtons: !0,
    allowOutsideClick: !1,
    showLoaderOnConfirm: !0,
    preConfirm: function() {
      return new Promise(function(e, a) {
        (groups = []),
          $("#users")
            .select2("data")
            .forEach(function(e) {
              groups.push({
                id: parseInt(e.id),
                name: e.text
              });
            });
        var t = $("#send_by_date").val();
        "" != t &&
          (t = moment(t, "MM/DD/YYYY hh:mm a")
            .utc()
            .format()),
          (campaign = {
            name: $("#name").val(),
            from_address: $("#from_address").val(),
            template: {
              id: parseInt($("#template").select2("data")[0].id),
              name: $("#template").select2("data")[0].text
            },
            url: $("#url").val(),
            page: {
              name: $("#page").select2("data")[0].text
            },
            smtp: {
              name: $("#profile").select2("data")[0].id
            },
            launch_date: moment($("#launch_date").val(), "MM/DD/YYYY hh:mm a")
              .utc()
              .format(),
            send_by_date: t || null,
            groups: groups,
            group_id: parseInt($("#users").select2("data")[0].id),
            start_time: $("#during_certain_hours_checkbox").prop("checked")
              ? $("#start_time").val()
              : "",
            end_time: $("#during_certain_hours_checkbox").prop("checked")
              ? $("#end_time").val()
              : "",
            time_zone: $("#during_certain_hours_checkbox").prop("checked")
              ? $("#time_zone").val()
              : "",
            remove_non_clickers: $("#remove_nonclickers_checkbox").prop(
              "checked"
            ),
            clickers_group_id:
              parseInt(
                $("#clickers_group_id")
                  .find(":selected")
                  .val()
              ) || 0,
            clickers_group: $("#clickers_group").val(),
            creator:
              parseInt(
                $("#creator")
                  .find(":selected")
                  .val()
              ) || 0
          });

        if (
          $("#during_certain_hours_checkbox").prop("checked") &&
          (!campaign.start_time || !campaign.end_time || !campaign.time_zone)
        ) {
          $("#modal\\.flashes")
            .empty()
            .append(
              '<div style="text-align:center" class="alert alert-danger">            <i class="fa fa-exclamation-circle"></i> ' +
                "Start/End Time and/or Time Zone not specified" +
                "</div>"
            );
          scrollToError();
          swal.close();
          return;
        }

        if (
          campaign.start_time &&
          campaign.end_time &&
          moment(campaign.end_time, "h:mm A").isBefore(
            moment(campaign.start_time, "h:mm A")
          )
        ) {
          $("#modal\\.flashes")
            .empty()
            .append(
              '<div style="text-align:center" class="alert alert-danger">            <i class="fa fa-exclamation-circle"></i> ' +
                "The End Time cannot be earlier than the Start Time" +
                "</div>"
            );
          scrollToError();
          swal.close();
          return;
        }

        api.campaigns
          .post(campaign)
          .success(function(a) {
            e(), (campaign = a);
          })
          .error(function(e) {
            $("#modal\\.flashes")
              .empty()
              .append(
                '<div style="text-align:center" class="alert alert-danger">            <i class="fa fa-exclamation-circle"></i> ' +
                  e.responseJSON.message +
                  "</div>"
              ),
              scrollToError();
            swal.close();
          });
      });
    }
  }).then(function() {
    swal(
      "Campaign Scheduled!",
      "This campaign has been scheduled for launch!",
      "success"
    );

    window.localStorage.setItem("NewCampaign", "true");

    $('button:contains("OK")').on("click", function() {
      window.location = "/campaigns/" + campaign.id.toString();
    });
  });
}

function sendTestEmail() {
  var e = {
    template: {
      name: $("#template").select2("data")[0].text
    },
    first_name: $("input[name=to_first_name]").val(),
    last_name: $("input[name=to_last_name]").val(),
    from_address: $("#from_address").val(),
    email: $("input[name=to_email]").val(),
    position: $("input[name=to_position]").val(),
    url: $("#url").val(),
    page: {
      name: $("#page").select2("data")[0].text
    },
    smtp: {
      name: $("#profile").select2("data")[0].id
    }
  };

  (btnHtml = $("#sendTestModalSubmit").html()),
    $("#sendTestModalSubmit").html(
      '<i class="fa fa-spinner fa-spin"></i> Sending'
    ),
    api
      .send_test_email(e)
      .success(function(e) {
        $("#sendTestEmailModal\\.flashes")
          .empty()
          .append(
            '<div style="text-align:center" class="alert alert-success">            <i class="fa fa-check-circle"></i> Email Sent!</div>'
          ),
          $("#sendTestModalSubmit").html(btnHtml);
      })
      .error(function(e) {
        $("#sendTestEmailModal\\.flashes")
          .empty()
          .append(
            '<div style="text-align:center" class="alert alert-danger">            <i class="fa fa-exclamation-circle"></i> ' +
              e.responseJSON.message +
              "</div>"
          ),
          $("#sendTestModalSubmit").html(btnHtml);
      });
}

function dismiss() {
  $("#modal\\.flashes").empty();
  $("#name").val("");

  if ($("#creator").length) {
    $("#creator")
      .val("")
      .change();
  }

  $("#template").select2("data", null),
    $("#page")
      .val("")
      .change(),
    $("#url").val(""),
    $("#profile")
      .val("")
      .change(),
    $("#users")
      .val("")
      .change(),
    $("#clickers_group_id")
      .val("")
      .change(),
    $("#clickers_group").val(""),
    $("#time_zone")
      .val("")
      .change();
  if ($("#during_certain_hours_checkbox").prop("checked")) {
    $("#during_certain_hours_checkbox").click();
  }
  $("#modal").modal("hide");
}

function dismissPreview() {
  $("#modalforpreview").modal("hide");
}

function deleteCampaign(e) {
  swal({
    title: "Are you sure?",
    text: "This will delete the campaign. This can't be undone!",
    type: "warning",
    animation: !1,
    showCancelButton: !0,
    confirmButtonText: "Delete " + campaigns[e].name,
    confirmButtonColor: "#428bca",
    reverseButtons: !0,
    allowOutsideClick: !1,
    preConfirm: function() {
      return new Promise(function(a, t) {
        api.campaignId
          .delete(campaigns[e].id)
          .success(function(e) {
            a();
          })
          .error(function(e) {
            t(e.responseJSON.message);
          });
      });
    }
  }).then(function() {
    swal("Campaign Deleted!", "This campaign has been deleted!", "success"),
      $('button:contains("OK")').on("click", function() {
        location.reload();
      });
  });
}

function setupOptions() {
  if ($("#creator").length) {
    api.users.get().success(function(r) {
      $("#creator.form-control").select2({
        placeholder: "You (" + user.username + ")",
        allowClear: true,
        data: r
          .map(function(user) {
            return { id: user.id, text: user.username, role: user.role };
          })
          .filter(function(_user) {
            return (
              _user.text !== user.username &&
              _user.role !== "LMS User" &&
              _user.role !== "Child User" &&
              _user.role !== "Partner" &&
              _user.role !== "Administrator"
            );
          })
      });

      $("#creator.form-control")
        .off("select2:select")
        .on("select2:select", function(e) {
          loadTemplates("public-and-uid-" + e.params.data.id);
          loadPages("public-and-uid-" + e.params.data.id);
          loadGroups("uid-" + e.params.data.id, false);
        });

      $("#creator.form-control")
        .off("select2:unselect")
        .on("select2:unselect", function(e) {
          loadTemplates("own-and-public");
          loadPages("own-and-public");
          loadGroups("own", false);
        });
    });
  }

  loadGroups("own", true);

  if (!$("#template.form-control").hasClass("select2-hidden-accessible")) {
    loadTemplates("own-and-public");
  }

  if (!$("#page.form-control").hasClass("select2-hidden-accessible")) {
    loadPages("own-and-public");
  }

  api.SMTP.domains().success(function(e) {
    if (0 == e.length) return modalError("No profiles found!"), !1;
    var a = $.map(e, function(e) {
        return ((e.id = e.name), (e.text = e.name + " (" + e.host + ")")), e;
      }),
      t = $("#profile.form-control");

    t
      .select2({
        placeholder: "Select a Sending Profile",
        data: a
      })
      .select2("val", a[0]),
      1 === e.length && (t.val(a[0].id), t.trigger("change.select2"));
  });
}

function loadGroups(filter, redirect) {
  api.groups.summary(filter).success(function(e) {
    if (redirect) {
      if (0 == e.total) return (document.location = "/users?ref=campaigns"), !1;
    }

    var a = $.map(e.groups, function(e) {
      return (e.text = e.name), e;
    });
    $("#users.form-control")
      .empty()
      .select2({
        placeholder: "Select Groups",
        data: a
      });

    $("#clickers_group_id")
      .empty()
      .select2({
        placeholder: "Existing group...",
        data: a,
        allowClear: true
      })
      .val(null)
      .trigger("change");

    $("#clickers_group_id")
      .off("select2:select")
      .on("select2:select", function(e) {
        $("#clickers_group").val("");
      });

    $("#clickers_group")
      .off("keyup")
      .on("keyup", function(e) {
        $("#clickers_group_id")
          .val(null)
          .trigger("change");
      });
  });
}

function loadTemplates(filter) {
  var addresses = {};
  var pages = {};
  var data = [];

  api.templates.get(filter).success(function(e) {
    if (e.length > 0) {
      var a = $.map(e, function(e) {
        addresses[e.id] = e.from_address;
        pages[e.id] = e.default_page_id;
        return (e.text = e.name), e;
      });

      data = a
        .map(function(t) {
          return {
            id: t.id,
            text: t.name,
            category: categories.find(function(c) {
              return c.id === t.tag;
            })
          };
        })
        .reduce(function(groups, t) {
          if (t.category !== undefined) {
            if (groups[t.category.name] !== undefined) {
              groups[t.category.name].push(t);
            } else {
              groups[t.category.name] = [t];
            }
          } else {
            groups["Misc"] !== undefined
              ? groups["Misc"].push(t)
              : (groups["Misc"] = [t]);
          }

          return groups;
        }, {});

      data = Object.keys(data).map(function(group) {
        children = data[group];

        if (children.length > 1) {
          children.unshift({
            id: 1000000 + children[0].category.id,
            text: "RANDOM (" + children[0].category.name + ")"
          });
        }

        return {
          text: group,
          children: children
        };
      });
    }

    $("#template.form-control").change(function(event) {
      $("#from_address").val(addresses[event.target.value]);
      if (
        pages[event.target.value] !== 0 &&
        pages[event.target.value] !== undefined
      ) {
        $("#page.form-control").val(pages[event.target.value]);
        $("#page.form-control").trigger("change.select2");
      }

      if ($(this).val() !== "" && $(this).val() != 0) {
        $("#preview-btn").prop("disabled", "");
      } else {
        $("#preview-btn").prop("disabled", "disabled");
      }
    });

    $("#template.form-control")
      .empty()
      .select2({
        placeholder: "Select a Template",
        data: data
      });

    if (e.length === 1) {
      $("#template.form-control").val(a[0].id);
      $("#template.form-control").trigger("change.select2");
      $("#preview-btn").prop("disabled", "");
    }
  });
}

function loadPages(filter) {
  var data = [];

  api.pages.get(filter).success(function(e) {
    if (e.length > 0) {
      var a = $.map(e, function(e) {
        return (e.text = e.name), e;
      });

      data = a
        .map(function(p) {
          return {
            id: p.id,
            text: p.name,
            category: categories.find(function(c) {
              return c.id === p.tag;
            })
          };
        })
        .reduce(function(groups, p) {
          if (p.category !== undefined) {
            if (groups[p.category.name] !== undefined) {
              groups[p.category.name].push(p);
            } else {
              groups[p.category.name] = [p];
            }
          } else {
            groups["Misc"] !== undefined
              ? groups["Misc"].push(p)
              : (groups["Misc"] = [p]);
          }

          return groups;
        }, {});

      data = Object.keys(data).map(function(group) {
        return {
          text: group,
          children: data[group]
        };
      });
    }

    $("#page.form-control")
      .empty()
      .select2({
        placeholder: "Select a Landing Page",
        data: data
      });

    1 === e.length &&
      ($("#page.form-control").val(a[0].id),
      $("#page.form-control").trigger("change.select2"));
  });
}

function edit(e) {
  $("#modal .modal-title").html("NEW CAMPAIGN"), setupOptions();
}

function copy(e) {
  $("#modal .modal-title").html("COPY CAMPAIGN"),
    setupOptions(),
    api.campaignId
      .get(campaigns[e].id)
      .success(function(e) {
        $("#name").val("Copy of " + e.name),
          e.template.id
            ? ($("#template").val(e.template.id.toString()),
              $("#template").trigger("change.select2"))
            : $("#template").select2({
                placeholder: e.template.name
              }),
          e.page.id
            ? ($("#page").val(e.page.id.toString()),
              $("#page").trigger("change.select2"))
            : $("#page").select2({
                placeholder: e.page.name
              }),
          e.smtp.id
            ? ($("#profile").val(e.smtp.name.toString()),
              $("#profile").trigger("change.select2"))
            : $("#profile").select2({
                placeholder: e.smtp.name
              }),
          $("#url").val(e.url);
      })
      .error(function(e) {
        $("#modal\\.flashes")
          .empty()
          .append(
            '<div style="text-align:center" class="alert alert-danger">            <i class="fa fa-exclamation-circle"></i> ' +
              e.responseJSON.message +
              "</div>"
          );
      });
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
  $("input[type=radio][name=filter]").change(function(event) {
    var filter = event.target.value;
    sessionStorage.setItem("CampaignFilter", filter);
    load(filter);
  });

  $("#during_certain_hours_checkbox").change(function(event) {
    if ($(this).prop("checked")) {
      $("#certain_hours input, #certain_hours select").prop("disabled", "");
    } else {
      $("#certain_hours input, #certain_hours select").prop(
        "disabled",
        "disabled"
      );
    }
  });

  api.phishtags.get().success(function(_categories) {
    categories = _categories;
  });

  setTimeout(function() {
    $("#time_zone.form-control").select2({
      placeholder: "Select Timezone",
      data: moment.tz.names()
    });

    if (_timezone) {
      $("#time_zone")
        .val(_timezone)
        .trigger("change");
    }
  }, 1000);

  $("#launch_date").datetimepicker({
    widgetPositioning: {
      vertical: "bottom"
    },
    showTodayButton: !0,
    defaultDate: moment(),
    collapse: false
  }),
    $("#send_by_date").datetimepicker({
      widgetPositioning: {
        vertical: "bottom"
      },
      showTodayButton: !0,
      useCurrent: !1,
      collapse: false
    }),
    $("#start_time").datetimepicker({
      format: "LT"
    }),
    $("#end_time").datetimepicker({
      format: "LT"
    }),
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

  var filter = sessionStorage.getItem("CampaignFilter") || "own";

  if ($("input[type=radio][name=filter]").val() != filter) {
    $("input[type=radio][name=filter]").val([filter]);
  }

  load(filter);

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
});

function load(filter) {
  if (campaignTable === undefined) {
    campaignTable = $("#campaignTable").DataTable({
      autoWidth: false,
      columnDefs: [
        {
          orderable: !1,
          targets: "no-sort"
        },
        { targets: 2, orderData: 5 },
        { targets: 5, visible: false }
      ],
      order: [[2, "desc"]],
      deferRender: true
    });
  } else {
    campaignTable.clear();
    campaignTable.draw();
  }

  api.campaigns
    .summary(filter)
    .success(function(e) {
      (campaigns = e.campaigns),
        $("#loading").hide(),
        campaigns.length > 0
          ? ($("#campaignTable").show(),
            $.each(campaigns, function(e, a) {
              label = labels[a.status] || "label-default";
              var t;
              if (moment(a.launch_date).isAfter(moment())) {
                t =
                  "Scheduled to start: " +
                  moment(a.launch_date).format("MMMM Do YYYY, h:mm:ss a");
                var n = t + "<br><br>Number of recipients: " + a.stats.total;
              } else {
                t =
                  "Launch Date: " +
                  moment(a.launch_date).format("MMMM Do YYYY, h:mm:ss a");
                var n =
                  t +
                  "<br><br>Number of recipients: " +
                  a.stats.total +
                  "<br><br>Emails opened: " +
                  a.stats.opened +
                  "<br><br>Emails clicked: " +
                  a.stats.clicked +
                  "<br><br>Submitted Credentials: " +
                  a.stats.submitted_data +
                  "<br><br>Errors: " +
                  a.stats.error +
                  "<br><br>Reported: " +
                  a.stats.email_reported +
                  "<br><br>Phish Risk: " +
                  Math.round((a.stats.clicked / a.stats.total) * 100) +
                  "%";
              }
              campaignTable.row
                .add([
                  escapeHtml(a.name) +
                    (a.locked
                      ? ' <i class="fa fa-lock" data-toggle="tooltip" data-placement="right" data-original-title="Your subscription has expired, therefore your campagains have been locked. Please contact your account manager to extend your subscription."></i>'
                      : ""),
                  a.username,
                  moment(a.created_date).format("MMMM Do YYYY, h:mm:ss a"),
                  '<span class="label ' +
                    label +
                    '" data-toggle="tooltip" data-placement="right" data-html="true" title="' +
                    n +
                    '">' +
                    a.status +
                    "</span>",
                  a.locked
                    ? ""
                    : "<div class='pull-right'><a class='btn btn-primary' href='/campaigns/" +
                      a.id +
                      "' data-toggle='tooltip' data-placement='left' title='View Results'>                    <i class='fa fa-bar-chart'></i>                    </a>            <span data-toggle='modal' data-backdrop='static' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Copy Campaign' onclick='copy(" +
                      e +
                      ")'>                    <i class='fa fa-copy'></i>                    </button></span>                    <button class='btn btn-danger' onclick='deleteCampaign(" +
                      e +
                      ")' data-toggle='tooltip' data-placement='left' title='Delete Campaign'>                    <i class='fa fa-trash-o'></i>                    </button></div>",
                  moment(a.created_date).format("X")
                ])
                .draw(),
                $('[data-toggle="tooltip"]').tooltip();
            }))
          : $("#emptyMessage").hide();
    })
    .error(function() {
      $("#loading").hide(), errorFlash("Error fetching campaigns");
    });
}

function preview() {
  $("#modalforpreview").modal("show");

  if ($("#preview-btn").prop("disabled")) {
    return;
  }

  api.templateId.get($("#template").select2("data")[0].id).success(function(t) {
    $("#modalforpreview .tempname").html(t.name);
    $("#modalforpreview .from_address").text(
      $("#from_address").val() ||
        t.from_address ||
        "First Last <first.last@test.com>"
    );
    $("#modalforpreview .subject").html(t.subject);

    api.auth.lak
      .get("/api/templates/" + t.id + "/preview")
      .success(function(r) {
        if (!r.success || r.data == null) {
          errorFlash("Could not retrieve access key for template preview");
          return;
        }

        $("#modalforpreview .modal-body iframe").prop(
          "src",
          "/api/templates/" + t.id + "/preview?access_key=" + r.data
        );
      });
  });
}
