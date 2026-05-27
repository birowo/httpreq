package httpreq

import (
	"bytes"
	"errors"
)

const (
	rnrnStr    = "\r\n\r\n"
	rnrnLen    = len(rnrnStr)
	clKStr     = "\r\nContent-Length: "
	clKLen     = len(clKStr)
	maxHdrsNum = 32
)

var (
	rnrn          = []byte(rnrnStr)
	ErrBadRequest = errors.New("bad request")
	clK           = []byte(clKStr)
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

// ParseHttpRequest memproses buffer secara zero-alloc.
// Mengembalikan (bytesProcessed, incomplete, error).
// Jika incomplete == true, artinya data belum lengkap (incomplete), gnet harus menunggu data baru.
func Parse(buf []byte, req *Request) (int, bool, error) {
	// 1. Cari batas akhir seluruh hdrs (\r\n\r\n)
	hdrEnd := bytes.Index(buf, []byte(rnrn))
	if hdrEnd == -1 {
		return 0, true, nil // Incomplete data
	}

	totalHdrLen := hdrEnd + rnrnLen
	hdrBuf := buf[:totalHdrLen]

	// 2. Cari content-length (Pasti lowercase karena cloudflared)
	clKIdx := bytes.Index(hdrBuf, clK)
	var (
		cl     int
		reqLen int
	)
	if clKIdx != -1 {
		clVBgn := clKIdx + clKLen
		clVEnd := bytes.IndexByte(hdrBuf[clVBgn:], '\r') + clVBgn
		cl = parseUintBytes(hdrBuf[clVBgn:clVEnd])

		// Pastikan seluruh Body sudah masuk di buffer gnet
		if len(buf) < reqLen {
			return 0, true, nil // Incomplete data
		}
		if cl > 0 {
			reqLen = totalHdrLen + cl
			//req.ContentLen = cl
			req.Body = buf[totalHdrLen:reqLen]
		}
	} else {
		reqLen = totalHdrLen
	}

	// 3. Parsing Request Line

	// Method
	sp1 := bytes.IndexByte(hdrBuf, ' ')
	if sp1 == -1 {
		return 0, false, ErrBadRequest
	}
	req.Method = hdrBuf[:sp1]

	// Path
	sp1++
	sp2 := bytes.IndexByte(
		hdrBuf[sp1:], ' ',
	)
	if sp2 == -1 {
		return 0, false, ErrBadRequest
	}
	sp2 += sp1
	req.Path = hdrBuf[sp1:sp2]

	// Protocol
	sp2++
	reqLineEnd := bytes.IndexByte(hdrBuf[sp2:], '\r') + sp2
	req.Proto = hdrBuf[sp2:reqLineEnd]

	// 4. Parsing Seluruh Headers (Key otomatis TitleCase dari cloudflared)
	remainHdrs := hdrBuf[reqLineEnd+2:]
	hdrIdx := 0
	for remainHdrs[0] != '\r' && hdrIdx < maxHdrsNum {
		hdrEnd := bytes.IndexByte(remainHdrs, '\r')
		hdrKV := remainHdrs[:hdrEnd]
		colonIdx := bytes.IndexByte(hdrKV, ':')
		if colonIdx != -1 {
			req.Headers[hdrIdx] = KeyVal{
				K: hdrKV[:colonIdx],
				V: hdrKV[colonIdx+2:],
			}
			hdrIdx++
		}
		remainHdrs = remainHdrs[hdrEnd+2:]
	}
	req.HdrsNum = hdrIdx
	return reqLen, false, nil
}

func parseUintBytes(b []byte) (val int) {
	for _, c := range b {
		if c >= '0' || c <= '9' {
			val = val*10 + int(c-'0')
		}
	}
	return val
}
