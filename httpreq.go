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
)

var (
	rnrn          = []byte(rnrnStr)
	ErrBadRequest = errors.New("bad request")
	clKey         = []byte(clKeyStr)
)

type (
	KV struct {
		Key, Val []byte
	}
	slc struct {
		Bgn, End int
	}
	Request struct {
		Method, Path slc
		Query, Proto []byte
		Headers      []KV
		//ContentLen int
		Body []byte
	}
)

// func Parse Http Request memproses buffer secara zero-alloc.
// Mengembalikan (request, consumed, incomplete, error).
// Jika incomplete == true, artinya data belum lengkap (incomplete), gnet harus menunggu data baru.
func Parse(buf []byte, req *Request, bodyLenMax int) (reqLen int, incomplete bool, err error) {
	// 1. Cari batas akhir seluruh hdrs (\r\n\r\n)
	hdrLen := bytes.Index(buf, rnrn) + rnLen
	if hdrLen == (rnLen - 1) {
		incomplete = true
		return // Incomplete data
	}
	reqLen = hdrLen + rnLen
	if bodyLenMax > 0 {
		// 2. Cari Content-Length (Pasti Title-Case karena dari cloudflared tunnel)
		bgn := bytes.Index(buf[:hdrLen], clKey) + clKeyLen
		if bgn != (clKeyLen - 1) {

			//covert content length from string to int
			var cl int
			for _, chr := range buf[bgn : bgn+bytes.IndexByte(buf[bgn:hdrLen], rn)] {
				if cl < bodyLenMax && chr > ('0'-1) && chr < ('9'+1) {
					cl = (10 * cl) + int(chr-'0')
				} else {
					println("err1")
					err = ErrBadRequest
					return
				}
			}

			// Pastikan seluruh Body sudah masuk di buffer gnet
			reqLen += cl
			if len(buf) < reqLen {
				incomplete = true
				return // Incomplete data
			}

			//req.ContentLen = cl
			req.Body = buf[hdrLen+rnLen : reqLen]
		}
	}

	// 3. Parsing Request Line

	// Method
	sp1 := bytes.IndexByte(buf[:hdrLen], ' ')
	if sp1 == -1 {
		println("err2")
		err = ErrBadRequest
		return
	}
	req.Method = slc{0, sp1}
	//println("method:", string(buf[:sp1]))

	// Path & Query
	sp1++ //skip ' '
	idx := bytes.IndexByte(buf[sp1:hdrLen], ' ')
	if idx == -1 {
		println("err3")
		err = ErrBadRequest
		return
	}
	sp2 := sp1 + idx
	//println("path:", string(buf[sp1:sp2]))
	idx = sp1 + bytes.IndexByte(buf[sp1:sp2], '?')
	if idx != (sp1 - 1) {
		req.Path = slc{sp1, idx}
		req.Query = buf[idx+1 : sp2]
	} else {
		req.Path = slc{sp1, sp2}
	}

	// Protocol
	sp2++ //skip ' '
	reqLineEnd := bytes.IndexByte(buf[sp2:hdrLen], rn) + sp2
	req.Proto = buf[sp2:reqLineEnd]
	//println("proto:", string(buf[sp2:reqLineEnd]))

	// 4. Parsing Seluruh Headers (Key otomatis Title-Case karena dari cloudflared tunnel)
	kBgn := reqLineEnd + rnLen
	last := len(req.Headers) - 1
	for kBgn < hdrLen {
		kEnd := bytes.IndexByte(buf[kBgn:hdrLen], hdrSparatr)
		if kEnd == -1 {
			println("err4")
			err = ErrBadRequest
			return
		}
		kEnd += kBgn
		vBgn := kEnd + hdrSparatrLen
		vEnd := bytes.IndexByte(buf[vBgn:hdrLen], rn) + vBgn
		//println("k:", string(buf[kBgn:kEnd]), ",v:", string(buf[vBgn:vEnd]))
		for i, hdr := range req.Headers[:last+1] {
			if bytes.Equal(hdr.Key, buf[kBgn:kEnd]) {
				println(buf[kBgn:kEnd], ":", buf[vBgn:vEnd])
				req.Headers[i].Val = buf[vBgn:vEnd]
				req.Headers[i], req.Headers[last] = req.Headers[last], req.Headers[i]
				break
			}
		}
		last--
		if last == -1 {
			return
		}
		kBgn = vEnd + rnLen
	}
	return
}
