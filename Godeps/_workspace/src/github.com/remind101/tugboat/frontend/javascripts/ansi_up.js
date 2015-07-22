// ansi_up.js
// version : 1.0.0
// author : Dru Nelson
// license : MIT
// http://github.com/drudru/ansi_up

(function (Date, undefined) {

    var ansi_up,
        VERSION = "1.0.0",

        // check for nodeJS
        hasModule = (typeof module !== 'undefined'),

        // Normal and then Bright
        ANSI_COLORS = [
          ["0,0,0", "187, 0, 0", "0, 187, 0", "187, 187, 0", "0, 0, 187", "187, 0, 187", "0, 187, 187", "255,255,255" ],
          ["85,85,85", "255, 85, 85", "0, 255, 0", "255, 255, 85", "85, 85, 255", "255, 85, 255", "85, 255, 255", "255,255,255" ]
        ];

    function Ansi_Up() {
      this.fg = this.bg = null;
      this.bright = 0;
    }

    Ansi_Up.prototype.escape_for_html = function (txt) {
      return txt.replace(/[&<>]/gm, function(str) {
        if (str == "&") return "&amp;";
        if (str == "<") return "&lt;";
        if (str == ">") return "&gt;";
      });
    };

    Ansi_Up.prototype.linkify = function (txt) {
      return txt.replace(/(https?:\/\/[^\s]+)/gm, function(str) {
        return "<a href=\"" + str + "\">" + str + "</a>";
      });
    };

    Ansi_Up.prototype.ansi_to_html = function (txt) {

      var data4 = txt.split(/\033\[/);

      var first = data4.shift(); // the first chunk is not the result of the split

      var self = this;
      var data5 = data4.map(function (chunk) {
        return self.process_chunk(chunk);
      });

      data5.unshift(first);

      var flattened_data = data5.reduce( function (a, b) {
        if (Array.isArray(b))
          return a.concat(b);

        a.push(b);
        return a;
      }, []);

      var escaped_data = flattened_data.join('');

      return escaped_data;
    };

    Ansi_Up.prototype.process_chunk = function (text) {

      // Do proper handling of sequences (aka - injest vi split(';') into state machine
      //match,codes,txt = text.match(/([\d;]+)m(.*)/m);
      var matches = text.match(/([\d;]+?)m([^]*)/m);

      if (!matches) return text;

      var orig_txt = matches[2];
      var nums = matches[1].split(';');

      var self = this;
      nums.map(function (num_str) {

        var num = parseInt(num_str);

        if (num === 0) {
          self.fg = self.bg = null;
          self.bright = 0;
        } else if (num === 1) {
          self.bright = 1;
        } else if ((num >= 30) && (num < 38)) {
          self.fg = "rgb(" + ANSI_COLORS[self.bright][(num % 10)] + ")";
        } else if (num === 39) {
          self.fg = null;
        } else if ((num >= 40) && (num < 48)) {
          self.bg = "rgb(" + ANSI_COLORS[0][(num % 10)] + ")";
        }
      });

      if ((self.fg === null) && (self.bg === null)) {
        return orig_txt;
      } else {
        var style = [];
        if (self.fg)
          style.push("color:" + self.fg);
        if (self.bg)
          style.push("background-color:" + self.bg);
        return ["<span style=\"" + style.join(';') + "\">", orig_txt, "</span>"];
      }
    };

    // Module exports
    ansi_up = {

      escape_for_html: function (txt) {
        var a2h = new Ansi_Up();
        return a2h.escape_for_html(txt);
      },

      linkify: function (txt) {
        var a2h = new Ansi_Up();
        return a2h.linkify(txt);
      },

      ansi_to_html: function (txt) {
        var a2h = new Ansi_Up();
        return a2h.ansi_to_html(txt);
      },

      ansi_to_html_obj: function () {
        return new Ansi_Up();
      }
    };

    // CommonJS module is defined
    if (hasModule) {
        module.exports = ansi_up;
    }
    /*global ender:false */
    if (typeof window !== 'undefined' && typeof ender === 'undefined') {
        window.ansi_up = ansi_up;
    }
    /*global define:false */
    if (typeof define === "function" && define.amd) {
        define("ansi_up", [], function () {
            return ansi_up;
        });
    }
})(Date);
