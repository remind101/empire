var gulp    = require('gulp'),
    concat  = require('gulp-concat'),
    uglify  = require('gulp-uglify'),
    sass    = require('gulp-sass'),
    html2js = require('gulp-ng-html2js'),
    merge   = require('merge-stream');

var paths = {
  dest:        'dist',
  scripts:     'javascripts/**/*.js',
  stylesheets: 'stylesheets/**/*.scss',
  templates:   'templates/**/*.html'
};

gulp.task('sass', function() {
  return gulp.src(paths.stylesheets)
    .pipe(sass())
    .pipe(gulp.dest(paths.dest))
});

gulp.task('stylesheets', ['sass']);

gulp.task('javascripts', function() {
  var templates = gulp.src(paths.templates)
    .pipe(html2js({ moduleName: 'templates' }));

  var scripts = gulp.src(paths.scripts);

  return merge(scripts, templates)
    .pipe(concat('app.js'))
    .pipe(gulp.dest(paths.dest));
});

gulp.task('html', function() {
  return gulp.src('index.html')
    .pipe(gulp.dest(paths.dest));
});

gulp.task('watch', ['javascripts', 'stylesheets', 'html'], function() {
  gulp.watch(paths.scripts, ['javascripts']);
  gulp.watch(paths.templates, ['javascripts']);
  gulp.watch(paths.stylesheets, ['stylesheets']);
  gulp.watch('index.html', ['html']);
});

gulp.task('default', ['stylesheets', 'javascripts', 'html']);
