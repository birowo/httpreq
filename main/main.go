package main

import (
	"sync"

	"github.com/birowo/httpreq"
	"github.com/panjf2000/gnet/v2"
)

// httpServer mengimplementasikan gnet.EventHandler
type httpServer struct {
	gnet.BuiltinEventEngine
}

var (
	badReqRes = []byte(
		"HTTP/1.1 400 Bad Request\r\nConnection: close\r\n\r\n",
	)
	bufPool = sync.Pool{
		New: func() any {
			return make([]byte, 9999)
		},
	}
	pongRes = []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\npong\n")
)

func (hs *httpServer) OnTraffic(c gnet.Conn) gnet.Action {
	// 1. Peek seluruh buffer tanpa alokasi memori baru
	buf, _ := c.Peek(c.InboundBuffered())

	// 2. Siapkan struct Request di stack (zero-alloc)

	headers := []httpreq.KV{
		{Key: []byte("Host")},
		{Key: []byte("Date")},
		{Key: []byte("User-Agent")},
		{Key: []byte("Accept")},
		{Key: []byte("Content-Type")},
		{Key: []byte("Content-Length")},
	}
	resHdr := []httpreq.KV{
		{[]byte("Content-Type"), []byte("text/html; charset=utf-8")},
	}
	req, consumed, incomplete, err := httpreq.Parse(buf, headers, 1000000)

	// 3. Handle error format HTTP (Bad Request)
	if err != nil {
		println(err.Error())
		c.Write(badReqRes)
		return gnet.Close
	}

	// 4. Jika paket belum lengkap, tunggu data berikutnya di event OnTraffic selanjutnya
	if incomplete {
		println("incomplete")
		return gnet.None
	}

	// --- LOGIKA BISNIS (Eksekusi sebelum c.Discard) ---

	println("\nsebelum di-parse:\n", string(buf[:consumed]))
	println("\nsetelah di-parse:")
	mthd := req.Method
	path := req.Path
	println(
		"method:", string(buf[mthd.Bgn:mthd.End]),
		"\npath:", string(buf[path.Bgn:path.End]),
		"\nquery:", string(req.Query),
		"\nproto:", string(req.Proto),
	)
	for _, hdr := range headers {
		println(
			string(hdr.Key), ":", string(hdr.Val),
		)
	}
	if len(req.Body) != 0 {
		println("body:", string(req.Body))
	}

	// Kirim response balik ke client
	res := bufPool.Get().([]byte)
	n := httpreq.Res(req.Proto, httpreq.StatusOK, resHdr, res, []byte("hello world"))
	//println(string(res[:n]))
	c.Write(res[:n])
	bufPool.Put(res)

	// 5. Geser/buang buffer gnet yang sudah selesai diproses
	c.Discard(consumed)

	return gnet.None
}

func main() {
	// Menjalankan gnet server langsung ke localhost:8080 dengan multicore aktif
	err := gnet.Run(
		&httpServer{},
		"tcp://localhost:8080",
		gnet.WithMulticore(true),
		gnet.WithReusePort(true),
	)
	if err != nil {
		println(
			"Gagal menjalankan gnet server:", err.Error(),
		)
	}
}
