package xmlsec

import (
	"fmt"
	"unsafe"
)

// #include <xmlsec/xmlsec.h>
// #include <xmlsec/xmltree.h>
// #include <xmlsec/xmlenc.h>
// #include <xmlsec/templates.h>
// #include <xmlsec/crypto.h>
import "C"

// #include <libxml/parser.h>
// #include <libxml/parserInternals.h>
// #include <libxml/xmlmemory.h>
import "C"

// Decrypt finds the first encrypted part of doc, decrypts it using
// privateKey and returns the plaintext of the embedded document.
func Decrypt(privateKey []byte, doc []byte) ([]byte, error) {
	startProcessingXML()
	defer stopProcessingXML()

	keysMngr := C.xmlSecKeysMngrCreate()
	if keysMngr == nil {
		return nil, popError()
	}
	defer C.xmlSecKeysMngrDestroy(keysMngr)

	if rv := C.xmlSecCryptoAppDefaultKeysMngrInit(keysMngr); rv < 0 {
		return nil, popError()
	}

	key := C.xmlSecCryptoAppKeyLoadMemory(
		(*C.xmlSecByte)(unsafe.Pointer(&privateKey[0])),
		C.xmlSecSize(len(privateKey)),
		C.xmlSecKeyDataFormatPem,
		nil, nil, nil)
	if key == nil {
		return nil, popError()
	}

	if rv := C.xmlSecCryptoAppDefaultKeysMngrAdoptKey(keysMngr, key); rv < 0 {
		return nil, popError()
	}

	parsedDoc, err := newDoc(doc, nil)
	if err != nil {
		return nil, err
	}
	defer closeDoc(parsedDoc)

	// create encryption context
	encCtx := C.xmlSecEncCtxCreate(keysMngr)
	if encCtx == nil {
		return nil, popError()
	}
	defer C.xmlSecEncCtxDestroy(encCtx)

	encDataNode := C.xmlSecFindNode(C.xmlDocGetRootElement(parsedDoc),
		(*C.xmlChar)(unsafe.Pointer(&C.xmlSecNodeEncryptedData)),
		(*C.xmlChar)(unsafe.Pointer(&C.xmlSecEncNs)))
	if encDataNode == nil {
		return nil, fmt.Errorf("xmlSecFindNode cannot find EncryptedData node")
	}

	// decrypt the data
	if rv := C.xmlSecEncCtxDecrypt(encCtx, encDataNode); rv < 0 {
		return nil, popError()
	}
	encDataNode = nil // the template is inserted in the doc, so we don't own it

	return dumpDoc(parsedDoc), nil
}
