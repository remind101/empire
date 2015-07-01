/**
 * This is the C API for the Agent SDK's Collector Client Library.
 */
#ifndef NEWRELIC_COLLECTOR_CLIENT_H_
#define NEWRELIC_COLLECTOR_CLIENT_H_

#ifdef __cplusplus
extern "C" {
#endif /* __cplusplus */

/**
 * CollectorClient status codes
 */
static const int NEWRELIC_STATUS_CODE_SHUTDOWN = 0;
static const int NEWRELIC_STATUS_CODE_STARTING = 1;
static const int NEWRELIC_STATUS_CODE_STOPPING = 2;
static const int NEWRELIC_STATUS_CODE_STARTED = 3;

/**
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Embedded-mode only
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 *
 * Register this function to handle messages carrying application performance
 * data between the instrumented app and embedded CollectorClient. A daemon-mode
 * message handler is registered by default.
 *
 * If you register this handler using newrelic_register_message_handler
 * (declared in newrelic_transaction.h), messages will be passed directly
 * to the CollectorClient. Otherwise, the daemon-mode message handler will send
 * messages to the CollectorClient daemon via domain sockets.
 *
 * Note: Register newrelic_message_handler before calling newrelic_init.
 *
 * @param raw_message  message containing application performance data
 */
void *newrelic_message_handler(void *raw_message);

/**
 * Register a function to be called whenever the status of the CollectorClient
 * changes.
 *
 * @param callback  status callback function to register
 */
void newrelic_register_status_callback(void(*callback)(int));

/**
 * Start the CollectorClient and the harvester thread that sends application
 * performance data to New Relic once a minute.
 *
 * @param license  New Relic account license key
 * @param app_name  name of instrumented application
 * @param language  name of application programming language
 * @param language_version  application programming language version
 * @return  segment id on success, error code on error, else warning code
 */
int newrelic_init(const char *license, const char *app_name, const char *language, const char *language_version);

/**
 * Tell the CollectorClient to shutdown and stop reporting application
 * performance data to New Relic.
 *
 * @reason reason for shutdown request
 * @return  0 on success, error code on error, else warning code
 */
int newrelic_request_shutdown(const char *reason);

#ifdef __cplusplus
} //! extern "C"
#endif /* __cplusplus */

#endif /* NEWRELIC_COLLECTOR_CLIENT_H_ */
