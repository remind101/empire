/*
 * This is the C API for the Agent SDK's Transaction Library
 *
 * The Transaction Library provides functions that are used to instrument
 * application transactions and the segment operations within transactions.
 */
#ifndef NEWRELIC_TRANSACTION_H_
#define NEWRELIC_TRANSACTION_H_

#ifdef __cplusplus
extern "C" {
#endif /* __cplusplus */

/*
 * NEWRELIC_AUTOSCOPE may be used in place of parent_segment_id to automatically
 * identify the last segment that was started within a transaction.
 *
 * In cases where a transaction runs uninterrupted from beginning to end within
 * the same thread, NEWRELIC_AUTOSCOPE may also be used in place of
 * transaction_id to automatically identify a transaction.
 */
static const long NEWRELIC_AUTOSCOPE = 1;

/*
 * NEWRELIC_ROOT_SEGMENT is used in place of parent_segment_id when a segment
 * does not have a parent.
 */
static const long NEWRELIC_ROOT_SEGMENT = 0;

/*
 * Datastore operations
 */
static const char * const NEWRELIC_DATASTORE_SELECT = "select";
static const char * const NEWRELIC_DATASTORE_INSERT = "insert";
static const char * const NEWRELIC_DATASTORE_UPDATE = "update";
static const char * const NEWRELIC_DATASTORE_DELETE = "delete";

/*
 * Disable/enable instrumentation. By default, instrumentation is enabled.
 *
 * All Transaction library functions used for instrumentation will immediately
 * return when you disable.
 *
 * @param set_enabled  0 to enable, 1 to disable
 */
void newrelic_enable_instrumentation(int set_enabled);

/*
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Embedded-mode only
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 *
 * Register a function to handle messages carrying application performance data
 * between the instrumented app and CollectorClient. By default, a daemon-mode
 * message handler is registered.
 *
 * If you register the embedded-mode message handler, newrelic_message_handler
 * (declared in newrelic_collector_client.h), messages will be passed directly
 * to the CollectorClient. Otherwise, the daemon-mode message handler will send
 * messages to the CollectorClient via domain sockets.
 *
 * Note: Register newrelic_message_handler before calling newrelic_init.
 *
 * @param handler  message handler for embedded-mode
 */
void newrelic_register_message_handler(void*(*handler)(void*));

/*
 * Record a custom metric.
 * 
 * @param   name  the name of the metric
 * @param   value   the value of the metric
 * @return  0 on success, else negative warning code or error code 
 */
int newrelic_record_metric(const char *name, double value);


/*
 * Record CPU user time in seconds and as a percentage of CPU capacity.
 *
 * @param cpu_user_time_seconds  number of seconds CPU spent processing user-level code
 * @param cpu_usage_percent  CPU user time as a percentage of CPU capacity
 * @return  0 on success, else negative warning code or error code
 */
int newrelic_record_cpu_usage(double cpu_user_time_seconds, double cpu_usage_percent);

/*
 * Record the current amount of memory being used.
 *
 * @param memory_megabytes  amount of memory currently being used
 * @return  0 on success, else negative warning code or error code
 */
int newrelic_record_memory_usage(double memory_megabytes);

/*
 * Identify the beginning of a transaction. By default, transaction type is set
 * to 'WebTransaction' and transaction category is set to 'Uri'. You can change
 * the transaction type using newrelic_transaction_set_type_other or 
 * newrelic_transaction_set_type_web. You can change the transaction category 
 * using newrelic_transaction_set_category.
 *
 * @return  transaction id on success, else negative warning code or error code
 */
long newrelic_transaction_begin();

/*
 * Set the transaction type to 'WebTransaction'. This will automatically change
 * the category to 'Uri'. You can change the transaction category using 
 * newrelic_transaction_set_category.
 *
 * @param transaction_id  id of transaction
 * @return  0 on success, else negative warning code or error code 
 */
int newrelic_transaction_set_type_web(long transaction_id);

/*
 * Set the transaction type to 'OtherTransaction'. This will automatically 
 * change the category to 'Custom'. You can change the transaction category 
 * using newrelic_transaction_set_category.
 *
 * @param transaction_id  id of transaction
 * @return  0 on success, else negative warning code or error code 
 */
int newrelic_transaction_set_type_other(long transaction_id);

/*
 * Set transaction category name, e.g. Uri in WebTransaction/Uri/<txn_name>
 *
 * @param transaction_id  id of transaction
 * @param category  name of the transaction category
 * @return  0 on success, else negative warning code or error code 
 */
int newrelic_transaction_set_category(long transaction_id, const char *category);

/*
 * Identify an error that occurred during the transaction. The first identified
 * error is sent with each transaction.
 *
 * @param transaction_id  id of transaction
 * @param exception_type  type of exception that occurred
 * @param error_message  error message
 * @param stack_trace  stacktrace when error occurred
 * @param stack_frame_delimiter  delimiter to split stack trace into frames
 * @return  0 on success, else negative warning code or error code
 */
int newrelic_transaction_notice_error(long transaction_id, const char *exception_type, const char *error_message, const char *stack_trace, const char *stack_frame_delimiter);


/*
 * Set a transaction attribute. Up to the first 50 attributes added are sent
 * with each transaction.
 *
 * @param transaction_id  id of transaction
 * @param name  attribute name
 * @param value  attribute value
 * @return  0 on success, else negative warning code or error code
 */
int newrelic_transaction_add_attribute(long transaction_id, const char *name, const char *value);

/*
 * Set the name of a transaction.
 *
 * @param transaction_id  id of transaction
 * @param name  transaction name
 * @return  0 on success, else negative warning code or error code
 */
int newrelic_transaction_set_name(long transaction_id, const char *name);

/*
 * Set the request url of a transaction. The query part of the url is
 * automatically stripped from the url.
 *
 * @param transaction_id  id of transaction
 * @param request_url  request url for a web transaction
 * @return  0 on success, else negative warning code or error code
 */
int newrelic_transaction_set_request_url(long transaction_id, const char *request_url);

/*
 * Set the maximum number of trace segments allowed in a transaction trace. By
 * default, the maximum is set to 2000, which means the first 2000 segments in a
 * transaction will create trace segments if the transaction exceeds the
 * trace threshold (4 x apdex_t).
 *
 * @param transaction_id  id of transaction
 * @param max_trace_segments  maximum number of trace segments
 * @return  0 on success, else negative warning code or error code
 */
int newrelic_transaction_set_max_trace_segments(long transaction_id, int max_trace_segments);

/*
 * Identify the end of a transaction
 *
 * @param transaction_id  id of transaction
 * @return  0 on success, else negative warning code or error code
 */
int newrelic_transaction_end(long transaction_id);

/*
 * Identify the beginning of a segment that performs a generic operation. This
 * type of segment does not create metrics, but can show up in a transaction
 * trace if a transaction is slow enough.
 *
 * @param transaction_id  id of transaction
 * @param parent_segment_id  id of parent segment
 * @param name  name to represent segment
 * @return  segment id on success, else negative warning code or error code
 */
long newrelic_segment_generic_begin(long transaction_id, long parent_segment_id, const char *name);

/*
 * Identify the beginning of a segment that performs a database operation.
 *
 * 
 * SQL Obfuscation
 * ===============
 * If you supply the sql_obfuscator parameter with NULL, the supplied SQL string
 * will go through our basic literal replacement obfuscator that strips the SQL 
 * string literals (values between single or double quotes) and numeric 
 * sequences, replacing them with the ? character. For example:
 * 
 * This SQL:
 * 		SELECT * FROM table WHERE ssn=‘000-00-0000’
 *		
 * obfuscates to:
 * 		SELECT * FROM table WHERE ssn=?
 *
 * Because our default obfuscator just replaces literals, there could be 
 * cases that it does not handle well. For instance, it will not strip out 
 * comments from your SQL string, it will not handle certain database-specific 
 * language features, and it could fail for other complex cases.
 *
 * If this level of obfuscation is not sufficient, you can supply your own 
 * custom obfuscator via the sql_obfuscator parameter.
 * 
 * SQL Trace Rollup
 * ================
 * The agent aggregates similar SQL statements together using the supplied
 * sql_trace_rollup_name.
 *
 * To make the most out of this feature, you should either (1) supply the 
 * sql_trace_rollup_name parameter with a name that describes what the SQL is
 * doing, such as "get_user_account" or (2) pass it NULL, in which case
 * it will use the sql obfuscator to generate a name.
 *
 * @param transaction_id  id of transaction
 * @param parent_segment_id  id of parent segment
 * @param table  name of the database table
 * @param operation  name of the sql operation
 * @param sql  the sql string
 * @param sql_trace_rollup_name  the rollup name for the sql trace
 * @param sql_obfuscator  a function pointer that takes sql and obfuscates it
 * @return  segment id on success, else negative warning code or error code
 */
long newrelic_segment_datastore_begin(
	long transaction_id, 
	long parent_segment_id, 
	const char *table, 
	const char *operation, 
	const char *sql, 
	const char *sql_trace_rollup_name, 
	char *(*sql_obfuscator)(const char *)
);

/*
 * Identify the beginning of a segment that performs an external service.
 *
 * @param transaction_id  id of transaction
 * @param parent_segment_id  id of parent segment
 * @param host  name of the host of the external call
 * @param name  name of the external transaction
 * @return  segment id on success, else negative warning code or error code
 */
long newrelic_segment_external_begin(long transaction_id, long parent_segment_id, const char *host, const char *name);

/*
 * Identify the end of a segment
 *
 * @param transaction_id  id of transaction
 * @param egment_id  id of the segment to end
 * @return  0 on success, else negative warning code or error code
 */
int newrelic_segment_end(long transaction_id, long segment_id);

#ifdef __cplusplus
} /* ! extern "C" */
#endif /* __cplusplus */

#endif /* NEWRELIC_TRANSACTION_H_ */
