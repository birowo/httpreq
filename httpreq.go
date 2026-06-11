package httpreq

import (
	"bytes"
	"errors"
)

const (
	rnrnStr       = "\r\n\r\n"
	rnrnLen       = len(rnrnStr)
	rn            = '\r'
	rnLen         = 2
	clKeyStr      = "\r\nContent-Length: "
	clKeyLen      = len(clKeyStr)
	hdrSparatr    = ':'
	hdrSparatrLen = 2
	hdrsMax       = 32
)

var (
	rnrn          = []byte(rnrnStr)
	ErrBadRequest = errors.New("bad request")
	clKey         = []byte(clKeyStr)
)

type kv struct {
	Key, Val []byte
}

type request struct {
	Method, Path, Proto []byte
	Headers             [hdrsMax]kv
	HdrsNum             int
	//ContentLen          int
	Body []byte
}

// func Parse Http Request memproses buffer secara zero-alloc.
// Mengembalikan (request, consumed, incomplete, error).
// Jika incomplete == true, artinya data belum lengkap (incomplete), gnet harus menunggu data baru.
func Parse(buf []byte) (req request, reqLen int, incomplete bool, err error) {
	// 1. Cari batas akhir seluruh hdrs (\r\n\r\n)
	hdrLen := bytes.Index(buf, rnrn)
	if hdrLen == -1 {
		incomplete = true
		return // Incomplete data
	}
	hdrEnd := hdrLen + rnrnLen
	reqLen = hdrEnd

	// 2. Cari Content-Length (Pasti Title-Case karena dari cloudflared tunnel)
	clIdx := bytes.Index(buf[:hdrLen], clKey)
	if clIdx != -1 {
		bgn := clIdx + clKeyLen
		cl := parseUintBytes(buf[bgn : bgn+bytes.IndexByte(buf[bgn:hdrEnd], rn)])

		// Pastikan seluruh Body sudah masuk di buffer gnet
		reqLen += cl
		if len(buf) < reqLen {
			incomplete = true
			return // Incomplete data
		}
		if cl > 0 {
			//req.ContentLen = cl
			req.Body = buf[hdrEnd:reqLen]
		}
	}

	// 3. Parsing Request Line

	// Method
	sp1 := bytes.IndexByte(buf[:hdrLen], ' ')
	if sp1 == -1 {
		err = ErrBadRequest
		return
	}
	req.Method = buf[:sp1]
	//println(string(buf[:sp1]))

	// Path
	sp1++ //skip ' '
	sp2 := bytes.IndexByte(buf[sp1:hdrLen], ' ')
	if sp2 == -1 {
		err = ErrBadRequest
		return
	}
	sp2 += sp1
	req.Path = buf[sp1:sp2]
	//println(string(buf[sp1:sp2]))

	// Protocol
	sp2++ //skip ' '
	reqLineEnd := bytes.IndexByte(buf[sp2:hdrEnd], rn) + sp2
	req.Proto = buf[sp2:reqLineEnd]
	//println(string(buf[sp2:reqLineEnd]))

	// 4. Parsing Seluruh Headers (Key otomatis Title-Case karena dari cloudflared tunnel)
	kBgn := reqLineEnd + rnLen
	hdrIdx := 0
	for kBgn < hdrLen {
		kEnd := bytes.IndexByte(buf[kBgn:hdrLen], hdrSparatr)
		if kEnd == -1 {
			err = ErrBadRequest
			return
		}
		kEnd += kBgn
		vBgn := kEnd + hdrSparatrLen
		vEnd := bytes.IndexByte(buf[vBgn:hdrEnd], rn) + vBgn
		//println("k:", string(buf[kBgn:kEnd]), ",v:", string(buf[vBgn:vEnd]))
		req.Headers[hdrIdx] = kv{
			Key: buf[kBgn:kEnd],
			Val: buf[vBgn:vEnd],
		}
		hdrIdx++
		if hdrIdx == hdrsMax {
			err = ErrBadRequest
			return
		}
		kBgn = vEnd + rnLen
	}
	req.HdrsNum = hdrIdx
	return
}
func parseUintBytes(bs []byte) (val int) {
	for _, chr := range bs {
		if chr >= '0' && chr <= '9' {
			val = val*10 + int(chr-'0')
		}
	}
	return
}
