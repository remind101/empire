package xmlsec

import (
	"errors"
	"unsafe"
)

// #include <xmlsec/xmlsec.h>
// #include <xmlsec/xmltree.h>
// #include <xmlsec/xmlenc.h>
// #include <xmlsec/xmldsig.h>
// #include <xmlsec/errors.h>
// #include <xmlsec/crypto.h>
import "C"

// SignatureOptions represents additional, less commonly used, options for Sign and
// Verify
type SignatureOptions struct {
	// Specify the name of ID attributes for specific elements. This
	// may be required if the signed document contains Reference elements
	// that define which parts of the document are to be signed.
	//
	// https://www.aleksey.com/xmlsec/faq.html#section_3_2
	// http://www.w3.org/TR/xml-id/
	// http://xmlsoft.org/html/libxml-valid.html#xmlAddID
	XMLID []XMLIDOption
}

// XMLIDOption represents the definition of an XML reference element
// (See http://www.w3.org/TR/xml-id/)
type XMLIDOption struct {
	ElementName      string
	ElementNamespace string
	AttributeName    string
}

// Sign returns a version of doc signed with key according to
// the XMLDSIG standard. doc is a template document meaning
// that it contains an `http://www.w3.org/2000/09/xmldsig#Signature`
// element whose properties define how and what to sign.
func Sign(key []byte, doc []byte, opts SignatureOptions) ([]byte, error) {
	startProcessingXML()
	defer stopProcessingXML()

	ctx := C.xmlSecDSigCtxCreate(nil)
	if ctx == nil {
		return nil, errors.New("failed to create signature context")
	}
	defer C.xmlSecDSigCtxDestroy(ctx)

	ctx.signKey = C.xmlSecCryptoAppKeyLoadMemory(
		(*C.xmlSecByte)(unsafe.Pointer(&key[0])),
		C.xmlSecSize(len(key)),
		C.xmlSecKeyDataFormatPem,
		nil, nil, nil)
	if ctx.signKey == nil {
		return nil, errors.New("failed to load pem key")
	}

	parsedDoc, err := newDoc(doc, opts.XMLID)
	if err != nil {
		return nil, err
	}
	defer closeDoc(parsedDoc)

	node := C.xmlSecFindNode(C.xmlDocGetRootElement(parsedDoc),
		(*C.xmlChar)(unsafe.Pointer(&C.xmlSecNodeSignature)),
		(*C.xmlChar)(unsafe.Pointer(&C.xmlSecDSigNs)))
	if node == nil {
		return nil, errors.New("cannot find start node")
	}

	if rv := C.xmlSecDSigCtxSign(ctx, node); rv < 0 {
		return nil, errors.New("failed to sign")
	}

	return dumpDoc(parsedDoc), nil

}

// ErrVerificationFailed is returned from Verify when the signature is incorrect
var ErrVerificationFailed = errors.New("signature verification failed")

// values returned from xmlSecDSigCtxVerify
const (
	xmlSecDSigStatusUnknown   = 0
	xmlSecDSigStatusSucceeded = 1
	xmlSecDSigStatusInvalid   = 2
)

// Verify checks that the signature in doc is valid according
// to the XMLDSIG specification. publicKey is the public part of
// the key used to sign doc. If the signature is not correct,
// this function returns ErrVerificationFailed.
func Verify(publicKey []byte, doc []byte, opts SignatureOptions) error {
	startProcessingXML()
	defer stopProcessingXML()

	keysMngr := C.xmlSecKeysMngrCreate()
	if keysMngr == nil {
		return mustPopError()
	}
	defer C.xmlSecKeysMngrDestroy(keysMngr)

	if rv := C.xmlSecCryptoAppDefaultKeysMngrInit(keysMngr); rv < 0 {
		return mustPopError()
	}

	key := C.xmlSecCryptoAppKeyLoadMemory(
		(*C.xmlSecByte)(unsafe.Pointer(&publicKey[0])),
		C.xmlSecSize(len(publicKey)),
		C.xmlSecKeyDataFormatCertPem,
		nil, nil, nil)
	if key == nil {
		return mustPopError()
	}

	if rv := C.xmlSecCryptoAppKeyCertLoadMemory(key,
		(*C.xmlSecByte)(unsafe.Pointer(&publicKey[0])),
		C.xmlSecSize(len(publicKey)),
		C.xmlSecKeyDataFormatCertPem); rv < 0 {
		C.xmlSecKeyDestroy(key)
		return mustPopError()
	}

	if rv := C.xmlSecCryptoAppDefaultKeysMngrAdoptKey(keysMngr, key); rv < 0 {
		return mustPopError()
	}

	dsigCtx := C.xmlSecDSigCtxCreate(keysMngr)
	if dsigCtx == nil {
		return mustPopError()
	}
	defer C.xmlSecDSigCtxDestroy(dsigCtx)

	parsedDoc, err := newDoc(doc, opts.XMLID)
	if err != nil {
		return err
	}
	defer closeDoc(parsedDoc)

	node := C.xmlSecFindNode(C.xmlDocGetRootElement(parsedDoc),
		(*C.xmlChar)(unsafe.Pointer(&C.xmlSecNodeSignature)),
		(*C.xmlChar)(unsafe.Pointer(&C.xmlSecDSigNs)))
	if node == nil {
		return errors.New("cannot find start node")
	}

	if rv := C.xmlSecDSigCtxVerify(dsigCtx, node); rv < 0 {
		return ErrVerificationFailed
	}

	if dsigCtx.status != xmlSecDSigStatusSucceeded {
		return ErrVerificationFailed
	}
	return nil
}
