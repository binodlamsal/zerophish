var groupTable;

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
          modalError(e.responseJSON.message);
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
          modalError(e.responseJSON.message);
        });
}

function dismiss() {
  $("#targetsTable")
    .dataTable()
    .DataTable()
    .clear()
    .draw(),
    $("#name").val(""),
    $("#modal\\.flashes").empty();
}

function edit(e) {
  if (
    ((targets = $("#targetsTable").dataTable({
      destroy: !0,
      columnDefs: [
        {
          orderable: !1,
          targets: "no-sort"
        }
      ]
    })),
    $("#modalSubmit")
      .unbind("click")
      .click(function() {
        save(e);
      }),
    -1 == e)
  );
  else
    api.groupId
      .get(e)
      .success(function(e) {
        $("#name").val(e.name),
          $.each(e.targets, function(e, a) {
            targets
              .DataTable()
              .row.add([
                escapeHtml(a.first_name),
                escapeHtml(a.last_name),
                escapeHtml(a.email),
                escapeHtml(a.position),
                '<span style="cursor:pointer;"><i class="fa fa-trash-o"></i></span>'
              ])
              .draw();
          });
      })
      .error(function() {
        errorFlash("Error fetching group");
      });
  $("#csvupload").fileupload({
    url: "/api/import/group?api_key=" + user.api_key,
    dataType: "json",
    add: function(e, a) {
      $("#modal\\.flashes").empty();
      var t = /(csv|txt)$/i,
        s = a.originalFiles[0].name;
      if (s && !t.test(s.split(".").pop()))
        return modalError("Unsupported file extension (use .csv or .txt)"), !1;
      a.submit();
    },
    done: function(e, a) {
      $.each(a.result, function(e, a) {
        addTarget(a.first_name, a.last_name, a.email, a.position);
      }),
        targets.DataTable().draw();
    }
  });
}

function addTarget(e, a, t, s) {
  var o = escapeHtml(t).toLowerCase(),
    r = [
      escapeHtml(e),
      escapeHtml(a),
      o,
      escapeHtml(s),
      '<span style="cursor:pointer;"><i class="fa fa-trash-o"></i></span>'
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
                  escapeHtml(t.name),
                  t.username,
                  escapeHtml(t.num_targets),
                  moment(t.modified_date).format("MMMM Do YYYY, h:mm:ss a"),
                  "<div class='pull-right'><button class='btn btn-primary' data-toggle='modal' data-backdrop='static' data-target='#modal' onclick='edit(" +
                    t.id +
                    ")'>                    <i class='fa fa-pencil'></i>                    </button>                    <button class='btn btn-danger' onclick='deleteGroup(" +
                    t.id +
                    ")'>                    <i class='fa fa-trash-o'></i>                    </button></div>"
                ])
                .draw();
            });
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
    return (
      addTarget(
        $("#firstName").val(),
        $("#lastName").val(),
        $("#email").val(),
        $("#position").val()
      ),
      targets.DataTable().draw(),
      $("#targetForm>div>input").val(""),
      $("#firstName").focus(),
      !1
    );
  }),
    $("#targetsTable").on("click", "span>i.fa-trash-o", function() {
      targets
        .DataTable()
        .row($(this).parents("tr"))
        .remove()
        .draw();
    }),
    $("#modal").on("hide.bs.modal", function() {
      dismiss();
    }),
    $("#csv-template").click(downloadCSVTemplate);
});
