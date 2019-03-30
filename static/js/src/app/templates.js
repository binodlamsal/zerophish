var templateTable;
var filter = "own";

function save(e) {
  var t = {
    attachments: []
  };

  (t.name = $("#name").val()),
    (t.tag = parseInt($("#category").val())),
    (t.public = $("#publicly_available").prop("checked")),
    (t.subject = $("#subject").val()),
    (t.rating = parseInt($("input[name=stars]:checked").val())),
    (t.html = CKEDITOR.instances.html_editor.getData()),
    (t.html = t.html.replace(/https?:\/\/{{\.URL}}/gi, "{{.URL}}")),
    (t.from_address = $("#from_address").val()),
    $("#use_tracker_checkbox").prop("checked")
      ? -1 == t.html.indexOf("{{.Tracker}}") &&
        -1 == t.html.indexOf("{{.TrackingUrl}}") &&
        (t.html = t.html.replace("</body>", "{{.Tracker}}</body>"))
      : (t.html = t.html.replace("{{.Tracker}}</body>", "</body>")),
    (t.text = $("#text_editor").val()),
    $.each(
      $("#attachmentsTable")
        .DataTable()
        .rows()
        .data(),
      function(e, a) {
        t.attachments.push({
          name: unescapeHtml(a[1]),
          content: a[3],
          type: a[4]
        });
      }
    ),
    -1 != e
      ? ((t.id = templates[e].id),
        api.templateId
          .put(t)
          .success(function(e) {
            successFlash("Template edited successfully!"),
              load($("input[type=radio][name=filter]:checked").val()),
              dismiss();
          })
          .error(function(e) {
            modalError(e.responseJSON.message),
            scrollToError()
          }))
      : api.templates
          .post(t)
          .success(function(e) {
            successFlash("Template added successfully!"),
              load($("input[type=radio][name=filter]:checked").val()),
              dismiss();
          })
          .error(function(e) {
            modalError(e.responseJSON.message), scrollToError()
          });
}

function dismiss() {
  $("#modal\\.flashes").empty(),
    $("#attachmentsTable")
      .dataTable()
      .DataTable()
      .clear()
      .draw(),
    $("#name").val(""),
    $("#from_address").val(""),
    $("#subject").val(""),
    $("#text_editor").val(""),
    $("#html_editor").val(""),
    $("#category").val(""),
    $("#modal").modal("hide");
}

function deleteTemplate(e) {
  confirm("Delete " + templates[e].name + "?") &&
    api.templateId.delete(templates[e].id).success(function(e) {
      successFlash(e.message),
        load($("input[type=radio][name=filter]:checked").val());
    });
}

function attach(e) {
  (attachmentsTable = $("#attachmentsTable").DataTable({
    autoWidth: false,
    destroy: !0,
    order: [[1, "asc"]],
    columnDefs: [
      {
        orderable: !1,
        targets: "no-sort"
      },
      {
        sClass: "datatable_hidden",
        targets: [3, 4]
      }
    ]
  })),
    $.each(e, function(e, t) {
      var a = new FileReader();
      (a.onload = function(e) {
        var o = icons[t.type] || "fa-file-o";
        attachmentsTable.row
          .add([
            '<i class="fa ' + o + '"></i>',
            escapeHtml(t.name),
            '<span class="remove-row"><i class="fa fa-trash-o"></i></span>',
            a.result.split(",")[1],
            t.type || "application/octet-stream"
          ])
          .draw();
      }),
        (a.onerror = function(e) {
          console.log(e);
        }),
        a.readAsDataURL(t);
    });
}

function edit(e) {
  $("#modalSubmit")
    .unbind("click")
    .click(function() {
      save(e);
    }),
    $("#attachmentUpload")
      .unbind("click")
      .click(function() {
        this.value = null;
      }),
    $("#html_editor").ckeditor(),
    $("#attachmentsTable").show(),
    (attachmentsTable = $("#attachmentsTable").DataTable({
      autoWidth: false,
      destroy: !0,
      order: [[1, "asc"]],
      columnDefs: [
        {
          orderable: !1,
          targets: "no-sort"
        },
        {
          sClass: "datatable_hidden",
          targets: [3, 4]
        }
      ]
    }));
  var t = {
    attachments: []
  };
  -1 != e &&
    ((t = templates[e]),
    console.log(t),
    $("#publicly_available").prop("checked", t.public),
    $("#name").val(t.name),
    $("#from_address").val(t.from_address),
    $("#subject").val(t.subject),
    $("#html_editor").val(t.html),
    $("#text_editor").val(t.text),
    $.each(t.attachments, function(e, t) {
      var a = icons[t.type] || "fa-file-o";
      attachmentsTable.row
        .add([
          '<i class="fa ' + a + '"></i>',
          escapeHtml(t.name),
          '<span class="remove-row"><i class="fa fa-trash-o"></i></span>',
          t.content,
          t.type || "application/octet-stream"
        ])
        .draw();
    }),
    -1 != t.html.indexOf("{{.Tracker}}")
      ? $("#use_tracker_checkbox").prop("checked", !0)
      : $("#use_tracker_checkbox").prop("checked", !1)),
    $(":radio").prop("checked", false);
  $(":radio[value=" + t.rating + "]").prop("checked", true);

  $("#attachmentsTable")
    .unbind("click")
    .on("click", "span>i.fa-trash-o", function() {
      attachmentsTable
        .row($(this).parents("tr"))
        .remove()
        .draw();
    });

  //fill the categories by the API
  $("#category")
    .find("option")
    .not(":first")
    .remove();
  api.phishtags.get().success(function(s) {
    $.each(s, function(e, ss) {
      var sel = "";
      if (t.tag == ss.id) {
        sel = 'selected = "selected"';
      }

      $("#category").append(
        '<option value="' + ss.id + '"  ' + sel + ">" + ss.name + "</option>"
      );
    });
  });
}

function copy(e) {
  $("#modalSubmit")
    .unbind("click")
    .click(function() {
      save(-1);
    }),
    $("#attachmentUpload")
      .unbind("click")
      .click(function() {
        this.value = null;
      }),
    $("#html_editor").ckeditor(),
    $("#attachmentsTable").show(),
    (attachmentsTable = $("#attachmentsTable").DataTable({
      autoWidth: false,
      destroy: !0,
      order: [[1, "asc"]],
      columnDefs: [
        {
          orderable: !1,
          targets: "no-sort"
        },
        {
          sClass: "datatable_hidden",
          targets: [3, 4]
        }
      ]
    }));
  var t = {
    attachments: []
  };
  (t = templates[e]),
    $("#name").val("Copy of " + t.name),
    $("#subject").val(t.subject),
    $("#from_address").val(t.from_address),
    $("#html_editor").val(t.html),
    $("#text_editor").val(t.text),
    $(":radio").prop("checked", false);
  $(":radio[value=" + t.rating + "]").prop("checked", true);

  $.each(t.attachments, function(e, t) {
    var a = icons[t.type] || "fa-file-o";
    attachmentsTable.row
      .add([
        '<i class="fa ' + a + '"></i>',
        escapeHtml(t.name),
        '<span class="remove-row"><i class="fa fa-trash-o"></i></span>',
        t.content,
        t.type || "application/octet-stream"
      ])
      .draw();
  }),
    $("#attachmentsTable")
      .unbind("click")
      .on("click", "span>i.fa-trash-o", function() {
        attachmentsTable
          .row($(this).parents("tr"))
          .remove()
          .draw();
      }),
    -1 != t.html.indexOf("{{.Tracker}}")
      ? $("#use_tracker_checkbox").prop("checked", !0)
      : $("#use_tracker_checkbox").prop("checked", !1);

  //fill the categories by the API
  $("#category")
    .find("option")
    .not(":first")
    .remove();

  api.phishtags.get().success(function(s) {
    $.each(s, function(e, ss) {
      var sel = "";
      if (t.tag == ss.id) {
        sel = 'selected = "selected"';
      }

      $("#category").append(
        '<option value="' + ss.id + '"  ' + sel + ">" + ss.name + "</option>"
      );
    });
  });
}

function preview(e) {
  t = templates[e];
  console.log(t);
  $("#modalforpreview .tempname").html(t.name);
  $("#modalforpreview .from_address").text(t.from_address);
  $("#modalforpreview .subject").html(t.subject);
  if (t.html != "") {
    $("#modalforpreview .modal-body").html(t.html);
  } else {
    $("#modalforpreview .modal-body").html(t.text);
  }
}

function importEmail() {
  (raw = $("#email_content").val()),
    (convert_links = $("#convert_links_checkbox").prop("checked")),
    raw
      ? api
          .import_email({
            content: raw,
            convert_links: convert_links
          })
          .success(function(e) {
            $("#text_editor").val(e.text),
              $("#html_editor").val(e.html),
              $("#subject").val(e.subject),
              e.html &&
                (CKEDITOR.instances.html_editor.setMode("wysiwyg"),
                $('.nav-tabs a[href="#html"]').click()),
              $("#importEmailModal").modal("hide");
          })
          .error(function(e) {
            modalError(e.responseJSON.message);
          })
      : modalError("No Content Specified!");
}

function load(filter) {
  if (templateTable === undefined) {
    templateTable = $("#templateTable").DataTable({
      autoWidth: false,
      destroy: !0,
      columnDefs: [
        {
          orderable: !1,
          targets: "no-sort"
        }
      ]
    });
    $("#templateTable").show();
  } else {
    templateTable.clear();
    templateTable.draw();
  }

  $("#loading").show(),
    api.templates
      .get(filter)
      .success(function(e) {
        (templates = e),
          $("#loading").hide(),
          templates.length > 0
            ? ($("#templateTable").show(),
              $.each(templates, function(e, t) {
                var rating = "";
                if (t.rating == 5) {
                  rating =
                    "<span> &#9733; &#9733; &#9733; &#9733; &#9733;</span>";
                }
                if (t.rating == 4) {
                  rating =
                    "<span> &#9733; &#9733; &#9733; &#9733; &#9734;</span>";
                }
                if (t.rating == 3) {
                  rating =
                    "<span> &#9733; &#9733; &#9733; &#9734; &#9734;</span>";
                }
                if (t.rating == 2) {
                  rating =
                    "<span> &#9733; &#9733; &#9734; &#9734; &#9734;</span>";
                }
                if (t.rating == 1) {
                  rating =
                    "<span> &#9733; &#9734; &#9734; &#9734; &#9734;</span>";
                }

                templateTable.row
                  .add([
                    escapeHtml(t.name),
                    t.username,
                    rating,
                    moment(t.modified_date).format("MMMM Do YYYY, h:mm:ss a"),
                    "<div class='pull-right'>" +
                      (t.writable
                        ? "<span data-toggle='modal' data-backdrop='static' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Edit Template' onclick='edit(" +
                          e +
                          ")'> <i class='fa fa-pencil'></i></button></span>\t\t"
                        : "") +
                      "<span data-toggle='modal' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Copy Template' onclick='copy(" +
                      e +
                      ")'><i class='fa fa-copy'></i></button></span>  \t\t    " +
                      "<span data-toggle='modal' data-target='#modalforpreview'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Preview Template' onclick='preview(" +
                      e +
                      ")'><i class='fa fa-eye'></i></button></span> \t\t    " +
                      (t.writable
                        ? "<button class='btn btn-danger' data-toggle='tooltip' data-placement='left' title='Delete Template' onclick='deleteTemplate(" +
                          e +
                          ")'><i class='fa fa-trash-o'></i></button>"
                        : "") +
                      "</div>"
                  ])
                  .draw();
              }),
              $('[data-toggle="tooltip"]').tooltip())
            : $("#emptyMessage").hide();
      })
      .error(function() {
        $("#loading").hide(), errorFlash("Error fetching templates");
      });
}
var templates = [],
  icons = {
    "application/vnd.ms-excel": "fa-file-excel-o",
    "text/plain": "fa-file-text-o",
    "image/gif": "fa-file-image-o",
    "image/png": "fa-file-image-o",
    "application/pdf": "fa-file-pdf-o",
    "application/x-zip-compressed": "fa-file-archive-o",
    "application/x-gzip": "fa-file-archive-o",
    "application/vnd.openxmlformats-officedocument.presentationml.presentation":
      "fa-file-powerpoint-o",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
      "fa-file-word-o",
    "application/octet-stream": "fa-file-o",
    "application/x-msdownload": "fa-file-o"
  },
  deleteTemplate = function(e) {
    swal({
      title: "Are you sure?",
      text: "This will delete the template. This can't be undone!",
      type: "warning",
      animation: !1,
      showCancelButton: !0,
      confirmButtonText: "Delete " + escapeHtml(templates[e].name),
      confirmButtonColor: "#428bca",
      reverseButtons: !0,
      allowOutsideClick: !1,
      preConfirm: function() {
        return new Promise(function(t, a) {
          api.templateId
            .delete(templates[e].id)
            .success(function(e) {
              t();
            })
            .error(function(e) {
              a(e.responseJSON.message);
            });
        });
      }
    }).then(function() {
      swal("Template Deleted!", "This template has been deleted!", "success"),
        $('button:contains("OK")').on("click", function() {
          location.reload();
        });
    });
  };
$(document).ready(function() {
  $(".modal").on("hidden.bs.modal", function(e) {
    $(this).removeClass("fv-modal-stack"),
      $("body").data("fv_open_modals", $("body").data("fv_open_modals") - 1);
  }),
    $(".modal").on("shown.bs.modal", function(e) {
      $("#filter-" + filter).prop("checked", true);

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
    ($.fn.modal.Constructor.prototype.enforceFocus = function() {
      $(document)
        .off("focusin.bs.modal")
        .on(
          "focusin.bs.modal",
          $.proxy(function(e) {
            this.$element[0] === e.target ||
              this.$element.has(e.target).length ||
              $(e.target).closest(".cke_dialog, .cke").length ||
              this.$element.trigger("focus");
          }, this)
        );
    }),
    $(document).on("hidden.bs.modal", ".modal", function() {
      $(".modal:visible").length && $(document.body).addClass("modal-open");
    }),
    $("#modal").on("hidden.bs.modal", function(e) {
      dismiss();
    }),
    $("#importEmailModal").on("hidden.bs.modal", function(e) {
      $("#email_content").val("");
    }),
    $("input[type=radio][name=filter]").change(function(event) {
      filter = event.target.value;
      load(filter);
    });

  load("own");
});
