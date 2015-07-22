(function(angular) {
  'use strict';

  var module = angular.module('app.services', [
    'ng',
    'ngResource'
  ]);

  /**
   * A pusher client service.
   */
  module.factory('pusher', function($window) {
    var apiKey = $window.$("meta[name='pusher.key']").attr('content');

    return new Pusher(apiKey);
  });

  module.factory('Job', function($resource) {
    var resource = $resource(
      '/jobs/:jobId',
      { jobId: '@id' }
    );

    function Job(attributes){
      this.setAttributes(attributes);
    }

    /**
     * Get a single job.
     */
    Job.find = function(id) {
      return resource.get({ jobId: id }).$promise.then(function(job) {
        return new Job(job);
      });
    };

    /**
     * Get all jobs.
     */
    Job.all = function() {
      return resource.query().$promise;
    };

    _.extend(Job.prototype, {
      /**
       * Set the attributes on this model.
       *
       * @param {Object} attributes
       */
      setAttributes: function(attributes) {
        var job = this;

        _.each(attributes, function(value, key) {
          job[key] = value;
        });
      },

      /**
       * Append some log output.
       *
       * @param {String} output
       */
      appendOutput: function(output) {
        this.output += output;
      },

      /**
       * Whether or not the job is queueud.
       *
       * @return {Boolean}
       */
      isQueued: function() {
        return this.status === 'pending';
      },

      /**
       * Whether or not the job has started to be worked on.
       *
       * @return {Boolean}
       */
      isStarted: function() {
        return this.status === 'started';
      },

      /**
       * Whether or not the job successfully deployed.
       *
       * @return {Boolean}
       */
      isDeployed: function() {
        return this.status === 'succeeded';
      },

      /**
       * Whether or not the job failed to deploy.
       *
       * @return {Boolean}
       */
      isFailed: function() {
        return this.status === 'failed';
      },

      /**
       * Whether or not the job errored.
       *
       * @return {Boolean}
       */
      isErrored: function() {
        return this.status === 'errored';
      }
    });

    return Job;
  });

  /**
   * A service to bind pusher events to a job.
   */
  module.factory('jobEvents', function(pusher) {
    var channels = {};

    function subscribe(scope, job) {
      var channel = channels[job.id] = channels[job.id] || pusher.subscribe('private-deployments-' + job.id);

      channel.bind('log_line', function(data) {
        scope.$apply(function() {
          job.appendOutput(data.output);
        });
      });

      channel.bind('status', function(data) {
        scope.$apply(function() {
          job.status = data.status;
        });
      });

      return job;
    };

    return {
      subscribe: subscribe
    };
  });

})(angular);
