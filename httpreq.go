package httpreq

import (
	"bytes"
	"errors"
)

const (
	rnrnStr         = "\r\n\r\n"
	rnrnLen         = len(rnrnStr)
	rn              = '\r'
	rnLen           = 2
	clKStr          = "\r\nContent-Length: "
	clKeyLen        = len(clKStr)
	hdrKVSparatr    = ':'
	hdrKVSparatrLen = 2
	maxHdrsNum      = 32
)

var (
	rnrn          = []byte(rnrnStr)
	ErrBadRequest = errors.New("bad request")
	clKey         = []byte(clKStr)
)

type KeyVal struct {
	K, V []byte
}

type Request struct {
	Method, Path, Proto []byte
	Headers             [maxHdrsNum]KeyVal
	HdrsNum             int
	//ContentLen          int
	Body []byte
}

// func Parse Http Request memproses buffer secara zero-alloc.
// Mengembalikan (bytesProcessed, incomplete, err).
// Jika incomplete == true, artinya data belum lengkap (incomplete), gnet harus menunggu data baru.
func Parse(buf []byte, req *Request) (int, bool, error) {
	// 1. Cari batas akhir seluruh hdrs (\r\n\r\n)
	hdrEnd := bytes.Index(buf, rnrn)
	if hdrEnd == -1 {
		return 0, true, nil // Incomplete data
	}
	reqLen := hdrEnd + rnrnLen
	hdrBuf := buf[:reqLen]

	// 2. Cari Content-Length (Pasti Title-Case karena dari cloudflared tunnel)
	clIdx := bytes.Index(hdrBuf, clKey)
	if clIdx != -1 {
		bgn := clIdx + clKeyLen
		cl := parseUintBytes(hdrBuf[bgn : bytes.IndexByte(hdrBuf[bgn:], rn)+bgn])

		// Pastikan seluruh Body sudah masuk di buffer gnet
		bgn = reqLen
		reqLen += cl
		if len(buf) < reqLen {
			return 0, true, nil // Incomplete data
		}
		if cl > 0 {
			//req.ContentLen = cl
			req.Body = buf[bgn:reqLen]
		}
	}

	// 3. Parsing Request Line

	// Method
	reqLineEnd := bytes.IndexByte(hdrBuf, rn)
	reqLine := hdrBuf[:reqLineEnd]
	sp1 := bytes.IndexByte(reqLine, ' ')
	if sp1 == -1 {
		return 0, false, ErrBadRequest
	}
	req.Method = reqLine[:sp1]

	// Path
	sp1++ //skip ' '
	sp2 := bytes.IndexByte(
		reqLine[sp1:], ' ',
	)
	if sp2 == -1 {
		return 0, false, ErrBadRequest
	}
	sp2 += sp1
	req.Path = reqLine[sp1:sp2]

	// Protocol
	sp2++ //skip ' '
	req.Proto = reqLine[sp2:]

	// 4. Parsing Seluruh Headers (Key otomatis Title-Case karena dari cloudflared tunnel)
	bgn := reqLineEnd + rnLen
	hdrIdx := 0
	for bgn < hdrEnd && hdrIdx < maxHdrsNum {
		end := bytes.IndexByte(hdrBuf[bgn:], rn) + bgn
		colonIdx := bytes.IndexByte(hdrBuf[bgn:end], hdrKVSparatr)
		if colonIdx != -1 {
			colonIdx += bgn
			req.Headers[hdrIdx] = KeyVal{
				hdrBuf[bgn:colonIdx],
				hdrBuf[colonIdx+hdrKVSparatrLen : end],
			}
			hdrIdx++
		}
		bgn = end + rnLen
	}
	req.HdrsNum = hdrIdx
	return reqLen, false, nil
}
func parseUintBytes(bs []byte) (val int) {
	for _, chr := range bs {
		if chr >= '0' || chr <= '9' {
			val = val*10 + int(chr-'0')
		}
	}
	return
}
