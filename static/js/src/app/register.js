$.fn.deserialize = function(serializedString) {
  var $form = $(this);
  $form[0].reset();
  serializedString = serializedString.replace(/\+/g, "%20");
  var formFieldArray = serializedString.split("&");

  $.each(formFieldArray, function(i, pair) {
    var nameValue = pair.split("=");
    var name = decodeURIComponent(nameValue[0]);
    var value = decodeURIComponent(nameValue[1]);
    // Find one or more fields
    var $field = $form.find("[name=" + name + "]");

    if ($field[0].type == "radio" || $field[0].type == "checkbox") {
      var $fieldWithValue = $field.filter('[value="' + value + '"]');
      var isFound = $fieldWithValue.length > 0;
      if (!isFound && value == "on") {
        $field.first().prop("checked", true);
      } else {
        $fieldWithValue.prop("checked", isFound);
      }
    } else {
      $field.val(value);
    }
  });
};

$.fn.clear = function() {
  var form = $(this)[0];
  var elements = form.elements;
  form.reset();
  for (i = 0; i < elements.length; i++) {
    var field_type = elements[i].type.toLowerCase();
    switch (field_type) {
      case "text":
      case "password":
      case "textarea":
      case "hidden":
        elements[i].value = "";
        break;
      case "radio":
      case "checkbox":
        if (elements[i].checked) {
          elements[i].checked = false;
        }
        break;
      case "select-one":
      case "select-multi":
        elements[i].selectedIndex = -1;
        break;
      default:
        break;
    }
  }
  return this;
};

$(".form-signin").submit(function(event) {
  sessionStorage.setItem(
    "registration-form",
    $("form#registration").serialize()
  );
});

$(document).ready(function() {
  if ($(".alert").length) {
    const serializedForm = sessionStorage.getItem("registration-form");

    if (serializedForm !== null) {
      $("form#registration").deserialize(serializedForm);
    }
  }
});
