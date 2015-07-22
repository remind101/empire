(function(angular) {
  'use strict';

  var module = angular.module('app.filters', [
    'ng'
  ]);

  module.filter('ansi', function($window, $sce) {
    var ansi_up = $window.ansi_up,
        ansi_to_html = ansi_up.ansi_to_html,
        escape_for_html = ansi_up.escape_for_html;

    return function(input) {
      return $sce.trustAsHtml(ansi_to_html(escape_for_html(input)));
    };
  });

})(angular);
