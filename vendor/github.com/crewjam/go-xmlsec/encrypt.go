package xmlsec

// #include <xmlsec/xmlsec.h>
// #include <xmlsec/xmltree.h>
// #include <xmlsec/xmlenc.h>
// #include <xmlsec/templates.h>
// #include <xmlsec/crypto.h>
//
// // Note: the xmlSecKeyData*Id itentifiers are macros, so we need to wrap them
// // here to make them callable from go.
// static inline xmlSecKeyDataId MY_xmlSecKeyDataAesId(void) { return xmlSecKeyDataAesId; }
// static inline xmlSecKeyDataId MY_xmlSecKeyDataDesId(void) { return xmlSecKeyDataDesId; }
// static inline xmlSecTransformId MY_xmlSecTransformAes128CbcId(void) { return xmlSecTransformAes128CbcId; }
// static inline xmlSecTransformId MY_xmlSecTransformAes192CbcId(void) { return xmlSecTransformAes192CbcId; }
// static inline xmlSecTransformId MY_xmlSecTransformAes256CbcId(void) { return xmlSecTransformAes256CbcId; }
// static inline xmlSecTransformId MY_xmlSecTransformDes3CbcId(void) { return xmlSecTransformDes3CbcId; }
// static inline xmlSecTransformId MY_xmlSecTransformRsaOaepId(void) { return xmlSecTransformRsaOaepId; }
// static inline xmlSecTransformId MY_xmlSecTransformRsaPkcs1Id(void) { return xmlSecTransformRsaPkcs1Id; }
//
import "C"

import (
	"errors"
	"unsafe"
)

// SessionCipherType represents which session cipher to use to encrypt the document.
type SessionCipherType int

const (
	// DefaultSessionCipher (the zero value) represents the default session cipher, AES256-CBC
	DefaultSessionCipher SessionCipherType = iota

	// Aes128Cbc means the session cipher should be AES-128 in CBC mode.
	Aes128Cbc

	// Aes192Cbc means the session cipher should be AES-192 in CBC mode.
	Aes192Cbc

	// Aes256Cbc means the session cipher should be AES-256 in CBC mode.
	Aes256Cbc

	// Des3Cbc means the session cipher should be triple DES in CBC mode.
	Des3Cbc
)

// CipherType represent which cipher to use to encrypt the document
type CipherType int

const (
	// DefaultCipher (the zero value) represents the default cipher, RSA-OAEP
	DefaultCipher CipherType = iota

	// RsaOaep means the cipher should be RSA-OAEP
	RsaOaep

	// RsaPkcs1 means the cipher should be RSA-PKCS1
	RsaPkcs1
)

// DigestAlgorithmType represent which digest algorithm to use when encrypting the document.
type DigestAlgorithmType int

const (
	// DefaultDigestAlgorithm (the zero value) represents the default cipher, SHA1
	DefaultDigestAlgorithm DigestAlgorithmType = iota

	// Sha1 means the digest algorithm should be SHA-1
	Sha1

	// Sha256 means the digest algorithm should be SHA-256
	Sha256

	// Sha384 means the digest algorithm should be SHA-384
	Sha384

	// Sha512 means the digest algorithm should be SHA-512
	Sha512
)

// EncryptOptions specifies the ciphers to use to encrypt the document.
type EncryptOptions struct {
	SessionCipher   SessionCipherType
	Cipher          CipherType
	DigestAlgorithm DigestAlgorithmType
}

var errInvalidAlgorithm = errors.New("invalid algorithm")

// global string constants
// Note: the invocations of C.CString() here return a pointer to a string
// allocated from the C heap that would normally need to freed by calling
// C.free, but because these are global, we can just leak them.
var (
	constDsigNamespace = (*C.xmlChar)(unsafe.Pointer(C.CString("http://www.w3.org/2000/09/xmldsig#")))
	constDigestMethod  = (*C.xmlChar)(unsafe.Pointer(C.CString("DigestMethod")))
	constAlgorithm     = (*C.xmlChar)(unsafe.Pointer(C.CString("Algorithm")))
	constSha512        = (*C.xmlChar)(unsafe.Pointer(C.CString("http://www.w3.org/2001/04/xmlenc#sha512")))
	constSha384        = (*C.xmlChar)(unsafe.Pointer(C.CString("http://www.w3.org/2001/04/xmldsig-more#sha384")))
	constSha256        = (*C.xmlChar)(unsafe.Pointer(C.CString("http://www.w3.org/2001/04/xmlenc#sha256")))
	constSha1          = (*C.xmlChar)(unsafe.Pointer(C.CString("http://www.w3.org/2000/09/xmldsig#sha1")))
)

// Encrypt encrypts the XML document to publicKey and returns the encrypted
// document.
func Encrypt(publicKey, doc []byte, opts EncryptOptions) ([]byte, error) {
	startProcessingXML()
	defer stopProcessingXML()

	keysMngr := C.xmlSecKeysMngrCreate()
	if keysMngr == nil {
		return nil, mustPopError()
	}
	defer C.xmlSecKeysMngrDestroy(keysMngr)

	if rv := C.xmlSecCryptoAppDefaultKeysMngrInit(keysMngr); rv < 0 {
		return nil, mustPopError()
	}

	key := C.xmlSecCryptoAppKeyLoadMemory(
		(*C.xmlSecByte)(unsafe.Pointer(&publicKey[0])),
		C.xmlSecSize(len(publicKey)),
		C.xmlSecKeyDataFormatCertPem,
		nil, nil, nil)
	if key == nil {
		return nil, mustPopError()
	}

	if rv := C.xmlSecCryptoAppKeyCertLoadMemory(key,
		(*C.xmlSecByte)(unsafe.Pointer(&publicKey[0])),
		C.xmlSecSize(len(publicKey)),
		C.xmlSecKeyDataFormatCertPem); rv < 0 {
		C.xmlSecKeyDestroy(key)
		return nil, mustPopError()
	}

	if rv := C.xmlSecCryptoAppDefaultKeysMngrAdoptKey(keysMngr, key); rv < 0 {
		return nil, mustPopError()
	}

	parsedDoc, err := newDoc(doc, nil)
	if err != nil {
		return nil, err
	}
	defer closeDoc(parsedDoc)

	var sessionCipherTransform C.xmlSecTransformId
	switch opts.SessionCipher {
	case DefaultSessionCipher:
		sessionCipherTransform = C.MY_xmlSecTransformAes256CbcId()
	case Aes256Cbc:
		sessionCipherTransform = C.MY_xmlSecTransformAes256CbcId()
	case Aes192Cbc:
		sessionCipherTransform = C.MY_xmlSecTransformAes192CbcId()
	case Aes128Cbc:
		sessionCipherTransform = C.MY_xmlSecTransformAes128CbcId()
	case Des3Cbc:
		sessionCipherTransform = C.MY_xmlSecTransformDes3CbcId()
	default:
		return nil, errInvalidAlgorithm
	}

	// create encryption template to encrypt XML file and replace
	// its content with encryption result
	encDataNode := C.xmlSecTmplEncDataCreate(parsedDoc, sessionCipherTransform,
		nil, (*C.xmlChar)(unsafe.Pointer(&C.xmlSecTypeEncElement)), nil, nil)
	if encDataNode == nil {
		return nil, mustPopError()
	}
	defer func() {
		if encDataNode != nil {
			C.xmlFreeNode(encDataNode)
			encDataNode = nil
		}
	}()

	// we want to put encrypted data in the <enc:CipherValue/> node
	if C.xmlSecTmplEncDataEnsureCipherValue(encDataNode) == nil {
		return nil, mustPopError()
	}

	// add <dsig:KeyInfo/>
	keyInfoNode := C.xmlSecTmplEncDataEnsureKeyInfo(encDataNode, nil)
	if keyInfoNode == nil {
		return nil, mustPopError()
	}

	// add <enc:EncryptedKey/> to store the encrypted session key
	var cipherTransform C.xmlSecTransformId
	switch opts.Cipher {
	case DefaultCipher:
		cipherTransform = C.MY_xmlSecTransformRsaOaepId()
	case RsaOaep:
		cipherTransform = C.MY_xmlSecTransformRsaOaepId()
	case RsaPkcs1:
		cipherTransform = C.MY_xmlSecTransformRsaPkcs1Id()
	}
	encKeyNode := C.xmlSecTmplKeyInfoAddEncryptedKey(keyInfoNode, cipherTransform, nil, nil, nil)
	if encKeyNode == nil {
		return nil, mustPopError()
	}

	// we want to put encrypted key in the <enc:CipherValue/> node
	if C.xmlSecTmplEncDataEnsureCipherValue(encKeyNode) == nil {
		return nil, mustPopError()
	}

	// add <dsig:KeyInfo/> and <dsig:KeyName/> nodes to <enc:EncryptedKey/>
	keyInfoNode2 := C.xmlSecTmplEncDataEnsureKeyInfo(encKeyNode, nil)
	if keyInfoNode2 == nil {
		return nil, mustPopError()
	}

	// Add a DigestMethod element to the encryption method node
	{
		encKeyMethod := C.xmlSecTmplEncDataGetEncMethodNode(encKeyNode)
		var algorithm *C.xmlChar
		switch opts.DigestAlgorithm {
		case Sha512:
			algorithm = constSha512
		case Sha384:
			algorithm = constSha384
		case Sha256:
			algorithm = constSha256
		case Sha1:
			algorithm = constSha1
		case DefaultDigestAlgorithm:
			algorithm = constSha1
		default:
			return nil, errInvalidAlgorithm
		}
		node := C.xmlSecAddChild(encKeyMethod, constDigestMethod, constDsigNamespace)
		C.xmlSetProp(node, constAlgorithm, algorithm)
	}

	// add our certificate to KeyInfoNode
	x509dataNode := C.xmlSecTmplKeyInfoAddX509Data(keyInfoNode2)
	if x509dataNode == nil {
		return nil, mustPopError()
	}
	if dataNode := C.xmlSecTmplX509DataAddCertificate(x509dataNode); dataNode == nil {
		return nil, mustPopError()
	}

	// create encryption context
	var encCtx = C.xmlSecEncCtxCreate(keysMngr)
	if encCtx == nil {
		return nil, mustPopError()
	}
	defer C.xmlSecEncCtxDestroy(encCtx)

	// generate a key of the appropriate type
	switch opts.SessionCipher {
	case DefaultSessionCipher:
		encCtx.encKey = C.xmlSecKeyGenerate(C.MY_xmlSecKeyDataAesId(), 256,
			C.xmlSecKeyDataTypeSession)
	case Aes128Cbc:
		encCtx.encKey = C.xmlSecKeyGenerate(C.MY_xmlSecKeyDataAesId(), 128,
			C.xmlSecKeyDataTypeSession)
	case Aes192Cbc:
		encCtx.encKey = C.xmlSecKeyGenerate(C.MY_xmlSecKeyDataAesId(), 192,
			C.xmlSecKeyDataTypeSession)
	case Aes256Cbc:
		encCtx.encKey = C.xmlSecKeyGenerate(C.MY_xmlSecKeyDataAesId(), 256,
			C.xmlSecKeyDataTypeSession)
	case Des3Cbc:
		encCtx.encKey = C.xmlSecKeyGenerate(C.MY_xmlSecKeyDataDesId(), 192,
			C.xmlSecKeyDataTypeSession)
	default:
		return nil, errInvalidAlgorithm
	}
	if encCtx.encKey == nil {
		return nil, mustPopError()
	}

	// encrypt the data
	if rv := C.xmlSecEncCtxXmlEncrypt(encCtx, encDataNode, C.xmlDocGetRootElement(parsedDoc)); rv < 0 {
		return nil, mustPopError()
	}
	encDataNode = nil // the template is inserted in the doc, so we don't own it

	return dumpDoc(parsedDoc), nil
}
