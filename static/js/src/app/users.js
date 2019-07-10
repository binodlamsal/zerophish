var groupTable;
var groupId;

if (!String.prototype.endsWith) {
  Object.defineProperty(String.prototype, "endsWith", {
    value: function(searchString, position) {
      var subjectString = this.toString();
      if (position === undefined || position > subjectString.length) {
        position = subjectString.length;
      }
      position -= searchString.length;
      var lastIndex = subjectString.indexOf(searchString, position);
      return lastIndex !== -1 && lastIndex === position;
    }
  });
}

function save(e) {
  var a = [];
  $.each(
    $("#targetsTable")
      .DataTable()
      .rows()
      .data(),
    function(e, t) {
      a.push({
        first_name: unescapeHtml(t[0]),
        last_name: unescapeHtml(t[1]),
        email: unescapeHtml(t[2]),
        position: unescapeHtml(t[3])
      });
    }
  );
  var t = {
    name: $("#name").val(),
    creator:
      parseInt(
        $("#creator")
          .find(":selected")
          .val()
      ) || 0,
    targets: a
  };
  -1 != e
    ? ((t.id = e),
      api.groupId
        .put(t)
        .success(function(e) {
          successFlash("Group updated successfully!"),
            load($("input[type=radio][name=filter]:checked").val()),
            dismiss(),
            $("#modal").modal("hide");
        })
        .error(function(e) {
          modalError(e.responseJSON.message), scrollToError();
        }))
    : api.groups
        .post(t)
        .success(function(e) {
          successFlash("Group added successfully!"),
            load($("input[type=radio][name=filter]:checked").val()),
            dismiss(),
            $("#modal").modal("hide");
        })
        .error(function(e) {
          modalError(e.responseJSON.message), scrollToError();
        });
}

function dismiss() {
  $("#targetsTable")
    .DataTable()
    .clear()
    .draw();

  $("#name").val("");
  $("#modal\\.flashes").empty();
  $("#lms-modal\\.flashes").empty();
  $("#firstName").val("");
  $("#lastName").val("");
  $("#email").val("");
  $("#position").val("");

  if ($("#creator").length) {
    $("#creator")
      .val("")
      .change();
  }
}

function edit(e) {
  groupId = e;

  targets = $("#targetsTable").dataTable({
    autoWidth: false,
    destroy: !0,
    columnDefs: [
      {
        orderable: !1,
        targets: "no-sort"
      }
    ]
  });

  $("#modalSubmit")
    .unbind("click")
    .click(function() {
      save(e);
    });

  if (-1 == e) {
    $("#modal .modal-title").html("NEW GROUP");

    if ($("#creator").length) {
      $(".form-group[for=creator]").show();

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
      });
    }
  } else {
    $(".form-group[for=creator]").hide();

    api.groupId
      .get(e)
      .success(function(e) {
        $("#modal .modal-title").html("EDIT GROUP"),
          $("#name").val(e.name),
          $.each(e.targets, function(e, a) {
            targets
              .DataTable()
              .row.add([
                escapeHtml(a.first_name),
                escapeHtml(a.last_name),
                escapeHtml(a.email),
                escapeHtml(a.position),
                '<span style="cursor:pointer;" onclick="removeTarget()"><i class="fa fa-trash-o"></i></span>'
              ])
              .draw();
          });
      })
      .error(function() {
        errorFlash("Error fetching group");
      });
  }

  $("#csvupload").fileupload({
    url: "/api/import/group",
    dataType: "json",
    headers: { Authorization: "Bearer " + user.api_key },
    add: function(e, a) {
      $("#modal\\.flashes").empty();
      var t = /(csv|txt)$/i,
        s = a.originalFiles[0].name;
      if (s && !t.test(s.split(".").pop()))
        return modalError("Unsupported file extension (use .csv or .txt)"), !1;
      a.submit();
    },
    done: function(e, a) {
      var skipped = 0;

      $.each(a.result, function(e, a) {
        if (a.email.endsWith(_domain)) {
          addTarget(a.first_name, a.last_name, a.email, a.position);
        } else {
          skipped++;
        }
      });

      if (skipped) {
        modalError(
          skipped +
            " record(s) skipped - make sure all email addresses belong to domain " +
            _domain
        );
      }

      targets.DataTable().draw();
    }
  });
}

function lms(e) {
  groupId = e;

  if (
    ((lmsTargets = $("#lmsTargetsTable").dataTable({
      select: {
        style: "multi",
        selector: "td:first-child"
      },
      destroy: !0,
      columnDefs: [
        {
          orderable: false,
          className: "select-checkbox",
          width: "20px",
          targets: 0
        },
        {
          width: "30px",
          targets: -1
        }
      ],
      order: [[1, "asc"]],
      autoWidth: false
    })),
    -1 == e)
  );
  else {
    lmsTargets
      .DataTable()
      .rows()
      .deselect();

    lmsTargets.DataTable().clear();

    api.groupId
      .get(e)
      .success(function(e) {
        $.each(e.targets, function(e, a) {
          lmsTargets
            .DataTable()
            .row.add([
              "",
              escapeHtml(a.first_name),
              escapeHtml(a.last_name),
              escapeHtml(a.email),
              escapeHtml(a.position),
              a.is_lms_user ? "&nbsp;&nbsp;✔" : ""
            ])
            .node().id = a.id;
        });

        lmsTargets.DataTable().draw();

        $(".dataTables_empty").attr(
          "colspan",
          lmsTargets
            .DataTable()
            .columns()
            .count()
        );
      })
      .error(function() {
        errorFlash("Error fetching group");
      });
  }
}

function addTarget(e, a, t, s) {
  var o = escapeHtml(t).toLowerCase(),
    r = [
      escapeHtml(e),
      escapeHtml(a),
      o,
      escapeHtml(s),
      '<span style="cursor:pointer;" onclick="removeTarget()"><i class="fa fa-trash-o"></i></span>'
    ],
    n = targets.DataTable(),
    i = n
      .column(2, {
        order: "index"
      })
      .data()
      .indexOf(o);
  i >= 0
    ? n
        .row(i, {
          order: "index"
        })
        .data(r)
    : n.row.add(r);
}

function load(filter) {
  if (groupTable === undefined) {
    groupTable = $("#groupTable").DataTable({
      autoWidth: false,
      destroy: !0,
      columnDefs: [
        {
          orderable: !1,
          targets: "no-sort"
        }
      ]
    });

    $("#groupTable").show();
  } else {
    groupTable.clear();
    groupTable.draw();
  }

  $("#emptyMessage").hide(),
    $("#loading").show(),
    api.groups
      .summary(filter)
      .success(function(e) {
        if (($("#loading").hide(), e.total > 0)) {
          (groups = e.groups),
            $("#emptyMessage").hide(),
            $.each(groups, function(e, t) {
              groupTable.row
                .add([
                  escapeHtml(t.name) +
                    (t.locked
                      ? ' <i class="fa fa-lock" data-toggle="tooltip" data-placement="right" data-original-title="Your subscription has expired, therefore your user groups have been locked. Please contact your account manager to extend your subscription."></i>'
                      : ""),
                  t.username,
                  escapeHtml(t.num_targets),
                  moment(t.modified_date).format("MMMM Do YYYY, h:mm:ss a"),
                  t.locked
                    ? ""
                    : "<div class='pull-right'>" +
                      (isSubscribed || _role == "admin"
                        ? "<span data-toggle='modal' data-backdrop='static' data-target='#lms-modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' data-original-title='Create Training Users' onclick='lms(" +
                          t.id +
                          ")'>LMS</button></span>"
                        : "") +
                      "<span data-toggle='modal' data-backdrop='static' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' data-original-title='Edit Group'  onclick='edit(" +
                      t.id +
                      ")'>                    <i class='fa fa-pencil'></i>                    </button></span>                    <button class='btn btn-danger' data-toggle='tooltip' data-placement='left' data-original-title='Delete' onclick='deleteGroup(" +
                      t.id +
                      ")'>                    <i class='fa fa-trash-o'></i>                    </button></div>"
                ])
                .draw();
            });
          $('[data-toggle="tooltip"]').tooltip();
        } else $("#emptyMessage").hide();
      })
      .error(function() {
        errorFlash("Error fetching groups");
      });
}
var groups = [],
  downloadCSVTemplate = function() {
    var e = [
        {
          "First Name": "Example",
          "Last Name": "User",
          Email: "foobar@example.com",
          Position: "Systems Administrator"
        }
      ],
      a = Papa.unparse(e, {}),
      t = new Blob([a], {
        type: "text/csv;charset=utf-8;"
      });
    if (navigator.msSaveBlob) navigator.msSaveBlob(t, "group_template.csv");
    else {
      var s = window.URL.createObjectURL(t),
        o = document.createElement("a");
      (o.href = s),
        o.setAttribute("download", "group_template.csv"),
        document.body.appendChild(o),
        o.click(),
        document.body.removeChild(o);
    }
  },
  deleteGroup = function(e) {
    var a = groups.find(function(a) {
      return a.id === e;
    });
    a &&
      swal({
        title: "Are you sure?",
        text: "This will delete the group. This can't be undone!",
        type: "warning",
        animation: !1,
        showCancelButton: !0,
        confirmButtonText: "Delete " + escapeHtml(a.name),
        confirmButtonColor: "#428bca",
        reverseButtons: !0,
        allowOutsideClick: !1,
        preConfirm: function() {
          return new Promise(function(a, t) {
            api.groupId
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
        swal("Group Deleted!", "This group has been deleted!", "success"),
          $('button:contains("OK")').on("click", function() {
            location.reload();
          });
      });
  };
$(document).ready(function() {
  const params = new URLSearchParams(document.location.search);

  if (params.get("ref") === "campaigns") {
    errorFlash("Please create a user group first");
  }

  $("input[type=radio][name=filter]").change(function(event) {
    load(event.target.value);
  });

  load("own");

  $("#targetForm").submit(function() {
    if ((_role == "partner" || _role == "child_user") && !_domain) {
      modalError(
        'Please set your domain in the <a href="/settings">settings</a> first!'
      );

      return false;
    }

    var group = groups.find(function(g) {
      return g.id == groupId;
    });

    var isOwner = groupId == -1 || (group && group.username === user.username);

    if (
      !$("#creator")
        .find(":selected")
        .val() &&
      isOwner
    ) {
      if (
        !$("#email")
          .val()
          .endsWith(_domain)
      ) {
        modalError(
          "You may only add email addresses on your own domain (like user@" +
            _domain +
            ")"
        );
        return false;
      }
    }

    addTarget(
      $("#firstName").val(),
      $("#lastName").val(),
      $("#email").val(),
      $("#position").val()
    );

    targets.DataTable().draw();
    $("#targetForm input").val("");
    $("#firstName").focus();
    return false;
  });

  $("#modal").on("hide.bs.modal", function() {
    dismiss();
  });

  $("#lms-modal").on("hide.bs.modal", function() {
    dismiss();
  });

  $("#csv-template").click(downloadCSVTemplate);

  $("#toggle-all").change(function() {
    if ($("#toggle-all").prop("checked")) {
      lmsTargets
        .DataTable()
        .rows()
        .select();
    } else {
      lmsTargets
        .DataTable()
        .rows()
        .deselect();
    }
  });

  $("#lmsTargetsTable")
    .DataTable()
    .on("deselect", function(e, dt, type, indexes) {
      if (type === "row") {
        $("#toggle-all").prop("checked", false);
      }
    });

  $("#lmsTargetsTable")
    .DataTable()
    .on("select", function(e, dt, type, indexes) {
      if (type === "row") {
        var lmsTable = $("#lmsTargetsTable").DataTable();

        if (lmsTable.rows(".selected").count() === lmsTable.rows().count()) {
          $("#toggle-all").prop("checked", true);
        }
      }
    });

  $("#create-users").click(function() {
    $(".lms-buttons > button").prop("disabled", "disabled");

    var ids = lmsTargets
      .DataTable()
      .rows({ selected: true })
      .nodes()
      .map(function(n) {
        return parseInt(n.id);
      })
      .toArray();

    api.groupId.lms
      .post(groupId, ids)
      .success(function(resp) {
        if (resp.success) {
          var jobId = resp.data;

          var setProgress = function(progress) {
            if ($("#lms-progress-container").is(":hidden")) {
              $("#lms-spinner").toggle();
              $("#lms-progress-container").toggle();
            }

            $("#lms-progress-bar").width(progress + "%");

            if (progress == 100) {
              $("#lms-spinner").toggle();

              setTimeout(function() {
                $("#lms-progress-container").toggle();
              }, 500);
            }
          };

          var pollJob = function() {
            api.groupId.lms.jobs
              .get(groupId, jobId)
              .success(function(resp) {
                if (resp.success && resp.data.progress < 100) {
                  setProgress(resp.data.progress);
                  setTimeout(pollJob, 2000);
                } else if (resp.data.progress == 100) {
                  setProgress(resp.data.progress);
                  $(".lms-buttons > button").removeProp("disabled");
                  lms(groupId);

                  if (resp.data.errors.length == 0) {
                    delayedAlert("LMS user(s) created successfully");
                  } else {
                    delayedAlert(
                      "There was/were " +
                        resp.data.errors.length +
                        " erros(s)\n\n" +
                        resp.data.errors.join("\n")
                    );
                  }
                } else {
                  console.log("Job not found");
                }
              })
              .error(function(resp) {
                $(".lms-buttons > button").removeProp("disabled");

                if (
                  resp.responseJSON !== undefined &&
                  resp.responseJSON.message !== undefined
                ) {
                  delayedAlert(resp.responseJSON.message);
                } else {
                  delayedAlert("Something went wrong!");
                }

                lms(groupId);
              });
          };

          pollJob();
        }
      })
      .error(function(e) {
        $(".lms-buttons > button").removeProp("disabled");
        modalError(e.responseJSON.message);
      });
  });

  $("#delete-users").click(function() {
    $(".lms-buttons > button").prop("disabled", "disabled");

    var ids = lmsTargets
      .DataTable()
      .rows({ selected: true })
      .nodes()
      .map(function(n) {
        return parseInt(n.id);
      })
      .toArray();

    api.groupId.lms
      .delete(groupId, ids)
      .success(function(resp) {
        if (resp.success) {
          var jobId = resp.data;

          var setProgress = function(progress) {
            if ($("#lms-progress-container").is(":hidden")) {
              $("#lms-spinner").toggle();
              $("#lms-progress-container").toggle();
            }

            $("#lms-progress-bar").width(progress + "%");

            if (progress == 100) {
              $("#lms-spinner").toggle();

              setTimeout(function() {
                $("#lms-progress-container").toggle();
              }, 500);
            }
          };

          var pollJob = function() {
            api.groupId.lms.jobs
              .get(groupId, jobId)
              .success(function(resp) {
                if (resp.success && resp.data.progress < 100) {
                  setProgress(resp.data.progress);
                  setTimeout(pollJob, 2000);
                } else if (resp.data.progress == 100) {
                  setProgress(resp.data.progress);
                  $(".lms-buttons > button").removeProp("disabled");
                  lms(groupId);

                  if (resp.data.errors.length == 0) {
                    delayedAlert("LMS user(s) deleted successfully");
                  } else {
                    delayedAlert(
                      "There was/were " +
                        resp.data.errors.length +
                        " erros(s)\n\n" +
                        resp.data.errors.join("\n")
                    );
                  }
                } else {
                  console.log("Job not found");
                }
              })
              .error(function(resp) {
                $(".lms-buttons > button").removeProp("disabled");

                if (
                  resp.responseJSON !== undefined &&
                  resp.responseJSON.message !== undefined
                ) {
                  delayedAlert(resp.responseJSON.message);
                } else {
                  delayedAlert("Something went wrong!");
                }

                lms(groupId);
              });
          };

          pollJob();
        }
      })
      .error(function(e) {
        $(".lms-buttons > button").removeProp("disabled");
        modalError(e.responseJSON.message);
      });
  });

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

function delayedAlert(message) {
  setTimeout(function() {
    alert(message);
  }, 500);
}

function removeTarget() {
  $("#targetsTable")
    .DataTable()
    .row($(event.target).parents("tr"))
    .remove()
    .draw();
}
