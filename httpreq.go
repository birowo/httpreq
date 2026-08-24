package httpreq

import (
	"bytes"
	"errors"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/birowo/httpdateint64"
)

const (
	rnrnStr      = "\r\n\r\n"
	rn           = "\r\n"
	rnLen        = len(rn)
	clKeyStr     = "\r\nContent-Length: "
	clKeyLen     = len(clKeyStr)
	hdrSeparator = ": "
	hdrSepLen    = len(hdrSeparator)
)

var (
	r             = rn[0]
	hdrSep        = hdrSeparator[0]
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
		//ContentLen int
		Body []byte
	}
)

// func Parse Http Request memproses buffer secara zero-alloc.
// Mengembalikan (request, consumed, incomplete, error).
// Jika incomplete == true, artinya data belum lengkap (incomplete), gnet harus menunggu data baru.
func Parse(buf []byte, headers []KV, bodyLenMax uint) (req Request, reqLen int, incomplete bool, err error) {
	// 1. Cari batas akhir seluruh hdrs (\r\n\r\n)
	hdrLen := bytes.Index(buf, rnrn) + rnLen
	if hdrLen == (rnLen - 1) {
		incomplete = true
		return // Incomplete data
	}
	reqLen = hdrLen + rnLen
	if bodyLenMax != 0 {
		// 2. Cari Content-Length (Pasti Title-Case karena dari cloudflared tunnel)
		bgn := bytes.Index(buf[:hdrLen], clKey) + clKeyLen
		if bgn != (clKeyLen - 1) {

			//covert content length from string to int
			var cl uint
			for _, chr := range buf[bgn : bgn+bytes.IndexByte(buf[bgn:hdrLen], r)] {
				if cl < bodyLenMax && chr > ('0'-1) && chr < ('9'+1) {
					cl = (10 * cl) + uint(chr-'0')
				} else {
					println("err1")
					err = ErrBadRequest
					return
				}
			}

			// Pastikan seluruh Body sudah masuk di buffer gnet
			reqLen += int(cl)
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
	reqLineEnd := bytes.IndexByte(buf[sp2:hdrLen], r) + sp2
	req.Proto = buf[sp2:reqLineEnd]
	//println("proto:", string(buf[sp2:reqLineEnd]))

	// 4. Parsing Seluruh Headers (Key otomatis Title-Case karena dari cloudflared tunnel)
	kBgn := reqLineEnd + rnLen
	n := len(headers)
	for kBgn < hdrLen {
		kEnd := kBgn + bytes.IndexByte(buf[kBgn:hdrLen], hdrSep)
		if kEnd == kBgn-1 {
			println("err4")
			err = ErrBadRequest
			return
		}

		vBgn := kEnd + hdrSepLen
		vEnd := vBgn + bytes.IndexByte(buf[vBgn:hdrLen], r)
		//println("k:", string(buf[kBgn:kEnd]), ",v:", string(buf[vBgn:vEnd]))
		for i, hdr := range headers[:n] {
			if bytes.Equal(hdr.Key, buf[kBgn:kEnd]) {
				//println(string(buf[kBgn:kEnd]), ":", string(buf[vBgn:vEnd]))
				headers[i].Val = buf[vBgn:vEnd]
				n--
				if n == 0 {
					return
				}
				headers[i], headers[n] = headers[n], headers[i]
				break
			}
		}
		kBgn = vEnd + rnLen
	}
	return
}

const (
	cl    = "Content-Length: 0123456789\r\n"
	clLen = len(cl)
)

type Cl [clLen]byte

func BsInt(x uint32) (y [clLen]byte) {
	y = [clLen]byte{
		'C', 'o', 'n', 't', 'e', 'n', 't',
		'-',
		'L', 'e', 'n', 'g', 't', 'h', ':', ' ',
		' ', ' ', ' ', ' ', ' ',
		//....1....2....3....4
		' ', ' ', ' ', ' ', ' ', '\r', '\n',
		//....6....7....8....9
	}

	i := clLen - 2
	for x != 0 {
		i--
		y[i] = '0' + byte(x%10)
		x /= 10
	}
	return
}

const (
	hdKey  = "Date: "
	hdkLen = len(hdKey)
)

var (
	httpDate1 = []byte(
		hdKey + http.TimeFormat + rn,
	)
	httpDate2 = []byte(
		hdKey + http.TimeFormat + rn,
	)
	dateHdr atomic.Pointer[[]byte]
)

func init() {
	go func() {
		for {
			httpDate1, httpDate2 =
				httpDate2, httpDate1
			httpDate := httpdateint64.Conv(
				uint64(time.Now().Unix()),
			)
			copy(httpDate1[hdkLen:], httpDate[:])
			dateHdr.Store(&httpDate1)
			time.Sleep(time.Second)
		}
	}()
}

var (
	StatusOK = []byte("200 OK\r\n")
)

func ResHdrs(headers [][]byte, buf *[512][]byte) (i int) {
	buf[2] = *dateHdr.Load()
	//println(string(buf[:n]))
	var header []byte
	for i, header = range headers {
		buf[i+3] = header

	}
	buf[i+4] = []byte("\r\n")
	//println(string(buf[:n]))
	return i + 5
}
