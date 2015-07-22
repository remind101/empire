(function(angular) {
  'use strict';

  var module = angular.module('app.directives', [
    'ng'
  ]);

  /**
   * A directive for building a css3 spinner.
   */
  module.directive('spinner', function() {
    return {
      restrict: 'C',
      link: function(scope, elem) {
        function addRect(i) {
          elem.append('<div class="rect' + i + '"></div> ');
        }

        _(4).times(addRect);
      }
    };
  });

  module.directive('sticky', function($document, $window) {
    var padding = 100;

    return {
      restrict: 'A',
      link: function(scope, elem, attrs) {
        var $doc = $window.$($document),
            $win = $window.$($window);

        scope.$watch(attrs.sticky, function() {
          var sticky = $doc.scrollTop() + $win.height() >= $doc.height() - padding;

          if (sticky) {
            $doc.scrollTop($doc.height());
          }
        });
      }
    }
  });

  /**
   * A directive that for showing environment variables.
   */
  module.directive('environmentVariables', function($compile) {
    return {
      restrict: 'A',
      scope: { environmentVariables: '=' },
      link: function(scope, elem) {
        _.each(scope.environmentVariables, function(value, key) {
          elem.append($compile('<span environment-variable var="' + key + '" value="' + value + '" />')(scope));
        });
      }
    };
  });

  module.directive('environmentVariable', function() {
    return {
      restrict: 'EA',
      scope: { var: '@', value: '@' },
      template: '<div class="environment-variable"><span class="var" ng-bind="var"></span>=<span class="value" ng-bind="value"></span></div>'
    };
  });

})(angular);
