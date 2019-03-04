var Fixer = function (selector, minimum) {
  var self = this;

  self.selector = selector;
  self.minimum = minimum;
  self.offset = 0;
  self.stored = null;

  self.Handle = function () {
    var element = $(self.selector);
    if (!element) return;

    var pend = element.parent().offset().top + element.parent().height();
    var top = $(window).scrollTop();
    var offset = element.offset().top - top;

    if (offset < self.minimum && screen.width > 990 && ((pend - top) > (self.minimum + element.height()))) {
      if (self.stored == null)
        self.stored = top;
      element.css({"position": "fixed", "top": self.minimum});
    } else if (top < self.stored || screen.width <= 990) {
      element.removeAttr("style");
      self.stored = null;
    } else if (pend - top <= (element.height() + self.minimum)) {
      element.css({"z-index": "-100", "position": "fixed", "top": pend - top - element.height()});
    }
  } 

  $(window).scroll(self.Handle)
}

var Spier = function (index_selector, body_selector) {
  var self = this;

  self.Handle = function () {
    var top = $(window).scrollTop();
    var found = self.elements[0];

    self.elements.each(function (index, element) {
      var kOffset = 20;
      var distance = $(element).offset().top - top - kOffset;
      if (distance > 0) {
        return;
      }
      
      if (distance >= ($(found).offset().top - top - kOffset)) {
        found = element;
      }
    });

    if (found != null) {
      var id = "#" + $(found).attr("id");
      if (self.links[id] && self.selected &&
          self.links[id] == self.selected) {
        return;
      }

      if (self.selected) {
        $(self.selected).css({"font-weight": "normal"});
      }
      if (self.links[id]) {
        $(self.links[id]).css({"font-weight": "bolder"});
        self.selected = self.links[id];
      }
    }
  }

  $(document).ready(function () {
    self.elements = $(body_selector + " *[id]").sort(function (a, b) {
      return $(a).offset().top - $(b).offset().top;
    });

    self.links = {}
    $(index_selector + " a[href^='#']").each(function (index, element) {
      var href = $(element).attr("href");
      if (!self.links[href]) {
        self.links[href] = [];
      }
      self.links[href].push(element);
    });

    if (self.elements.length)
      $(window).scroll(self.Handle)
  });
}

var fixer = new Fixer("#sidebar", 75);
var spier = new Spier("#sidebar", "article.Prose");

$(document).ready(function () {
$("#loginsubmit").click(function () {
                        console.log("SUBMIT");
 $.ajax({
           type: "POST",
           url: $("#loginform").attr("action"),
           data: $("#loginform").serialize(), // serializes the form's elements.
           success: function(data)
           {
              console.log("DISABILITATO");
              $("#loginsubmit").prop("disabled", true);
              $("#loginemail").prop("disabled", true);
              $("#logintext").text("Ti abbiamo inviato un'email di verifica, segui le indicazioni fornite");

              console.log("SUCCESS");
           }
         });

    return false; // avoid to execute the actual submit of the form.
});
});
