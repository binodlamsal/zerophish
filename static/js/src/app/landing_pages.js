var pagesTable;

function save(e) {
  var a = {};
  (a.name = $("#name").val()),
    (a.tag = parseInt($("#category").val())),
    (a.public = $("#publicly_available").prop("checked")),
    (a.shared = $("#shared").prop("checked")),
    (editor = CKEDITOR.instances.html_editor),
    (a.html = editor.getData()),
    (a.capture_credentials = false),
    (a.capture_passwords = false),
    (a.redirect_url = $("#redirect_url_input").val()),
    -1 != e
      ? ((a.id = pages[e].id),
        api.pageId.put(a).success(function(e) {
          successFlash("Page edited successfully!"),
            load($("input[type=radio][name=filter]:checked").val()),
            dismiss();
        }))
      : api.pages
          .post(a)
          .success(function(e) {
            successFlash("Page added successfully!"), load("own"), dismiss();
          })
          .error(function(e) {
            modalError(e.responseJSON.message), scrollToError();
          });
}

function dismiss() {
  $("#modal\\.flashes").empty(),
    $("#name").val(""),
    $("#html_editor").val(""),
    $("#url").val(""),
    $("#redirect_url_input").val(""),
    $("#modal")
      .find("input[type='checkbox']")
      .prop("checked", !1),
    $("#modal").modal("hide");
}

function importSite() {
  (url = $("#url").val()),
    url
      ? api
          .clone_site({
            url: url,
            include_resources: !1
          })
          .success(function(e) {
            $("#html_editor").val(e.html),
              CKEDITOR.instances.html_editor.setMode("wysiwyg"),
              $("#importSiteModal").modal("hide");
          })
          .error(function(e) {
            modalError(e.responseJSON.message);
          })
      : modalError("No URL Specified!");
}

function edit(e) {
  $("#modalSubmit")
    .unbind("click")
    .click(function() {
      save(e);
    }),
    $("#modal .modal-title").html("NEW LANDING PAGE"),
    $("#html_editor").ckeditor();
  var a = {};
  -1 != e &&
    ((a = pages[e]),
    $("#modal .modal-title").html("EDIT LANDING PAGE"),
    $("#name").val(a.name),
    $("#html_editor").val(a.html),
    $("#publicly_available").prop("checked", a.public),
    $("#shared").prop("checked", a.shared),
    $("#redirect_url_input").val(a.redirect_url));

  $("#category.form-control").val(a.tag);
  $("#category.form-control").trigger("change.select2");
}

function preview(e) {
  p = pages[e];
  console.log(p);
  $("#modalforpreview .pagename").html(p.name);

  api.auth.lak.get("/api/pages/" + p.id + "/preview").success(function(r) {
    if (!r.success || r.data == null) {
      errorFlash("Could not retrieve access key for page preview");
      return;
    }

    $("#modalforpreview .modal-body iframe").prop(
      "src",
      "/api/pages/" + p.id + "/preview?access_key=" + r.data
    );
  });
}

function copy(e) {
  $("#modalSubmit")
    .unbind("click")
    .click(function() {
      save(-1);
    }),
    $("#modal .modal-title").html("COPY LANDING PAGE"),
    $("#html_editor").ckeditor();
  var a = pages[e];
  $("#name").val("Copy of " + a.name), $("#html_editor").val(a.html);
  $("#category.form-control")
    .val(null)
    .trigger("change.select2");
}

function load(filter) {
  if ($("input[type=radio][name=filter]:checked").val() !== filter) {
    $("input[type=radio][name=filter][value=" + filter + "]").prop(
      "checked",
      true
    );
  }

  if (pagesTable === undefined) {
    pagesTable = $("#pagesTable").DataTable({
      autoWidth: false,
      destroy: !0,
      columnDefs: [
        {
          orderable: !1,
          targets: "no-sort"
        }
      ]
    });
    $("#pagesTable").show();
  } else {
    pagesTable.clear();
    pagesTable.draw();
  }

  $("#loading").show(),
    api.pages
      .get(filter)
      .success(function(e) {
        (pages = e),
          $("#loading").hide(),
          pages.length > 0
            ? ($("#pagesTable").show(),
              $.each(pages, function(e, a) {
                pagesTable.row
                  .add([
                    escapeHtml(a.name),
                    a.username,
                    moment(a.modified_date).format("MMMM Do YYYY, h:mm:ss a"),
                    "<div class='pull-right'>" +
                      (a.writable
                        ? "<span data-toggle='modal' data-backdrop='static' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Edit Page' onclick='edit(" +
                          e +
                          ")'><i class='fa fa-pencil'></i></button></span>\t\t"
                        : "") +
                      "  <span data-toggle='modal' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Copy Page' onclick='copy(" +
                      e +
                      ")'><i class='fa fa-copy'></i></button></span>\t\t" +
                      "<span data-toggle='modal' data-target='#modalforpreview'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Preview Landing Page' onclick='preview(" +
                      e +
                      ")'><i class='fa fa-eye'></i></button></span> \t\t    " +
                      (a.writable
                        ? "<button class='btn btn-danger' data-toggle='tooltip' data-placement='left' title='Delete Page' onclick='deletePage(" +
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
        $("#loading").hide(), errorFlash("Error fetching pages");
      });
}
var pages = [],
  deletePage = function(e) {
    swal({
      title: "Are you sure?",
      text: "This will delete the landing page. This can't be undone!",
      type: "warning",
      animation: !1,
      showCancelButton: !0,
      confirmButtonText: "Delete " + escapeHtml(pages[e].name),
      confirmButtonColor: "#428bca",
      reverseButtons: !0,
      allowOutsideClick: !1,
      preConfirm: function() {
        return new Promise(function(a, t) {
          api.pageId
            .delete(pages[e].id)
            .success(function(e) {
              a();
            })
            .error(function(e) {
              t(e.responseJSON.message);
            });
        });
      }
    }).then(function() {
      swal(
        "Landing Page Deleted!",
        "This landing page has been deleted!",
        "success"
      ),
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
    $("input[type=radio][name=filter]").change(function(event) {
      load(event.target.value);
    });

  setTimeout(function() {
    api.phishtags.get().success(function(s) {
      var data = s.map(function(c) {
        return {
          id: c.id,
          text: c.name
        };
      });

      $("#category.form-control").select2({
        placeholder: "Select Category",
        data: data
      });
    });
  }, 1000);

  load(hasPages ? "own" : "public");

  $.fn.select2.defaults.set("width", "100%"),
    $.fn.select2.defaults.set("dropdownParent", $("#modal_body")),
    $.fn.select2.defaults.set("theme", "bootstrap");
});
