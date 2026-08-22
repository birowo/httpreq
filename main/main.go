package main

import (
	"fmt"

	"github.com/birowo/httpreq"
	"github.com/panjf2000/gnet/v2"
)

// httpServer mengimplementasikan gnet.EventHandler
type httpServer struct {
	gnet.BuiltinEventEngine
}

var (
	badReqRes = []byte(
		" 400 Bad Request\r\nConnection: close\r\n\r\n",
	)
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

	req, consumed, incomplete, err := httpreq.Parse(buf, headers, 1000000)

	// 3. Handle error format HTTP (Bad Request)
	if err != nil {
		println(err.Error())
		c.Write(req.Proto)
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
	// 5. Geser/buang buffer gnet yang sudah selesai diproses
	c.Discard(consumed)

	// Kirim response balik ke client
	resBody := []byte("hello world")
	resBodyLen := httpreq.BsInt(uint32(len(resBody)))
	resHdr := [][]byte{
		[]byte("Content-Type: text/html; charset=utf-8\r\n"),
		resBodyLen[:],
	}
	var res [512][]byte
	res[0] = []byte("HTTP/1.1 ")
	res[1] = httpreq.StatusOK
	n := httpreq.ResHdrs(resHdr, &res)
	res[n] = resBody
	fmt.Printf("%q\n", res[:n+1])
	c.AsyncWritev(res[:n+1], nil)

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
