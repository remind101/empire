package xmlsec

// #include <stdio.h>
// #include <stdarg.h>
// #include <libxml/parser.h>
// #include <libxml/parserInternals.h>
// #include <libxml/xmlmemory.h>
// #include <xmlsec/xmlsec.h>
// #include <xmlsec/errors.h>
//
// void onXmlError(const char *msg);  // implemented in go
// void onXmlsecError(const char *file, int line, const char *funcName, const char *errorObject, const char *errorSubject, int reason, const char *msg);  // implemented in go
//
// static void onXmlGenericError_cgo(void *ctx, const char *format, ...) {
// 	char buffer[256];
// 	va_list args;
// 	va_start(args, format);
// 	vsnprintf(buffer, 256, format, args);
// 	va_end (args);
//  onXmlError(buffer);
// }
//
// static void onXmlsecError_cgo(const char *file, int line, const char *funcName, const char *errorObject, const char *errorSubject, int reason, const char *msg) {
// 	onXmlsecError(file, line, funcName, errorObject, errorSubject, reason, msg);
// }
//
// void captureXmlErrors() {
// 	xmlSecErrorsSetCallback(onXmlsecError_cgo);
// 	xmlSetGenericErrorFunc(NULL, onXmlGenericError_cgo);
// }
import "C"
