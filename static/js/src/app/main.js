jQuery(function($) {
    $('.mobile-hamburger').click(function(){
        $('.dashboard-box').toggleClass('slide-dashboard');
        $('body').toggleClass('slide-body');
        $('.overlap-full').toggleClass('toggle-overlap');
    });
     $('.hamburger-menu').click(function(){
        $('.dashboard-box').toggleClass('slide-dashboard');
        $('body').toggleClass('slide-body');
        $('.overlap-full').toggleClass('toggle-overlap');
    });
    $('.overlap-full').click(function(){
      $('.dashboard-box').toggleClass('slide-dashboard');
        $('body').toggleClass('slide-body');
        $('.overlap-full').toggleClass('toggle-overlap');
    });
    /*--------------------------------------------------------------
    #Sticky Header
    --------------------------------------------------------------*/
      /*Mobile Mebu*/
      $('#nav-icon').on('click', function(e) {
        e.preventDefault();
        $(this).toggleClass('menu-opened');
        $('.main-navigation').toggleClass('menu-open');
    });


    // tooltip
     // $('[data-toggle="tooltip"]').tooltip();
     // Initialize tooltip component
      $(function () {
        $('[data-toggle="tooltip"]').tooltip()
      })

      // Initialize popover component
      $(function () {
        $('[data-toggle="popover"]').popover()
      })

      // select

});
    /*--------------------------------------------------------------
    #svg icon
    --------------------------------------------------------------*/
//  jQuery('img.svg').each(function () {
//         var $img = jQuery(this);
//         var imgID = $img.attr('id');
//         var imgClass = $img.attr('class');
//         var imgURL = $img.attr('src');

//         jQuery.get(imgURL, function (data) {
//             // Get the SVG tag, ignore the rest
//             var $svg = jQuery(data).find('svg');

//             // Add replaced image's ID to the new SVG
//             if (typeof imgID !== 'undefined') {
//                 $svg = $svg.attr('id', imgID);
//             }
//             // Add replaced image's classes to the new SVG
//             if (typeof imgClass !== 'undefined') {
//                 $svg = $svg.attr('class', imgClass + ' replaced-svg');
//             }

//             // Remove any invalid XML tags as per http://validator.w3.org
//             $svg = $svg.removeAttr('xmlns:a');

//             // Check if the viewport is set, else we gonna set it if we can.
//             if (!$svg.attr('viewBox') && $svg.attr('height') && $svg.attr('width')) {
//                 $svg.attr('viewBox', '0 0 ' + $svg.attr('height') + ' ' + $svg.attr('width'))
//             }

//             // Replace image with new SVG
//             $img.replaceWith($svg);

//         }, 'xml');

//     });
$(document).ready(function() {
    $(".checkAll").change(function(){
      if(this.checked){
        $(".cb-element").each(function(){
          this.checked=true;
        })              
      }else{
        $(".cb-element").each(function(){
          this.checked=false;
        })              
      }
    });

    $(".cb-element").click(function () {
      if ($(this).is(":checked")){
        var isAllChecked = 0;
        $(".checkSingle").each(function(){
          if(!this.checked)
             isAllChecked = 1;
        })              
        if(isAllChecked == 0){ $(".checkAll").prop("checked", true); }     
      }else {
        $(".checkAll").prop("checked", false);
      }
    });
  });



// Inspiration: https://tympanus.net/codrops/2012/10/04/custom-drop-down-list-styling/

function DropDown(el) {
    this.dd = el;
    this.placeholder = this.dd.children('span');
    this.opts = this.dd.find('ul.drop li');
    this.val = '';
    this.index = -1;
    this.initEvents();
}

DropDown.prototype = {
    initEvents: function () {
        var obj = this;
        obj.dd.on('click', function (e) {
            e.preventDefault();
            e.stopPropagation();
            $(this).toggleClass('active');
        });
        obj.opts.on('click', function () {
            var opt = $(this);
            obj.val = opt.text();
            obj.index = opt.index();
            obj.placeholder.text(obj.val);
            opt.siblings().removeClass('selected');
            opt.filter(':contains("' + obj.val + '")').addClass('selected');
        }).change();
    },
    getValue: function () {
        return this.val;
    },
    getIndex: function () {
        return this.index;
    }
};

$(function () {
    // create new variable for each menu
    var dd1 = new DropDown($('#noble-gases'));
    // var dd2 = new DropDown($('#other-gases'));
    $(document).click(function () {
        // close menu on document click
        $('.wrap-drop').removeClass('active');
    });
});