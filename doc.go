// Package empire provides the core internal API to Empire. This provides a
// simple API for performing actions like creating applications, setting
// environment variables and performing deployments.
//
// Consumers of this API are usually in-process control layers, like the Heroku
// Platform API compatibility layer, and the GitHub Deployments integration,
// which can be found under the server package.
package empire
