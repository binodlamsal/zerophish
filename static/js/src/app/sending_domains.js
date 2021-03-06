function sendTestEmail() {
    var e = [];
    $.each($("#headersTable").DataTable().rows().data(), function(a, t) {
        e.push({
            key: unescapeHtml(t[0]),
            value: unescapeHtml(t[1])
        })
    });
    var a = {
        template: {},
        first_name: $("input[name=to_first_name]").val(),
        last_name: $("input[name=to_last_name]").val(),
        email: $("input[name=to_email]").val(),
        position: $("input[name=to_position]").val(),
        url: "",
        smtp: {
            from_address: $("#from").val(),
            host: $("#host").val(),
            username: $("#username").val(),
            password: $("#password").val(),
            ignore_cert_errors: $("#ignore_cert_errors").prop("checked"),
            headers: e
        }
    };
    btnHtml = $("#sendTestModalSubmit").html(), $("#sendTestModalSubmit").html('<i class="fa fa-spinner fa-spin"></i> Sending'), api.send_test_email(a).success(function(e) {
        $("#sendTestEmailModal\\.flashes").empty().append('<div style="text-align:center" class="alert alert-success">\t    <i class="fa fa-check-circle"></i> Email Sent!</div>'), $("#sendTestModalSubmit").html(btnHtml)
    }).error(function(e) {
        $("#sendTestEmailModal\\.flashes").empty().append('<div style="text-align:center" class="alert alert-danger">\t    <i class="fa fa-exclamation-circle"></i> ' + e.responseJSON.message + "</div>"), $("#sendTestModalSubmit").html(btnHtml)
    })
}

function save(e) {
    var a = {
        headers: []
    };
    $.each($("#headersTable").DataTable().rows().data(), function(e, t) {
        a.headers.push({
            key: unescapeHtml(t[0]),
            value: unescapeHtml(t[1])
        })
    }), a.name = $("#name").val(), a.interface_type = $("#interface_type").val(), a.from_address = $("#from").val(), a.host = $("#host").val(), a.username = $("#username").val(), a.password = $("#password").val(), a.ignore_cert_errors = $("#ignore_cert_errors").prop("checked"), -1 != e ? (a.id = profiles[e].id, api.SMTPId.put(a).success(function(e) {
        successFlash("Profile edited successfully!"), load(), dismiss()
    }).error(function(e) {
        modalError(e.responseJSON.message), scrollToError()
    })) : api.SMTP.post(a).success(function(e) {
        successFlash("Profile added successfully!"), load(), dismiss()
    }).error(function(e) {
        modalError(e.responseJSON.message), scrollToError()
    })
}

function dismiss() {
    $("#modal\\.flashes").empty(), $("#name").val(""), $("#interface_type").val("SMTP"), $("#from").val(""), $("#host").val(""), $("#username").val(""), $("#password").val(""), $("#ignore_cert_errors").prop("checked", !0), $("#headersTable").dataTable().DataTable().clear().draw(), $("#modal").modal("hide")
}

function edit(e) {
    headers = $("#headersTable").dataTable({
        destroy: !0,
        columnDefs: [{
            orderable: !1,
            targets: "no-sort"
        }]
    }), 
    $("#modal .modal-title").html("NEW SENDING DOMAIN"),
    $("#modalSubmit").unbind("click").click(function() {
        save(e)
    });
    var a = {}; - 1 != e && (a = profiles[e], $("#modal .modal-title").html("EDIT SENDING DOMAIN"), $("#name").val(a.name), $("#interface_type").val(a.interface_type), $("#from").val(a.from_address), $("#host").val(a.host), $("#username").val(a.username), $("#password").val(a.password), $("#ignore_cert_errors").prop("checked", a.ignore_cert_errors), $.each(a.headers, function(e, a) {
        addCustomHeader(a.key, a.value)
    }))
}

function copy(e) {
    $("#modalSubmit").unbind("click").click(function() {
        save(-1)
    });
    var a = {};
    a = profiles[e], $("#name").val("Copy of " + a.name), $("#interface_type").val(a.interface_type), $("#from").val(a.from_address), $("#host").val(a.host), $("#username").val(a.username), $("#password").val(a.password), $("#ignore_cert_errors").prop("checked", a.ignore_cert_errors)
}

function load() {
    $("#profileTable").hide(), $("#emptyMessage").hide(), $("#loading").show(), api.SMTP.domains().success(function(e) {
        profiles = e, $("#loading").hide(), profiles.length > 0 ? ($("#profileTable").show(), profileTable = $("#profileTable").DataTable({
            destroy: !0,
            columnDefs: [{
                orderable: !1,
                targets: "no-sort"
            }]
        }), profileTable.clear(), $.each(profiles, function(e, a) {
            profileTable.row.add([escapeHtml(a.name), a.interface_type, moment(a.modified_date).format("MMMM Do YYYY, h:mm:ss a"), "<div class='pull-right'><span data-toggle='modal' data-backdrop='static' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Edit Profile' onclick='edit(" + e + ")'>                    <i class='fa fa-pencil'></i>                    </button></span>\t\t    <span data-toggle='modal' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Copy Profile' onclick='copy(" + e + ")'>                    <i class='fa fa-copy'></i>                    </button></span>                    <button class='btn btn-danger' data-toggle='tooltip' data-placement='left' title='Delete Profile' onclick='deleteProfile(" + e + ")'>                    <i class='fa fa-trash-o'></i>                    </button></div>"]).draw()
        }), $('[data-toggle="tooltip"]').tooltip()) : $("#emptyMessage").show()
    }).error(function() {
        $("#loading").hide(), errorFlash("Error fetching profiles")
    })
}

function addCustomHeader(e, a) {
    var t = [escapeHtml(e), escapeHtml(a), '<span style="cursor:pointer;"><i class="fa fa-trash-o"></i></span>'],
        s = headers.DataTable(),
        o = s.column(0).data().indexOf(escapeHtml(e));
    o >= 0 ? s.row(o, {
        order: "index"
    }).data(t) : s.row.add(t), s.draw()
}
var profiles = [],
    dismissSendTestEmailModal = function() {
        $("#sendTestEmailModal\\.flashes").empty(), $("#sendTestModalSubmit").html("<i class='fa fa-envelope'></i> Send")
    },
    deleteProfile = function(e) {
        swal({
            title: "Are you sure?",
            text: "This will delete the sending profile. This can't be undone!",
            type: "warning",
            animation: !1,
            showCancelButton: !0,
            confirmButtonText: "Delete " + escapeHtml(profiles[e].name),
            confirmButtonColor: "#428bca",
            reverseButtons: !0,
            allowOutsideClick: !1,
            preConfirm: function() {
                return new Promise(function(a, t) {
                    api.SMTPId.delete(profiles[e].id).success(function(e) {
                        a()
                    }).error(function(e) {
                        t(e.responseJSON.message)
                    })
                })
            }
        }).then(function() {
            swal("Sending Profile Deleted!", "This sending profile has been deleted!", "success"), $('button:contains("OK")').on("click", function() {
                location.reload()
            })
        })
    };
$(document).ready(function() {
    $(".modal").on("hidden.bs.modal", function(e) {
        $(this).removeClass("fv-modal-stack"), $("body").data("fv_open_modals", $("body").data("fv_open_modals") - 1)
    }), $(".modal").on("shown.bs.modal", function(e) {
        void 0 === $("body").data("fv_open_modals") && $("body").data("fv_open_modals", 0), $(this).hasClass("fv-modal-stack") || ($(this).addClass("fv-modal-stack"), $("body").data("fv_open_modals", $("body").data("fv_open_modals") + 1), $(this).css("z-index", 1040 + 10 * $("body").data("fv_open_modals")), $(".modal-backdrop").not(".fv-modal-stack").css("z-index", 1039 + 10 * $("body").data("fv_open_modals")), $(".modal-backdrop").not("fv-modal-stack").addClass("fv-modal-stack"))
    }), $.fn.modal.Constructor.prototype.enforceFocus = function() {
        $(document).off("focusin.bs.modal").on("focusin.bs.modal", $.proxy(function(e) {
            this.$element[0] === e.target || this.$element.has(e.target).length || $(e.target).closest(".cke_dialog, .cke").length || this.$element.trigger("focus")
        }, this))
    }, $(document).on("hidden.bs.modal", ".modal", function() {
        $(".modal:visible").length && $(document.body).addClass("modal-open")
    }), $("#modal").on("hidden.bs.modal", function(e) {
        dismiss()
    }), $("#sendTestEmailModal").on("hidden.bs.modal", function(e) {
        dismissSendTestEmailModal()
    }), $("#headersForm").on("submit", function() {
        return headerKey = $("#headerKey").val(), headerValue = $("#headerValue").val(), "" != headerKey && "" != headerValue && (addCustomHeader(headerKey, headerValue), $("#headersForm>div>input").val(""), $("#headerKey").focus(), !1)
    }), $("#headersTable").on("click", "span>i.fa-trash-o", function() {
        headers.DataTable().row($(this).parents("tr")).remove().draw()
    }), load()
});