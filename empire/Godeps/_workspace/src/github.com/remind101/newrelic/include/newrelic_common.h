/**
 * This is used by TransactionLib and CollectorClientLib
 */
#ifndef NEWRELIC_COMMON_H_
#define NEWRELIC_COMMON_H_

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif /* __cplusplus */

/**
 * Return codes
 */
static const int NEWRELIC_RETURN_CODE_OK = 0;
static const int NEWRELIC_RETURN_CODE_OTHER = -0x10001;
static const int NEWRELIC_RETURN_CODE_DISABLED = -0x20001;
static const int NEWRELIC_RETURN_CODE_INVALID_PARAM = -0x30001;
static const int NEWRELIC_RETURN_CODE_INVALID_ID = -0x30002;
static const int NEWRELIC_RETURN_CODE_TRANSACTION_NOT_STARTED = -0x40001;
static const int NEWRELIC_RETURN_CODE_TRANSACTION_IN_PROGRESS = -0x40002;
static const int NEWRELIC_RETURN_CODE_TRANSACTION_NOT_NAMED = -0x40003;

/*
 * A basic literal replacement obfuscator that strips the SQL string literals 
 * (values between single or double quotes) and numeric sequences, replacing 
 * them with the ? character.
 *
 * For example:
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
 * @param raw  a raw sql string
 * @return  obfuscated sql
 */
char *newrelic_basic_literal_replacement_obfuscator(const char *raw);

#ifdef __cplusplus
} //! extern "C"
#endif /* __cplusplus */

#endif /* NEWRELIC_COMMON_H_ */
