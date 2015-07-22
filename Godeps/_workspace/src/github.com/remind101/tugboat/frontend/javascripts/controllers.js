(function(angular) {
  'use strict';

  var module = angular.module('app.controllers', [
    'ng'
  ]);

  module.controller('JobsListCtrl', function($scope, jobs) {
    $scope.jobs = jobs;
  });

  module.controller('JobsDetailCtrl', function($scope, $state, job, jobEvents) {
    $scope.job = jobEvents.subscribe($scope, job);
  });

})(angular);
